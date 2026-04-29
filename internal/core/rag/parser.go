package rag

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
)

const MaxUploadBytes int64 = 10 << 20 // 10 MB

type ParseOptions struct {
	MaxBytes    int64
	OCREngine   string
	OCRBaseURL  string
	OCRModel    string
	OCRPrompt   string
	OCRTimeout  time.Duration
	OCRMaxPages int
}

func Parse(filename string, r io.Reader) (string, error) {
	return ParseWithLimit(filename, r, MaxUploadBytes)
}

func ParseWithLimit(filename string, r io.Reader, maxBytes int64) (string, error) {
	return ParseWithOptions(filename, r, ParseOptions{MaxBytes: maxBytes})
}

func ParseWithOptions(filename string, r io.Reader, opts ParseOptions) (string, error) {
	maxBytes := opts.MaxBytes
	if maxBytes <= 0 {
		maxBytes = MaxUploadBytes
	}
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".txt", ".md":
		return parsePlainText(r, maxBytes)
	case ".pdf":
		return parsePDF(r, maxBytes, opts)
	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

func parsePlainText(r io.Reader, maxBytes int64) (string, error) {
	b, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	if int64(len(b)) > maxBytes {
		return "", fmt.Errorf("file too large")
	}
	return string(b), nil
}

func parsePDF(r io.Reader, maxBytes int64, opts ParseOptions) (string, error) {
	// เขียนลง temp file ก่อน
	tmp, err := os.CreateTemp("", "upload-*.pdf")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	lr := &io.LimitedReader{R: r, N: maxBytes + 1}
	written, err := io.Copy(tmp, lr)
	if err != nil {
		return "", fmt.Errorf("write temp file: %w", err)
	}
	if written > maxBytes {
		return "", fmt.Errorf("file too large")
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close temp file: %w", err)
	}

	text, err := parsePDFWithPdftotext(tmp.Name())
	if err != nil || text == "" {
		text, err = parsePDFWithGo(tmp.Name())
		if err != nil || text == "" {
			text, err = parsePDFWithOCR(tmp.Name(), opts)
			if err != nil {
				return "", fmt.Errorf("extract pdf text: %w", err)
			}
		}
	}

	if text == "" {
		return "", fmt.Errorf("no text extracted from PDF")
	}

	return text, nil
}

func parsePDFWithPdftotext(path string) (string, error) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return "", err
	}

	out, err := exec.Command("pdftotext", path, "-").Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func parsePDFWithGo(path string) (string, error) {
	f, reader, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	plainTextReader, err := reader.GetPlainText()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, plainTextReader); err != nil {
		return "", err
	}

	return strings.TrimSpace(buf.String()), nil
}

func parsePDFWithOCR(path string, opts ParseOptions) (string, error) {
	switch strings.ToLower(strings.TrimSpace(opts.OCREngine)) {
	case "", "tesseract":
		return parsePDFWithTesseractOCR(path, opts.OCRMaxPages)
	case "ollama":
		return parsePDFWithOllamaOCR(path, opts)
	case "disabled":
		return "", fmt.Errorf("ocr disabled")
	default:
		return "", fmt.Errorf("unsupported OCR engine: %s", opts.OCREngine)
	}
}

func parsePDFWithTesseractOCR(path string, maxPages int) (string, error) {
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return "", err
	}
	if _, err := exec.LookPath("tesseract"); err != nil {
		return "", err
	}

	tmpDir, err := os.MkdirTemp("", "pdf-ocr-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	// Convert every page to PNG; OCR runs page-by-page afterwards.
	if out, err := exec.Command("pdftoppm", "-png", path, filepath.Join(tmpDir, "page")).CombinedOutput(); err != nil {
		return "", fmt.Errorf("pdftoppm: %v: %s", err, strings.TrimSpace(string(out)))
	}

	images, err := filepath.Glob(filepath.Join(tmpDir, "page-*.png"))
	if err != nil {
		return "", err
	}
	if len(images) == 0 {
		return "", fmt.Errorf("ocr: no rendered pages")
	}
	sortPageImages(images)

	if maxPages <= 0 {
		maxPages = 20
	}
	if len(images) > maxPages {
		images = images[:maxPages]
	}

	var sb strings.Builder
	for _, image := range images {
		out, err := exec.Command("tesseract", image, "stdout", "-l", "tha+eng").CombinedOutput()
		if err != nil {
			continue
		}
		pageText := strings.TrimSpace(string(out))
		if pageText == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(pageText)
	}

	result := strings.TrimSpace(sb.String())
	if result == "" {
		return "", fmt.Errorf("ocr: no text extracted")
	}

	return result, nil
}

