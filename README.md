# Enterprise AI Orchestrator

สมองกลาง AI สำหรับองค์กร สร้างด้วย Go + Ollama + pgvector

## Architecture

Open WebUI (53000) → Go Orchestrator (50000) → RAG/MCP/LLM

## Quick Start

### 1. Copy config

cp .env.example .env

### 2. Start services

docker compose up -d

### 3. Run Go server

go run cmd/server/main.go

## API

| Method | Endpoint             | หน้าที่                   |
| ------ | -------------------- | ------------------------- |
| GET    | /health              | ตรวจสอบสถานะ              |
| POST   | /v1/chat/completions | Chat (OpenAI-compatible)  |
| POST   | /v1/rag/ingest       | อัพโหลดเอกสาร             |
| POST   | /v1/rag/upload       | อัพโหลดไฟล์ .txt/.md/.pdf |

### Response format

ทุก endpoint จะตอบกลับในรูปแบบมาตรฐานเดียวกัน:

```json
{
   "data": {},
   "error": {
      "message": "",
      "code": ""
   }
}
```

หมายเหตุ: endpoint `/v1/chat/completions` ใช้ request schema แบบ OpenAI แต่ response ถูกครอบด้วย `data` ตามมาตรฐานบริษัท

### Upload constraints

- รองรับไฟล์: `.txt`, `.md`, `.pdf`
- ขนาดไฟล์สูงสุด: `10 MB`
- request ที่ไม่มีข้อความใน `messages` จะถูกปฏิเสธด้วย `400 BAD_REQUEST`

### PDF extraction notes

- ลำดับการดึงข้อความจาก PDF: `pdftotext` -> Go PDF parser -> OCR (`pdftoppm` + `tesseract`)
- สำหรับไฟล์สแกน แนะนำติดตั้งเครื่องมือ OCR:

```bash
brew install poppler tesseract tesseract-lang
```

### Upload test (real PDF)

```bash
UPLOAD_TEST_PDF="/Users/nipon.k/Documents/บค.002-2568 ประกาศ เรื่องแนวทางการขออนุมัติทำงานล่วงเวลาอย่างเคร่งครัด.pdf" \
go test ./internal/api/handlers -run TestUpload_WithProvidedPDF -v
```

### Incident note (important for running on another machine)

ปัญหาที่เจอจริงระหว่างทดสอบ `Upload`:

- อาการ: ได้ `400 PARSE_ERROR` พร้อมข้อความ `no text extracted from PDF`
- สาเหตุ: เครื่องที่รันไม่มีเครื่องมือ PDF/OCR ครบ หรือไฟล์ PDF เป็นสแกนที่ `pdftotext` ดึงข้อความไม่ได้
- แนวทางแก้ในโค้ด: เพิ่ม fallback การ parse เป็น 3 ชั้น
   - `pdftotext`
   - Go PDF parser
   - OCR (`pdftoppm` + `tesseract -l tha+eng`)

สิ่งที่ต้องมีใน runtime environment (เครื่องใหม่/เซิร์ฟเวอร์ใหม่):

- อย่างน้อย: `pdftotext` (จาก poppler)
- สำหรับไฟล์ PDF สแกน: ต้องมี `pdftoppm` + `tesseract` + Thai language data

คำสั่งติดตั้งตัวอย่าง:

- macOS (Homebrew)

```bash
brew install poppler tesseract tesseract-lang
```

- Alpine (Docker image)

```bash
apk add --no-cache poppler-utils tesseract-ocr tesseract-ocr-data-tha
```

หมายเหตุสำหรับ repo นี้: `Dockerfile` ติดตั้ง OCR dependencies แล้ว (`poppler-utils`, `tesseract-ocr`, `tesseract-ocr-data-eng`, `tesseract-ocr-data-tha`)

## Stack

- Language: Go 1.26
- Framework: Fiber v3
- LLM PoC: Ollama + qwen2.5:7b
- LLM Prod: vLLM + qwen2.5:32b
- Vector DB: PostgreSQL + pgvector
- Embedding: nomic-embed-text
