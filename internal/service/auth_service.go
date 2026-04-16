// Package service 提供认证业务逻辑
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"regexp"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"pronunciation-correction-system/internal/cache"
	"pronunciation-correction-system/internal/db"
	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/infrastructure/sms"
	"pronunciation-correction-system/internal/model"
	pkgjwt "pronunciation-correction-system/internal/pkg/jwt"
	pkguuid "pronunciation-correction-system/internal/pkg/uuid"
)

// ===================== 正则 =====================

var phoneRegexp = regexp.MustCompile(`^1[3-9]\d{9}$`)

// ===================== Redis 缓存结构 =====================

// smsCodeValue 短信验证码 Redis 存储结构
type smsCodeValue struct {
	Code         string `json:"code"`
	AttemptCount int    `json:"attempt_count"`
}

// ===================== 请求结构 =====================

// SendSMSRequest 发送短信验证码请求
type SendSMSRequest struct {
	Phone string `json:"phone" binding:"required"`
	IP    string `json:"-"` // 由 Handler 注入
}

// SMSLoginRequest 短信验证码登录请求
type SMSLoginRequest struct {
	Phone    string  `json:"phone" binding:"required"`
	Code     string  `json:"code" binding:"required"`
	Platform *string `json:"platform,omitempty"` // ios / android / web
	DeviceID *string `json:"device_id,omitempty"`
	IP       string  `json:"-"` // 由 Handler 注入
}

// WechatLoginRequest 微信登录请求
type WechatLoginRequest struct {
	AuthCode string  `json:"auth_code" binding:"required"`
	Platform *string `json:"platform,omitempty"`
	DeviceID *string `json:"device_id,omitempty"`
	IP       string  `json:"-"` // 由 Handler 注入
}

// RefreshTokenRequest 刷新 Token 请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// LogoutRequest 退出登录请求
type LogoutRequest struct {
	UserID string `json:"-"` // 由 middleware 注入
	JTI    string `json:"-"` // 由 middleware 注入
	IP     string `json:"-"` // 由 Handler 注入
}

// ===================== 响应结构 =====================

// SendSMSResponse 发送短信响应
type SendSMSResponse struct {
	ExpireSeconds     int `json:"expire_seconds"`
	RetryAfterSeconds int `json:"retry_after_seconds"`
}

// AuthLoginResponse 登录响应
type AuthLoginResponse struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	TokenType    string        `json:"token_type"`
	ExpiresIn    int           `json:"expires_in"`
	IsNewUser    bool          `json:"is_new_user"`
	WechatBound  bool          `json:"wechat_bound,omitempty"`
	User         *AuthUserInfo `json:"user"`
}

// AuthUserInfo 登录返回的用户信息
type AuthUserInfo struct {
	UserID    string  `json:"user_id"`
	Username  string  `json:"username"`
	Phone     string  `json:"phone,omitempty"` // 脱敏后
	AvatarURL *string `json:"avatar_url,omitempty"`
	Status    string  `json:"status"`
}

// TokenRefreshResponse Token 刷新响应
type TokenRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// ===================== Service 接口 =====================

// AuthService 认证业务接口
type AuthService interface {
	// SendSMS 发送短信验证码
	SendSMS(ctx context.Context, req *SendSMSRequest) (*SendSMSResponse, error)
	// SMSLogin 短信验证码登录/自动注册
	SMSLogin(ctx context.Context, req *SMSLoginRequest) (*AuthLoginResponse, error)
	// WechatLogin 微信登录/自动注册
	WechatLogin(ctx context.Context, req *WechatLoginRequest) (*AuthLoginResponse, error)
	// RefreshToken 刷新 Access Token
	RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*TokenRefreshResponse, error)
	// Logout 退出登录
	Logout(ctx context.Context, req *LogoutRequest) error
}

// ===================== 实现 =====================

type authServiceImpl struct {
	database     *gorm.DB
	rdb          *redis.Client
	repos        *db.Repositories
	smsClient    sms.Client
	wechatClient domain.WechatClient
	logger       *slog.Logger
}

// NewAuthService 创建 AuthService
func NewAuthService(
	database *gorm.DB,
	rdb *redis.Client,
	repos *db.Repositories,
	smsClient sms.Client,
	wechatClient domain.WechatClient,
	logger *slog.Logger,
) AuthService {
	return &authServiceImpl{
		database:     database,
		rdb:          rdb,
		repos:        repos,
		smsClient:    smsClient,
		wechatClient: wechatClient,
		logger:       logger,
	}
}

