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
| GET    | /live                | Liveness: process ยังตอบ HTTP |
| GET    | /ready               | Readiness: DB พร้อมรับ traffic |
| GET    | /health              | Full dependency health: DB + LLM + Embedder |
| POST   | /v1/chat/completions | Chat (OpenAI-compatible)  |
| POST   | /v1/rag/ingest       | อัพโหลดเอกสาร             |
| POST   | /v1/rag/upload       | อัพโหลดไฟล์ .txt/.md/.pdf |
| GET    | /v1/rag/documents    | รายการเอกสารใน RAG        |
| DELETE | /v1/rag/documents/:source | ลบเอกสารตาม source |

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

ทุก response จะมี header `X-Request-ID`; client ส่ง header นี้มาเองได้เพื่อ trace request ข้ามระบบ

หมายเหตุ: endpoint `/v1/chat/completions` ใช้ request schema แบบ OpenAI แต่ response ถูกครอบด้วย `data` ตามมาตรฐานบริษัท

### Chat constraints

- `model` ต้องตรงกับ model ที่ server expose ผ่าน `/v1/models`
- รองรับ role เฉพาะ `system`, `user`, `assistant`
- ต้องมี `messages` อย่างน้อย 1 รายการ และ message สุดท้ายต้องเป็น `user`
- จำกัด 64 messages, 20,000 ตัวอักษรต่อ message, 60,000 ตัวอักษรรวม

### Prompt management

- ค่า default ใช้ `SYSTEM_PROMPT_FILE=prompts/system_general.tmpl`
- ใช้ `SYSTEM_PROMPT` เป็น fallback/override เมื่อไม่ได้กำหนดไฟล์
- Docker image copy โฟลเดอร์ `prompts/` เข้า `/app/prompts`

### Upload constraints

- รองรับไฟล์: `.txt`, `.md`, `.pdf`
- ขนาดไฟล์สูงสุด: `10 MB`

### PDF extraction notes

- ลำดับการดึงข้อความจาก PDF: `pdftotext` -> Go PDF parser -> OCR
- OCR เลือกได้ด้วย `OCR_ENGINE`:
  - `ollama`: ใช้โมเดล AI OCR ผ่าน Ollama `/api/generate` เช่น `scb10x/typhoon-ocr1.5-3b:latest`
  - `tesseract`: ใช้ local binary `pdftoppm` + `tesseract -l tha+eng` ถ้าติดตั้งเอง
  - `disabled`: ไม่ทำ OCR ถ้า PDF ไม่มี embedded text
- `.env.example` ตั้งค่า dev เป็น `OCR_ENGINE=ollama` เพื่อใช้โมเดล OCR ที่รันใน Ollama อยู่แล้ว
- สำหรับ runtime แบบ Docker image ของ repo นี้ ติดตั้งเฉพาะ `poppler-utils` เพื่อใช้ `pdftotext` และ `pdftoppm`; ไม่ bundle Tesseract แล้ว
- ถ้าต้องการใช้ fallback แบบ `OCR_ENGINE=tesseract` บนเครื่อง local ให้ติดตั้งเพิ่ม:

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
   - OCR (`ollama` AI OCR เป็นค่า default ของ Docker image)

สิ่งที่ต้องมีใน runtime environment (เครื่องใหม่/เซิร์ฟเวอร์ใหม่):

- อย่างน้อย: `pdftotext` (จาก poppler)
- สำหรับ `OCR_ENGINE=ollama`: ต้องมี `pdftoppm` และ Ollama ที่มีโมเดล OCR
- สำหรับ `OCR_ENGINE=tesseract`: ต้องติดตั้ง `tesseract` + Thai language data เอง

คำสั่งติดตั้งตัวอย่าง:

- macOS (Homebrew)

```bash
brew install poppler
```

- Alpine

```bash
apk add --no-cache poppler-utils
```

หมายเหตุสำหรับ repo นี้: `Dockerfile` ติดตั้งเฉพาะ `poppler-utils`; scanned PDF ใช้ Ollama AI OCR ผ่าน `OCR_ENGINE=ollama`

## Stack

- Language: Go 1.26
- Framework: Fiber v3
- LLM PoC: Ollama + qwen2.5:7b
- LLM Prod: vLLM + qwen2.5:32b
- Vector DB: PostgreSQL + pgvector
- Embedding: nomic-embed-text
- Migrations: Embedded goose baseline migration at startup
