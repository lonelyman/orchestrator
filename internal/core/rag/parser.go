package rag

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Parse(filename string, r io.Reader) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".txt", ".md":
		return parsePlainText(r)
	case ".pdf":
		return parsePDF(r)
	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

func parsePlainText(r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	return string(b), nil
}

func parsePDF(r io.Reader) (string, error) {
	// เขียนลง temp file ก่อน
	tmp, err := os.CreateTemp("", "upload-*.pdf")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := io.Copy(tmp, r); err != nil {
		return "", fmt.Errorf("write temp file: %w", err)
	}
	tmp.Close()

	// รัน pdftotext
	out, err := exec.Command("pdftotext", tmp.Name(), "-").Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext: %w", err)
	}

	text := strings.TrimSpace(string(out))
	if text == "" {
		return "", fmt.Errorf("no text extracted from PDF")
	}

	return text, nil
}
