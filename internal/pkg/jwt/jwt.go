// Package jwt 提供 JWT Token 生成和解析工具
package jwt

import (
	"errors"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"pronunciation-correction-system/internal/config"
	pkguuid "pronunciation-correction-system/internal/pkg/uuid"
)

// ===================== Claims 结构体 =====================

// AccessClaims Access Token 自定义 Claims
type AccessClaims struct {
	jwtlib.RegisteredClaims
	UserID string `json:"user_id"`
	Status string `json:"status"`
	JTI    string `json:"jti"`
}

// RefreshClaims Refresh Token 自定义 Claims
type RefreshClaims struct {
	jwtlib.RegisteredClaims
	UserID    string `json:"user_id"`
	JTI       string `json:"jti"`
	TokenType string `json:"type"`
}

// ===================== 预定义错误 =====================

var (
	ErrTokenInvalid     = errors.New("token is invalid")
	ErrTokenExpired     = errors.New("token has expired")
	ErrTokenTypeMismatch = errors.New("token type mismatch")
	ErrEmptySecret      = errors.New("jwt secret is empty")
)

// ===================== 全局配置 =====================

var jwtCfg *config.JWTConfig

// Init 初始化 JWT 配置（应用启动时调用一次）
func Init(cfg *config.JWTConfig) {
	jwtCfg = cfg
}

// ===================== Token 生成 =====================

// GenerateAccessToken 生成 Access Token
// 返回: tokenString, jti, error
func GenerateAccessToken(userID, status string) (string, string, error) {
	if jwtCfg == nil || jwtCfg.AccessSecret == "" {
		return "", "", ErrEmptySecret
	}

	jti := pkguuid.New()
	now := time.Now()
	ttl := time.Duration(jwtCfg.AccessTTL) * time.Second

	claims := AccessClaims{
		RegisteredClaims: jwtlib.RegisteredClaims{
			ExpiresAt: jwtlib.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwtlib.NewNumericDate(now),
			ID:        jti,
		},
		UserID: userID,
		Status: status,
		JTI:    jti,
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtCfg.AccessSecret))
	if err != nil {
		return "", "", err
	}
	return tokenString, jti, nil
}

// GenerateRefreshToken 生成 Refresh Token
// 返回: tokenString, jti, error
func GenerateRefreshToken(userID string) (string, string, error) {
	if jwtCfg == nil || jwtCfg.RefreshSecret == "" {
		return "", "", ErrEmptySecret
	}

	jti := pkguuid.New()
	now := time.Now()
	ttl := time.Duration(jwtCfg.RefreshTTL) * time.Second

	claims := RefreshClaims{
		RegisteredClaims: jwtlib.RegisteredClaims{
			ExpiresAt: jwtlib.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwtlib.NewNumericDate(now),
			ID:        jti,
		},
		UserID:    userID,
		JTI:       jti,
		TokenType: "refresh",
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtCfg.RefreshSecret))
	if err != nil {
		return "", "", err
	}
	return tokenString, jti, nil
}

// ===================== Token 解析 =====================

// ParseAccessToken 解析并验证 Access Token
func ParseAccessToken(tokenString string) (*AccessClaims, error) {
	if jwtCfg == nil || jwtCfg.AccessSecret == "" {
		return nil, ErrEmptySecret
	}

	claims := &AccessClaims{}
	token, err := jwtlib.ParseWithClaims(tokenString, claims, func(t *jwtlib.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return []byte(jwtCfg.AccessSecret), nil
	})
	if err != nil {
		if errors.Is(err, jwtlib.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	if !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// ParseRefreshToken 解析并验证 Refresh Token
func ParseRefreshToken(tokenString string) (*RefreshClaims, error) {
	if jwtCfg == nil || jwtCfg.RefreshSecret == "" {
		return nil, ErrEmptySecret
	}

	claims := &RefreshClaims{}
	token, err := jwtlib.ParseWithClaims(tokenString, claims, func(t *jwtlib.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return []byte(jwtCfg.RefreshSecret), nil
	})
	if err != nil {
		if errors.Is(err, jwtlib.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	if !token.Valid {
		return nil, ErrTokenInvalid
	}

	// 验证 token type
	if claims.TokenType != "refresh" {
		return nil, ErrTokenTypeMismatch
	}

	return claims, nil
}

// ===================== 工具函数 =====================

// GetRemainingTTL 获取 Access Token 距离过期的剩余时长
func GetRemainingTTL(claims *AccessClaims) time.Duration {
	if claims == nil || claims.ExpiresAt == nil {
		return 0
	}
	remaining := time.Until(claims.ExpiresAt.Time)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetRefreshRemainingTTL 获取 Refresh Token 距离过期的剩余时长
func GetRefreshRemainingTTL(claims *RefreshClaims) time.Duration {
	if claims == nil || claims.ExpiresAt == nil {
		return 0
	}
	remaining := time.Until(claims.ExpiresAt.Time)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetAccessTTL 获取配置中的 Access Token TTL（秒）
func GetAccessTTL() int {
	if jwtCfg == nil {
		return 7200
	}
	return jwtCfg.AccessTTL
}
