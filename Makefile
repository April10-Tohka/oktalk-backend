# Makefile for pronunciation-correction-system

.PHONY: all build run test clean migrate help

# 默认目标
all: build

# 构建主程序
build:
	@echo "Building server..."
	go build -o bin/server ./cmd/server

# 运行主程序
run:
	@echo "Running server..."
	go run ./cmd/server/main.go

# 运行测试
test:
	@echo "Running tests..."
	go test -v ./...

# 运行数据库迁移
migrate:
	@echo "Running migrations..."
	go run ./cmd/migration/main.go

# 清理构建产物
clean:
	@echo "Cleaning..."
	rm -rf bin/

# 下载依赖
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# 代码格式化
fmt:
	@echo "Formatting code..."
	go fmt ./...

# 代码检查
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# 生成 mock 文件
mock:
	@echo "Generating mocks..."
	go generate ./...

# 帮助信息
help:
	@echo "Available targets:"
	@echo "  build    - Build the server binary"
	@echo "  run      - Run the server"
	@echo "  test     - Run tests"
	@echo "  migrate  - Run database migrations"
	@echo "  clean    - Clean build artifacts"
	@echo "  deps     - Download dependencies"
	@echo "  fmt      - Format code"
	@echo "  lint     - Run linter"
	@echo "  mock     - Generate mock files"
	@echo "  help     - Show this help"
