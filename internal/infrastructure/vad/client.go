// Package vad 提供 VAD gRPC 客户端工厂函数。
package vad

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"pronunciation-correction-system/internal/infrastructure/vad/vadpb"
)

// NewVADClient 创建并返回一个 VADServiceClient。
// addr 为 VAD gRPC 服务地址，例如 "localhost:50051"。
// 使用明文（不带 TLS）连接。
func NewVADClient(addr string) (vadpb.VADServiceClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("vad: dial %s: %w", addr, err)
	}
	return vadpb.NewVADServiceClient(conn), nil
}
