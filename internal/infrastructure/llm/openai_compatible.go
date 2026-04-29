package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

// OpenAICompatibleAdapter implements LLMPort for vLLM/OpenAI-compatible APIs.
type OpenAICompatibleAdapter struct {
	baseURL string
	model   string
	apiKey  string
	client  *http.Client
}

func NewOpenAICompatibleAdapter(baseURL, model, apiKey string) *OpenAICompatibleAdapter {
	return &OpenAICompatibleAdapter{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		apiKey:  apiKey,
		client:  newHTTPClient(),
	}
}

type openAIChatRequest struct {
	Model    string               `json:"model"`
	Messages []models.ChatMessage `json:"messages"`
	Stream   bool                 `json:"stream"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message models.ChatMessage `json:"message"`
	} `json:"choices"`
}

type openAIStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func (o *OpenAICompatibleAdapter) Chat(ctx context.Context, messages []models.ChatMessage) (string, error) {
	reqBody := openAIChatRequest{
		Model:    o.model,
		Messages: messages,
		Stream:   false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := o.newRequest(ctx, http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if err := checkStatus(resp, "openai-compatible chat"); err != nil {
		return "", err
	}

	var chatResp openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("openai-compatible chat returned no choices")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (o *OpenAICompatibleAdapter) ChatStream(ctx context.Context, messages []models.ChatMessage, onChunk func(string)) error {
	reqBody := openAIChatRequest{
		Model:    o.model,
		Messages: messages,
		Stream:   true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := o.newRequest(ctx, http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if err := checkStatus(resp, "openai-compatible chat stream"); err != nil {
		return err
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}

		var chunk openAIStreamResponse
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		if content := chunk.Choices[0].Delta.Content; content != "" {
			onChunk(content)
		}
	}

	return scanner.Err()
}

func (o *OpenAICompatibleAdapter) HealthCheck(ctx context.Context) error {
	req, err := o.newRequest(ctx, http.MethodGet, "/v1/models", nil)
	if err != nil {
		return err
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("openai-compatible llm not reachable: %w", err)
	}
	defer resp.Body.Close()

	return checkStatus(resp, "openai-compatible health")
}

func (o *OpenAICompatibleAdapter) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, o.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(o.apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+o.apiKey)
	}
	return req, nil
}
