package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

const maxErrorBodyBytes = 4 << 10

// NomicAdapter implement EmbedderPort สำหรับ nomic-embed-text ผ่าน Ollama
type NomicAdapter struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewNomicAdapter สร้าง NomicAdapter ใหม่
func NewNomicAdapter(baseURL string, model string) *NomicAdapter {
	return &NomicAdapter{
		baseURL: baseURL,
		model:   model,
		client:  newHTTPClient(),
	}
}

// ollamaEmbedRequest โครงสร้าง request ที่ส่งไป Ollama
type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// ollamaEmbedResponse โครงสร้าง response จาก Ollama
type ollamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// Embed แปลงข้อความเป็น Vector
func (n *NomicAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	reqBody := ollamaEmbedRequest{
		Model:  n.model,
		Prompt: text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	slog.Info("embed request", "model", n.model, "text_len", len([]rune(text)), "url", n.baseURL+"/api/embeddings")

	req, err := http.NewRequestWithContext(ctx, "POST", n.baseURL+"/api/embeddings", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if err := checkStatus(resp, "embedder"); err != nil {
		return nil, err
	}

	var embedResp ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	slog.Info("embed response", "embedding_len", len(embedResp.Embedding))

	if len(embedResp.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	return embedResp.Embedding, nil
}

// Dimensions บอกขนาด Vector ของ nomic-embed-text
func (n *NomicAdapter) Dimensions() int {
	return 768
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
