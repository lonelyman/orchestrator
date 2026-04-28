package rag

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

// Parse อ่านไฟล์และแปลงเป็น text
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

// parsePlainText อ่านไฟล์ text ธรรมดา
func parsePlainText(r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	return string(b), nil
}

// parsePDF อ่านไฟล์ PDF
func parsePDF(r io.Reader) (string, error) {
	// อ่าน bytes ทั้งหมดก่อน
	b, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read pdf: %w", err)
	}

	// สร้าง ReaderAt จาก bytes
	readerAt := bytes.NewReader(b)

	// เปิด PDF
	pdfReader, err := pdf.NewReader(readerAt, int64(len(b)))
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}

	// อ่านทุกหน้า
	var sb strings.Builder
	for i := 1; i <= pdfReader.NumPage(); i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}

	result := strings.TrimSpace(sb.String())
	if result == "" {
		return "", fmt.Errorf("no text extracted from PDF")
	}

	return result, nil
}
