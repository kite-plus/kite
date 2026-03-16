package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/amigoer/kite-blog/internal/config"
)

// AIService AI 辅助服务
type AIService struct {
	cfg *config.AIConfig
}

func NewAIService(cfg *config.AIConfig) *AIService {
	return &AIService{cfg: cfg}
}

// SummaryInput 自动摘要请求
type SummaryInput struct {
	Content string `json:"content"`
	Length  int    `json:"length"` // 目标字数，默认 200
}

// TagSuggestionInput 自动标签请求
type TagSuggestionInput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// chatMessage OpenAI 兼容的消息格式
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatRequest OpenAI 兼容的请求格式
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

// chatResponse OpenAI 兼容的响应格式
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// GenerateSummary 使用 AI 生成文章摘要
func (s *AIService) GenerateSummary(input SummaryInput) (string, error) {
	if !s.cfg.Enabled {
		return "", fmt.Errorf("AI 功能未启用")
	}

	length := input.Length
	if length <= 0 {
		length = 200
	}

	// 截取前 3000 字符避免 token 超限
	content := input.Content
	if len(content) > 3000 {
		content = content[:3000]
	}

	prompt := fmt.Sprintf("请为以下文章生成一段不超过 %d 字的中文摘要，要求简洁准确，直接输出摘要内容，不要加引号或前缀：\n\n%s", length, content)

	return s.callChat(prompt)
}

// SuggestTags 使用 AI 推荐标签
func (s *AIService) SuggestTags(input TagSuggestionInput) ([]string, error) {
	if !s.cfg.Enabled {
		return nil, fmt.Errorf("AI 功能未启用")
	}

	content := input.Content
	if len(content) > 2000 {
		content = content[:2000]
	}

	prompt := fmt.Sprintf("请根据以下文章标题和内容，推荐 3-5 个标签（tag），要求简短精准。请以 JSON 数组格式返回，例如 [\"Go\", \"博客\", \"架构\"]，不要加其他文字：\n\n标题：%s\n\n%s", input.Title, content)

	result, err := s.callChat(prompt)
	if err != nil {
		return nil, err
	}

	// 解析 JSON 数组
	result = strings.TrimSpace(result)
	// 去除可能的 markdown 代码块包裹
	result = strings.TrimPrefix(result, "```json")
	result = strings.TrimPrefix(result, "```")
	result = strings.TrimSuffix(result, "```")
	result = strings.TrimSpace(result)

	var tags []string
	if err := json.Unmarshal([]byte(result), &tags); err != nil {
		// 如果解析失败，按逗号分割
		parts := strings.Split(result, ",")
		for _, p := range parts {
			tag := strings.TrimSpace(strings.Trim(p, `"[]`))
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	return tags, nil
}

// callChat 调用 OpenAI 兼容 API
func (s *AIService) callChat(prompt string) (string, error) {
	apiURL := strings.TrimRight(s.cfg.Provider, "/") + "/v1/chat/completions"

	model := s.cfg.Model
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	reqBody := chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.7,
		MaxTokens:   500,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用 AI API 失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI API 返回错误 (%d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("AI 未返回结果")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}
