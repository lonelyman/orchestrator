package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

const maxErrorBodyBytes = 4 << 10

// OllamaAdapter implement LLMPort สำหรับ Ollama
type OllamaAdapter struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewOllamaAdapter สร้าง OllamaAdapter ใหม่
func NewOllamaAdapter(baseURL, model string) *OllamaAdapter {
	return &OllamaAdapter{
		baseURL: baseURL,
		model:   model,
		client:  newHTTPClient(),
	}
}

// ollamaRequest คือโครงสร้าง request ที่ส่งไป Ollama
type ollamaRequest struct {
	Model    string               `json:"model"`
	Messages []models.ChatMessage `json:"messages"`
	Stream   bool                 `json:"stream"`
}

// ollamaResponse คือโครงสร้าง response จาก Ollama
type ollamaResponse struct {
	Message models.ChatMessage `json:"message"`
	Done    bool               `json:"done"`
}

// Chat ส่งข้อความและรอรับคำตอบแบบ non-stream
func (o *OllamaAdapter) Chat(ctx context.Context, messages []models.ChatMessage) (string, error) {
	reqBody := ollamaRequest{
		Model:    o.model,
		Messages: messages,
		Stream:   false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/api/chat", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if err := checkStatus(resp, "ollama chat"); err != nil {
		return "", err
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return ollamaResp.Message.Content, nil
}

// ChatStream ส่งข้อความและรับคำตอบแบบ stream ทีละ chunk
func (o *OllamaAdapter) ChatStream(ctx context.Context, messages []models.ChatMessage, onChunk func(string)) error {
	reqBody := ollamaRequest{
		Model:    o.model,
		Messages: messages,
		Stream:   true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/api/chat", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if err := checkStatus(resp, "ollama chat stream"); err != nil {
		return err
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var chunk ollamaResponse
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			continue
		}
		if chunk.Message.Content != "" {
			onChunk(chunk.Message.Content)
		}
		if chunk.Done {
			break
		}
	}

	return scanner.Err()
}

// HealthCheck ตรวจสอบว่า Ollama พร้อมใช้งานไหม
func (o *OllamaAdapter) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", o.baseURL, nil)
	if err != nil {
		return err
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama not reachable: %w", err)
	}
	defer resp.Body.Close()

	if err := checkStatus(resp, "ollama health"); err != nil {
		return err
	}

	return nil
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
		},
	}
}

func checkStatus(resp *http.Response, operation string) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	message := strings.TrimSpace(string(body))
	if message == "" {
		return fmt.Errorf("%s returned status %d", operation, resp.StatusCode)
	}
	return fmt.Errorf("%s returned status %d: %s", operation, resp.StatusCode, message)
}
