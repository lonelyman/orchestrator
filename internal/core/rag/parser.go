package rag

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ledongthuc/pdf"
)

const MaxUploadBytes int64 = 10 << 20 // 10 MB

func Parse(filename string, r io.Reader) (string, error) {
	return ParseWithLimit(filename, r, MaxUploadBytes)
}

func ParseWithLimit(filename string, r io.Reader, maxBytes int64) (string, error) {
	if maxBytes <= 0 {
		maxBytes = MaxUploadBytes
	}
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".txt", ".md":
		return parsePlainText(r, maxBytes)
	case ".pdf":
		return parsePDF(r, maxBytes)
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

func parsePDF(r io.Reader, maxBytes int64) (string, error) {
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
			text, err = parsePDFWithOCR(tmp.Name())
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

func parsePDFWithOCR(path string) (string, error) {
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
	sort.Strings(images)

	if len(images) > 20 {
		images = images[:20]
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
