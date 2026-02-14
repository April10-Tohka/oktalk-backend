// Package http 提供 HTTP 客户端
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client HTTP 客户端
type Client struct {
	httpClient *http.Client
	baseURL    string
	headers    map[string]string
}

// ClientOption 客户端选项
type ClientOption func(*Client)

// WithTimeout 设置超时
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithBaseURL 设置基础 URL
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithHeader 设置请求头
func WithHeader(key, value string) ClientOption {
	return func(c *Client) {
		c.headers[key] = value
	}
}

// NewClient 创建 HTTP 客户端
func NewClient(opts ...ClientOption) *Client {
	client := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		headers: make(map[string]string),
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Get 发送 GET 请求
func (c *Client) Get(ctx context.Context, url string, result interface{}) error {
	return c.do(ctx, http.MethodGet, url, nil, result)
}

// Post 发送 POST 请求
func (c *Client) Post(ctx context.Context, url string, body, result interface{}) error {
	return c.do(ctx, http.MethodPost, url, body, result)
}

// Put 发送 PUT 请求
func (c *Client) Put(ctx context.Context, url string, body, result interface{}) error {
	return c.do(ctx, http.MethodPut, url, body, result)
}

// Delete 发送 DELETE 请求
func (c *Client) Delete(ctx context.Context, url string, result interface{}) error {
	return c.do(ctx, http.MethodDelete, url, nil, result)
}

// do 执行请求
func (c *Client) do(ctx context.Context, method, url string, body, result interface{}) error {
	// 完整 URL
	fullURL := url
	if c.baseURL != "" {
		fullURL = c.baseURL + url
	}

	// 构建请求体
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// 检查状态码
	if resp.StatusCode >= 400 {
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}
