# ==================== Stage 1: Build ====================
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# ⚠️ 确认你的 main.go 入口路径，如果是根目录就改 ./，是 cmd/server 就保持不动
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /build/bin/server ./cmd/server

# ==================== Stage 2: Run ====================
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/alpine:3.19

RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apk del tzdata

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /build/bin/server ./server
COPY --chown=appuser:appgroup internal/config/env/prod.yaml ./internal/config/env/prod.yaml
COPY --chown=appuser:appgroup configs/ ./configs/

RUN mkdir -p logs && chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

ENV APP_ENV=prod

ENTRYPOINT ["./server"]