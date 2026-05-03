package mimo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Client struct {
	APIKey  string
	BaseURL string
	Model   string
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func New() *Client {
	baseURL := os.Getenv("MIMO_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.mimo.xiaomi.com/v1"
	}
	model := os.Getenv("MIMO_MODEL")
	if model == "" {
		model = "MiMo-v2.5"
	}
	return &Client{
		APIKey:  os.Getenv("MIMO_API_KEY"),
		BaseURL: baseURL,
		Model:   model,
	}
}

func (c *Client) AnalyzeDrift(configContent, configType string) (string, int, error) {
	prompt := fmt.Sprintf(`Analyze this %s configuration for security misconfigurations, cost optimization opportunities, and compliance issues. 

Return a JSON array of findings:
[{"resource": "type.name", "drift_type": "security|cost|compliance", "severity": "critical|high|medium|low", "title": "...", "description": "...", "remediation": "..."}]

Config:
%s`, configType, configContent)

	reqBody := chatRequest{
		Model: c.Model,
		Messages: []message{
			{Role: "system", Content: "You are an infrastructure security expert. Analyze configs for issues. Return valid JSON only."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
		MaxTokens:   2048,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", c.BaseURL+"/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", 0, err
	}

	content := ""
	if len(chatResp.Choices) > 0 {
		content = chatResp.Choices[0].Message.Content
	}
	tokens := chatResp.Usage.PromptTokens + chatResp.Usage.CompletionTokens
	return content, tokens, nil
}