// ===================== 6.1 SendSMS 发送短信验证码 =====================

func (s *authServiceImpl) SendSMS(ctx context.Context, req *SendSMSRequest) (*SendSMSResponse, error) {
	// 1. 校验手机号格式
	if !phoneRegexp.MatchString(req.Phone) {
		return nil, &AuthError{Code: 400, Message: "手机号格式不正确"}
	}

	// 2. 速率限制 - 手机号维度
	intervalKey := fmt.Sprintf(cache.KeySMSInterval, req.Phone)
	ttlResult, err := s.rdb.TTL(ctx, intervalKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		s.logger.Error("redis ttl check failed", slog.String("error", err.Error()))
	}
	if ttlResult > 0 {
		return nil, &AuthError{
			Code:    429,
			Message: fmt.Sprintf("发送太频繁，请 %d 秒后重试", int(ttlResult.Seconds())),
		}
	}

	// 3. 速率限制 - IP 维度
	if req.IP != "" {
		ipCountKey := fmt.Sprintf(cache.KeySMSIPCount, req.IP)
		count, err := s.rdb.Incr(ctx, ipCountKey).Result()
		if err != nil {
			s.logger.Error("redis ip count incr failed", slog.String("error", err.Error()))
		} else {
			if count == 1 {
				// 首次写入时设置 TTL
				s.rdb.Expire(ctx, ipCountKey, cache.TTLSMSIPCount)
			}
			if count > 5 {
				return nil, &AuthError{Code: 429, Message: "操作过于频繁"}
			}
		}
	}

	// 4. 生成 6 位随机数字验证码
	code := fmt.Sprintf("%06d", rand.Intn(1000000))

	// 7. 发送短信
	result, err := s.smsClient.Send(req.Phone, code)
	if err != nil {
		s.logger.Error("sms send failed", slog.String("error", err.Error()))
		return nil, &AuthError{Code: 500, Message: "发送验证码失败"}
	}
	code = result.VerifyCode

	// 5. 存储验证码到 Redis
	codeKey := fmt.Sprintf(cache.KeySMSCode, req.Phone)
	codeVal := smsCodeValue{Code: code, AttemptCount: 0}
	codeJSON, _ := json.Marshal(codeVal)
	if err := s.rdb.Set(ctx, codeKey, codeJSON, cache.TTLSMSCode).Err(); err != nil {
		s.logger.Error("redis set sms code failed", slog.String("error", err.Error()))
		return nil, &AuthError{Code: 500, Message: "发送验证码失败"}
	}

	// 6. 设置间隔锁
	s.rdb.Set(ctx, intervalKey, "1", cache.TTLSMSInterval)

	// 8. 返回
	return &SendSMSResponse{
		ExpireSeconds:     300,
		RetryAfterSeconds: 0,
	}, nil
}

// ===================== 6.2 SMSLogin 短信验证码登录/自动注册 =====================