type ollamaOCRRequest struct {
	Model  string   `json:"model"`
	Prompt string   `json:"prompt"`
	Images []string `json:"images"`
	Stream bool     `json:"stream"`
}

type ollamaOCRResponse struct {
	Response string `json:"response"`
}

func parsePDFWithOllamaOCR(path string, opts ParseOptions) (string, error) {
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return "", err
	}

	if strings.TrimSpace(opts.OCRBaseURL) == "" {
		return "", fmt.Errorf("ollama OCR base URL is required")
	}
	if strings.TrimSpace(opts.OCRModel) == "" {
		return "", fmt.Errorf("ollama OCR model is required")
	}

	images, cleanup, err := renderPDFPages(path, opts.OCRMaxPages)
	if err != nil {
		return "", err
	}
	defer cleanup()

	timeout := opts.OCRTimeout
	if timeout <= 0 {
		timeout = 180 * time.Second
	}

	prompt := strings.TrimSpace(opts.OCRPrompt)
	if prompt == "" {
		prompt = "Extract all readable text from this image. Preserve Thai and English text. Return only the extracted text."
	}

	client := &http.Client{Timeout: timeout}
	var sb strings.Builder
	var lastErr error
	for _, image := range images {
		text, err := ocrImageWithOllama(client, opts.OCRBaseURL, opts.OCRModel, prompt, image)
		if err != nil {
			lastErr = err
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(text)
	}

	result := strings.TrimSpace(sb.String())
	if result == "" {
		if lastErr != nil {
			return "", fmt.Errorf("ollama ocr failed: %w", lastErr)
		}
		return "", fmt.Errorf("ollama ocr: no text extracted")
	}

	return result, nil
}

func renderPDFPages(path string, maxPages int) ([]string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "pdf-ocr-*")
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() { os.RemoveAll(tmpDir) }

	if out, err := exec.Command("pdftoppm", "-png", path, filepath.Join(tmpDir, "page")).CombinedOutput(); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("pdftoppm: %v: %s", err, strings.TrimSpace(string(out)))
	}

	images, err := filepath.Glob(filepath.Join(tmpDir, "page-*.png"))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if len(images) == 0 {
		cleanup()
		return nil, func() {}, fmt.Errorf("ocr: no rendered pages")
	}
	sortPageImages(images)

	if maxPages <= 0 {
		maxPages = 20
	}
	if len(images) > maxPages {
		images = images[:maxPages]
	}

	return images, cleanup, nil
}

func sortPageImages(images []string) {
	sort.Slice(images, func(i, j int) bool {
		return pageNumber(images[i]) < pageNumber(images[j])
	})
}

func pageNumber(path string) int {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	idx := strings.LastIndex(name, "-")
	if idx < 0 || idx == len(name)-1 {
		return 0
	}
	n, err := strconv.Atoi(name[idx+1:])
	if err != nil {
		return 0
	}
	return n
}

func ocrImageWithOllama(client *http.Client, baseURL, model, prompt, imagePath string) (string, error) {
	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}

	reqBody := ollamaOCRRequest{
		Model:  model,
		Prompt: prompt,
		Images: []string{base64.StdEncoding.EncodeToString(imageBytes)},
		Stream: false,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return "", fmt.Errorf("ollama ocr returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var out ollamaOCRResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}

	return out.Response, nil
}
