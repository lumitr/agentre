package toolext

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPExecutor 通过 HTTP 调用外部工具。
type HTTPExecutor struct {
	URL     string            `json:"url"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Timeout int               `json:"timeout,omitempty"`
}

// NewHTTPExecutor 从 JSON 配置字符串创建 HTTPExecutor。
func NewHTTPExecutor(conf string) *HTTPExecutor {
	var e HTTPExecutor
	_ = json.Unmarshal([]byte(conf), &e)
	if e.Method == "" {
		e.Method = "POST"
	}
	if e.Timeout <= 0 {
		e.Timeout = 30
	}
	return &e
}

func (e *HTTPExecutor) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	if e.URL == "" {
		return "", fmt.Errorf("http_executor: url is required")
	}
	timeout := time.Duration(e.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, e.Method, e.URL, bytes.NewReader(params))
	if err != nil {
		return "", fmt.Errorf("http_executor: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range e.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http_executor: do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", fmt.Errorf("http_executor: read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http_executor: status %d: %s", resp.StatusCode, string(body))
	}
	return string(body), nil
}