func (s *authServiceImpl) SMSLogin(ctx context.Context, req *SMSLoginRequest) (*AuthLoginResponse, error) {
	// 1. 校验参数
	if !phoneRegexp.MatchString(req.Phone) {
		return nil, &AuthError{Code: 400, Message: "手机号格式不正确"}
	}
	if len(req.Code) != 4 {
		return nil, &AuthError{Code: 400, Message: "验证码格式不正确"}
	}

	// 2. 速率限制：登录失败次数
	failKey := fmt.Sprintf(cache.KeyLoginFail, req.Phone)
	failCount, _ := s.rdb.Get(ctx, failKey).Int()
	if failCount >= 5 {
		return nil, &AuthError{Code: 429, Message: "登录失败次数过多，请10分钟后重试"}
	}

	// 3. 验证码校验
	codeKey := fmt.Sprintf(cache.KeySMSCode, req.Phone)
	codeJSON, err := s.rdb.Get(ctx, codeKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, &AuthError{Code: 400, Message: "验证码不存在或已过期"}
		}
		return nil, &AuthError{Code: 500, Message: "验证码校验失败"}
	}

	var codeVal smsCodeValue
	if err := json.Unmarshal(codeJSON, &codeVal); err != nil {
		return nil, &AuthError{Code: 500, Message: "验证码数据异常"}
	}

	if codeVal.AttemptCount >= 3 {
		return nil, &AuthError{Code: 400, Message: "验证码已失效，请重新发送"}
	}

	if codeVal.Code != req.Code {
		// 验证码不匹配，增加尝试次数
		codeVal.AttemptCount++
		updatedJSON, _ := json.Marshal(codeVal)
		// 保持原 TTL 写回
		ttl, _ := s.rdb.TTL(ctx, codeKey).Result()
		if ttl > 0 {
			s.rdb.Set(ctx, codeKey, updatedJSON, ttl)
		}
		// 增加登录失败计数
		count, _ := s.rdb.Incr(ctx, failKey).Result()
		if count == 1 {
			s.rdb.Expire(ctx, failKey, cache.TTLLoginFail)
		}
		remaining := 3 - codeVal.AttemptCount
		return nil, &AuthError{
			Code:    400,
			Message: fmt.Sprintf("验证码错误，还可尝试 %d 次", remaining),
		}
	}

	// 验证码匹配成功，删除验证码
	s.rdb.Del(ctx, codeKey)

	// 4. 查询用户
	isNewUser := false
	var user *model.User
	user, err = s.repos.User.GetByPhone(ctx, req.Phone)
	if err != nil {
		if !db.IsNotFound(err) {
			return nil, &AuthError{Code: 500, Message: "查询用户失败"}
		}

		// 新用户：事务中创建
		userID := pkguuid.New()
		phoneSuffix := req.Phone[7:]
		username := "用户" + phoneSuffix
		phone := req.Phone

		err = s.database.Transaction(func(tx *gorm.DB) error {
			txRepos := s.repos.WithTx(tx)
			avatarURL := "https://oktalk.oss-cn-heyuan.aliyuncs.com/assets/images/default_user_avatar.jpg"
			newUser := &model.User{
				ID:             userID,
				Username:       username,
				Phone:          &phone,
				Status:         "active",
				RegisterSource: "sms",
				AvatarURL:      &avatarURL,
			}
			if err := txRepos.User.Create(ctx, newUser); err != nil {
				return err
			}

			profile := &model.UserProfile{
				ID:     pkguuid.New(),
				UserID: userID,
			}
			if err := txRepos.UserProfile.Create(ctx, profile); err != nil {
				return err
			}

			user = newUser
			return nil
		})
		if err != nil {
			s.logger.Error("create new user failed", slog.String("error", err.Error()))
			return nil, &AuthError{Code: 500, Message: "创建用户失败"}
		}
		isNewUser = true
	} else {
		// 老用户：检查状态
		if user.Status != "active" {
			return nil, &AuthError{Code: 403, Message: "账户不可用，请联系客服"}
		}
	}

	// 5. 颁发 Token
	accessToken, _, err := pkgjwt.GenerateAccessToken(user.ID, user.Status)
	if err != nil {
		return nil, &AuthError{Code: 500, Message: "生成 Token 失败"}
	}
	refreshToken, _, err := pkgjwt.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, &AuthError{Code: 500, Message: "生成 Token 失败"}
	}

	// 存储 Refresh Token
	refreshKey := fmt.Sprintf(cache.KeyRefreshToken, user.ID)
	s.rdb.Set(ctx, refreshKey, refreshToken, cache.TTLRefreshToken)

	// 6. 异步写登录日志
	s.asyncWriteLoginLog(user.ID, "sms", "success", nil, req.IP, req.Platform, req.DeviceID)

	// 7. 手机号脱敏
	maskedPhone := maskPhone(req.Phone)

	// 8. 返回
	return &AuthLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    pkgjwt.GetAccessTTL(),
		IsNewUser:    isNewUser,
		User: &AuthUserInfo{
			UserID:    user.ID,
			Username:  user.Username,
			Phone:     maskedPhone,
			AvatarURL: user.AvatarURL,
			Status:    user.Status,
		},
	}, nil
}

// ===================== 6.3 WechatLogin 微信登录/自动注册 =====================

