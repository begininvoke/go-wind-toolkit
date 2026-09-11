package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout = 120 * time.Second
	// streamTimeout 流式请求整体上限:长生成(DDL/代码审查)可能远超普通请求。
	streamTimeout = 5 * time.Minute
)

// chatMessage OpenAI Chat API 消息
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatRequest OpenAI Chat Completion 请求
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// chatResponse OpenAI Chat Completion 响应
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// apiError API 错误响应
type apiError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Client OpenAI 兼容 HTTP 客户端
type Client struct {
	httpClient *http.Client
	config     *Config
}

// NewClient 创建 AI 客户端
func NewClient(config *Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: defaultTimeout},
		config:     config,
	}
}

// UpdateConfig 更新配置
func (c *Client) UpdateConfig(config *Config) {
	c.config = config
}

// Chat 发送聊天请求，返回回复内容
func (c *Client) Chat(systemPrompt string, userMessage string) (string, error) {
	if err := c.validate(); err != nil {
		return "", err
	}

	body, err := c.postChat(context.Background(), systemPrompt, userMessage)
	if err != nil {
		return "", err
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("API 未返回有效响应")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// chatStreamResponse OpenAI 流式 Chat Completion 的单个 SSE 数据帧
type chatStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// ChatStream 流式发送聊天请求,按 OpenAI 兼容的 SSE 协议接收增量回复。
// 每收到一个增量块就调用一次 onDelta(可为 nil);返回完整内容。
// 流式模式整体超时为 streamTimeout(长生成比普通请求更耗时)。
func (c *Client) ChatStream(systemPrompt string, userMessage string, onDelta func(string)) (string, error) {
	if err := c.validate(); err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), streamTimeout)
	defer cancel()

	resp, err := c.postChatStream(ctx, systemPrompt, userMessage)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", c.apiError(resp.StatusCode, body)
	}

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue // 空行或 SSE 注释
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}

		var frame chatStreamResponse
		if err := json.Unmarshal([]byte(payload), &frame); err != nil {
			// 部分网关在出错时也走 data: 帧返回 JSON 错误对象。
			return "", fmt.Errorf("解析流式响应帧失败: %w", err)
		}
		if len(frame.Choices) == 0 {
			continue
		}
		delta := frame.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		full.WriteString(delta)
		if onDelta != nil {
			onDelta(delta)
		}
	}
	if err := scanner.Err(); err != nil {
		return full.String(), fmt.Errorf("读取流式响应失败: %w", err)
	}

	content := full.String()
	if content == "" {
		return "", fmt.Errorf("API 未返回有效响应")
	}
	return content, nil
}

// validate 校验发起请求前的必要配置。
func (c *Client) validate() error {
	if c.config == nil {
		return fmt.Errorf("AI 配置未初始化")
	}
	if c.config.BaseURL == "" {
		return fmt.Errorf("API 地址不能为空")
	}
	if c.config.APIKey == "" && c.config.Provider != "ollama" {
		return fmt.Errorf("API 密钥不能为空")
	}
	return nil
}

// postChat 发送非流式请求并返回响应体。
func (c *Client) postChat(ctx context.Context, systemPrompt, userMessage string) ([]byte, error) {
	body, err := c.buildBody(systemPrompt, userMessage)
	if err != nil {
		return nil, err
	}

	url := strings.TrimRight(c.config.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.apiError(resp.StatusCode, data)
	}
	return data, nil
}

// postChatStream 发送流式请求,返回未读取的 SSE 响应,由调用方解析。
func (c *Client) postChatStream(ctx context.Context, systemPrompt, userMessage string) (*http.Response, error) {
	body, err := c.buildStreamBody(systemPrompt, userMessage)
	if err != nil {
		return nil, err
	}

	url := strings.TrimRight(c.config.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	return resp, nil
}

// buildBody 构造聊天请求体(流式由调用方在请求结构上追加)。
func (c *Client) buildBody(systemPrompt, userMessage string) ([]byte, error) {
	reqBody := chatRequest{
		Model:       c.config.Model,
		Messages:    []chatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: userMessage}},
		Temperature: c.config.Temperature,
		MaxTokens:   c.config.MaxTokens,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}
	return data, nil
}

// buildStreamBody 构造流式聊天请求体。
func (c *Client) buildStreamBody(systemPrompt, userMessage string) ([]byte, error) {
	reqBody := chatRequest{
		Model:       c.config.Model,
		Messages:    []chatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: userMessage}},
		Temperature: c.config.Temperature,
		MaxTokens:   c.config.MaxTokens,
		Stream:      true,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}
	return data, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}
}

// apiError 将非 200 响应转换为错误。
func (c *Client) apiError(status int, body []byte) error {
	var apiErr apiError
	if json.Unmarshal(body, &apiErr) == nil && apiErr.Error.Message != "" {
		return fmt.Errorf("API 错误 (%d): %s", status, apiErr.Error.Message)
	}
	return fmt.Errorf("API 错误 (%d): %s", status, string(body))
}

// TestConnection 测试 AI 连接
func (c *Client) TestConnection() (*StepResult, error) {
	content, err := c.Chat(
		"You are a helpful assistant. Reply with exactly: CONNECTION_OK",
		"Hello, please respond with CONNECTION_OK to confirm the connection is working.",
	)
	if err != nil {
		return &StepResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return &StepResult{
		Success: true,
		Content: content,
	}, nil
}
