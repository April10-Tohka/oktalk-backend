# AI 发音纠正系统 (Pronunciation Correction System)

基于 Go 语言开发的 AI 发音纠正系统后端服务。

## 项目结构

```
pronunciation-correction-system/
├── cmd/                    # 应用入口点
├── internal/               # 私有包
│   ├── config/            # 配置管理
│   ├── handler/           # HTTP 处理层
│   ├── router/            # 路由层
│   ├── service/           # 业务逻辑层
│   ├── db/                # 数据库操作层
│   ├── model/             # 数据模型
│   ├── external/          # 第三方 API 封装
│   ├── cache/             # 缓存层
│   ├── queue/             # 异步任务队列
│   ├── pkg/               # 通用工具包
│   ├── database/          # 数据库迁移
│   └── constants/         # 全局常量
└── ...
```

## 技术栈

- **Web 框架**: Gin
- **ORM**: GORM
- **数据库**: MySQL
- **缓存**: Redis
- **日志**: Zap
- **配置**: Viper

## 第三方服务

- 科大讯飞：语音评测
- 阿里云 OSS：文件存储
- 阿里云 CosyVoice：TTS 语音合成
- 通义千问：AI 对话与反馈生成

## 快速开始

### 环境要求

- Go 1.21+
- MySQL 8.0+
- Redis 6.0+

### 安装

```bash
# 克隆项目
git clone <repository-url>
cd pronunciation-correction-system

# 安装依赖
make deps

# 复制环境配置
cp .env.example .env
# 编辑 .env 文件，填入实际配置

# 运行数据库迁移
make migrate

# 启动服务
make run
```

### 构建

```bash
make build
```

### 测试

```bash
make test
```

## API 文档

详见 `doc/02-api设计.md`

## 许可证

MIT License