func (s *authServiceImpl) WechatLogin(ctx context.Context, req *WechatLoginRequest) (*AuthLoginResponse, error) {
	// 1. 校验 auth_code 不为空
	if req.AuthCode == "" {
		return nil, &AuthError{Code: 400, Message: "auth_code 不能为空"}
	}

	// 2. 调用微信 OAuth 接口
	oauthToken, err := s.wechatClient.GetOAuthToken(req.AuthCode)
	if err != nil {
		s.logger.Error("wechat oauth failed", slog.String("error", err.Error()))
		return nil, &AuthError{Code: 400, Message: "微信授权失败，请重试"}
	}

	// 3. 获取微信用户信息（失败不中断）
	var wxNickname, wxAvatarURL string
	userInfo, err := s.wechatClient.GetUserInfo(oauthToken.AccessToken, oauthToken.OpenID)
	if err != nil {
		s.logger.Warn("wechat get userinfo failed, using empty defaults", slog.String("error", err.Error()))
	} else {
		wxNickname = userInfo.Nickname
		wxAvatarURL = userInfo.HeadImgURL
	}

	// 4. 查询绑定关系
	isNewUser := false
	var user *model.User

	binding, err := s.repos.UserWechatBinding.FindByOpenID(ctx, oauthToken.OpenID)
	if err != nil && !db.IsNotFound(err) {
		return nil, &AuthError{Code: 500, Message: "查询绑定关系失败"}
	}

	if binding != nil {
		// 已有绑定：取 user
		user, err = s.repos.User.GetByID(ctx, binding.UserID)
		if err != nil {
			return nil, &AuthError{Code: 500, Message: "查询用户失败"}
		}
	} else {
		// 新用户：事务中创建
		userID := pkguuid.New()
		username := wxNickname
		if username == "" {
			username = fmt.Sprintf("微信用户%04d", rand.Intn(10000))
		}
		var avatarPtr *string
		if wxAvatarURL != "" {
			avatarPtr = &wxAvatarURL
		}
		var nicknamePtr *string
		if wxNickname != "" {
			nicknamePtr = &wxNickname
		}
		var avatarBindPtr *string
		if wxAvatarURL != "" {
			avatarBindPtr = &wxAvatarURL
		}

		err = s.database.Transaction(func(tx *gorm.DB) error {
			txRepos := s.repos.WithTx(tx)

			newUser := &model.User{
				ID:             userID,
				Username:       username,
				AvatarURL:      avatarPtr,
				Status:         "active",
				RegisterSource: "wechat",
			}
			if err := txRepos.User.Create(ctx, newUser); err != nil {
				return err
			}

			profile := &model.UserProfile{
				ID:     pkguuid.New(),
				UserID: userID,
			}
			if err := txRepos.UserProfile.Create(ctx, profile); err != nil {
				return err
			}

			wxBinding := &model.UserWechatBinding{
				ID:              pkguuid.New(),
				UserID:          userID,
				OpenID:          oauthToken.OpenID,
				WechatNickname:  nicknamePtr,
				WechatAvatarURL: avatarBindPtr,
			}
			if err := txRepos.UserWechatBinding.Create(ctx, wxBinding); err != nil {
				return err
			}

			user = newUser
			return nil
		})
		if err != nil {
			s.logger.Error("create wechat user failed", slog.String("error", err.Error()))
			return nil, &AuthError{Code: 500, Message: "创建用户失败"}
		}
		isNewUser = true
	}

	// 5. 检查用户状态
	if user.Status != "active" {
		return nil, &AuthError{Code: 403, Message: "账户不可用，请联系客服"}
	}

	// 6. 颁发 Token
	accessToken, _, err := pkgjwt.GenerateAccessToken(user.ID, user.Status)
	if err != nil {
		return nil, &AuthError{Code: 500, Message: "生成 Token 失败"}
	}
	refreshToken, _, err := pkgjwt.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, &AuthError{Code: 500, Message: "生成 Token 失败"}
	}

	// 存储 Refresh Token
	refreshKey := fmt.Sprintf(cache.KeyRefreshToken, user.ID)
	s.rdb.Set(ctx, refreshKey, refreshToken, cache.TTLRefreshToken)

	// 7. 异步写登录日志
	s.asyncWriteLoginLog(user.ID, "wechat", "success", nil, req.IP, req.Platform, req.DeviceID)

	// 8. 返回
	return &AuthLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    pkgjwt.GetAccessTTL(),
		IsNewUser:    isNewUser,
		WechatBound:  true,
		User: &AuthUserInfo{
			UserID:    user.ID,
			Username:  user.Username,
			AvatarURL: user.AvatarURL,
			Status:    user.Status,
		},
	}, nil
}

// ===================== 6.4 RefreshToken 刷新 Access Token =====================

func (s *authServiceImpl) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*TokenRefreshResponse, error) {
	// 1. 校验 refresh_token 不为空
	if req.RefreshToken == "" {
		return nil, &AuthError{Code: 401, Message: "请重新登录"}
	}

	// 2. 解析 Refresh Token
	claims, err := pkgjwt.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, &AuthError{Code: 401, Message: "请重新登录"}
	}

	// 3. 检查黑名单
	blacklistKey := fmt.Sprintf(cache.KeyTokenBlacklist, claims.JTI)
	exists, err := s.rdb.Exists(ctx, blacklistKey).Result()
	if err != nil {
		s.logger.Error("redis blacklist check failed", slog.String("error", err.Error()))
	}
	if exists > 0 {
		return nil, &AuthError{Code: 401, Message: "令牌已失效"}
	}

	// 4. 与 Redis 存储的 Refresh Token 比对
	refreshKey := fmt.Sprintf(cache.KeyRefreshToken, claims.UserID)
	storedToken, err := s.rdb.Get(ctx, refreshKey).Result()
	if err != nil || storedToken != req.RefreshToken {
		return nil, &AuthError{Code: 401, Message: "令牌已失效"}
	}

	// 5. 查询用户状态
	user, err := s.repos.User.GetByID(ctx, claims.UserID)
	if err != nil || user.Status != "active" {
		return nil, &AuthError{Code: 401, Message: "请重新登录"}
	}

	// 6. 旧 refresh_token 加入黑名单
	oldTTL := pkgjwt.GetRefreshRemainingTTL(claims)
	if oldTTL > 0 {
		s.rdb.Set(ctx, blacklistKey, "1", oldTTL)
	}

	// 7. 颁发新 Token 对
	newAccessToken, _, err := pkgjwt.GenerateAccessToken(user.ID, user.Status)
	if err != nil {
		return nil, &AuthError{Code: 500, Message: "生成 Token 失败"}
	}
	newRefreshToken, _, err := pkgjwt.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, &AuthError{Code: 500, Message: "生成 Token 失败"}
	}

	// 覆盖旧的 Refresh Token
	s.rdb.Set(ctx, refreshKey, newRefreshToken, cache.TTLRefreshToken)

	// 8. 异步写登录日志
	s.asyncWriteLoginLog(user.ID, "token_refresh", "success", nil, "", nil, nil)

	// 9. 返回
	return &TokenRefreshResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    pkgjwt.GetAccessTTL(),
	}, nil
}

// ===================== 6.5 Logout 退出登录 =====================

func (s *authServiceImpl) Logout(ctx context.Context, req *LogoutRequest) error {
	// 1. 从 Context 取 jti, user_id（由 Handler 注入）
	if req.UserID == "" || req.JTI == "" {
		return &AuthError{Code: 401, Message: "请重新登录"}
	}

	// 2. 计算 Access Token 剩余 TTL（从 jti 来的 claims 不好直接算，用配置的 AccessTTL 作为上限）
	// 这里直接使用 AccessTTL 的值作为黑名单 TTL 上限
	blacklistTTL := time.Duration(pkgjwt.GetAccessTTL()) * time.Second

	// 3. 写黑名单
	blacklistKey := fmt.Sprintf(cache.KeyTokenBlacklist, req.JTI)
	s.rdb.Set(ctx, blacklistKey, "1", blacklistTTL)

	// 4. 删除 Refresh Token
	refreshKey := fmt.Sprintf(cache.KeyRefreshToken, req.UserID)
	s.rdb.Del(ctx, refreshKey)

	// 5. 删除用户缓存
	profileKey := fmt.Sprintf(cache.KeyUserProfile, req.UserID)
	s.rdb.Del(ctx, profileKey)

	// 6. 异步写登录日志
	s.asyncWriteLoginLog(req.UserID, "logout", "success", nil, req.IP, nil, nil)

	return nil
}

// ===================== 辅助方法 =====================

// asyncWriteLoginLog 异步写登录日志（goroutine + recover）
func (s *authServiceImpl) asyncWriteLoginLog(
	userID, loginType, result string,
	failReason *string,
	ip string,
	platform *string,
	deviceID *string,
) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("asyncWriteLoginLog panic recovered", slog.Any("panic", r))
			}
		}()

		logEntry := &model.UserLoginLog{
			ID:         pkguuid.New(),
			UserID:     userID,
			LoginType:  loginType,
			Result:     result,
			FailReason: failReason,
			IP:         ip,
			Platform:   platform,
			DeviceID:   deviceID,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.repos.UserLoginLog.Insert(ctx, logEntry); err != nil {
			s.logger.Error("write login log failed",
				slog.String("user_id", userID),
				slog.String("login_type", loginType),
				slog.String("error", err.Error()),
			)
		}
	}()
}

// maskPhone 手机号脱敏: 13812345678 → 138****5678
func maskPhone(phone string) string {
	if len(phone) < 11 {
		return phone
	}
	return phone[:3] + "****" + phone[7:]
}

// ===================== 自定义错误 =====================

// AuthError 认证业务错误
type AuthError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *AuthError) Error() string {
	return e.Message
}
