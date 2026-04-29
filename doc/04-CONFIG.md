# 04 — Configuration
> **Last Updated:** 2026-04-29

---

## Ollama (Native on iMac)

```bash
# Start
brew services start ollama

# Optimization
launchctl setenv OLLAMA_FLASH_ATTENTION 1
launchctl setenv OLLAMA_KV_CACHE_TYPE q8_0

# Models
ollama pull qwen2.5:7b
ollama pull nomic-embed-text-v2-moe
ollama pull scb10x/typhoon-ocr1.5-3b:latest

# Check
ollama list
ollama ps
```

---

## .env

```env
# Server
API_PORT=50000
RATE_LIMIT_PER_MINUTE=20
SESSION_EXPIRY=30m
CHAT_TIMEOUT=120s
RAG_INGEST_TIMEOUT=120s
AUDIT_TIMEOUT=5s
HEALTH_TIMEOUT=5s
SHUTDOWN_TIMEOUT=10s
MIGRATION_TIMEOUT=30s
MAX_UPLOAD_MB=10

# LLM
LLM_BACKEND=ollama
LLM_HOST=host.docker.internal
LLM_PORT=11434
LLM_MODEL=qwen2.5:7b
LLM_API_KEY=

# Embedding
EMBED_HOST=host.docker.internal
EMBED_PORT=11434
EMBED_MODEL=nomic-embed-text-v2-moe

# OCR
OCR_ENGINE=ollama
OCR_HOST=host.docker.internal
OCR_PORT=11434
OCR_MODEL=scb10x/typhoon-ocr1.5-3b:latest
OCR_TIMEOUT=180s
OCR_MAX_PAGES=20
OCR_PROMPT=Extract all readable text from this image. Preserve Thai and English text. Return only the extracted text.

# Database
DB_HOST=orchestrator-postgres
DB_PORT=5432
DB_NAME=orchestrator
DB_USER=orchestrator
DB_PASS=changeme
DB_POOL_MAX_CONNS=20
DB_POOL_MIN_CONNS=2
DB_POOL_MAX_CONN_LIFETIME=30m
DB_POOL_MAX_CONN_IDLE_TIME=5m
DB_POOL_HEALTH_CHECK_PERIOD=1m

# Active Directory
AD_SERVER=192.168.2.1
AD_PORT=389
AD_BASE_DN=DC=nutrition,DC=com
AD_DOMAIN=nutrition.com

# JWT
JWT_SECRET=change-this-to-random-string-in-production
JWT_EXPIRY=8h

# System Prompt
SYSTEM_PROMPT_FILE=prompts/system_general.tmpl
SYSTEM_PROMPT=
```

---

## Docker Commands

```bash
# Start services
docker compose up --build -d postgres orchestrator

# Start WebUI
docker compose up -d open-webui

# View logs
docker logs orchestrator-api --tail 20 -f

# Stop all
docker compose down

# Reset DB (ระวัง! ลบข้อมูลทั้งหมด)
docker compose down && docker volume rm orchestrator_postgres_data

# Check containers
docker ps
```

หมายเหตุ migration:
- ตอนนี้ฐานข้อมูล development ยังว่างและยังไม่ได้ใช้เป็น production data
- migration จึงถูกจัดเป็น baseline เดียวที่ `internal/infrastructure/migrations/sql/00001_baseline.sql`
- หลังมี production data แล้ว ห้ามแก้ migration ที่รันไปแล้ว ให้เพิ่มไฟล์ migration ใหม่เท่านั้น

---

## AD Configuration

```
Domain:   nutrition.com (ไม่ใช่ nutritionprofess.com)
Server:   192.168.2.1:389
Base DN:  DC=nutrition,DC=com
Format:   username@nutrition.com

หมายเหตุ:
- email ใช้ nutritionprofess.com
- login ใช้ nutrition.com
- ใช้ StartTLS (InsecureSkipVerify: true)
```

---

## Production Migration Checklist

```
เมื่อย้ายไป Ubuntu + NVIDIA:

1. แก้ .env:
   LLM_HOST=<ubuntu-server-ip>
   LLM_PORT=51434
   LLM_BACKEND=vllm
   LLM_API_KEY=<vllm-api-key-if-enabled>

   # ถ้า embedding ยังรันผ่าน Ollama หรือ service อื่น ให้แยก endpoint ไว้
   EMBED_HOST=<embedding-server-ip>
   EMBED_PORT=11434

   # OCR แยก endpoint ไว้ได้ จะคง Ollama dev หรือย้ายเป็น OCR service ภายหลัง
   OCR_ENGINE=ollama
   OCR_HOST=<ocr-server-ip>
   OCR_PORT=11434
   OCR_MODEL=scb10x/typhoon-ocr1.5-3b:latest

2. เปิด vLLM (uncomment ใน docker-compose.yml):
   # vllm:
   #   image: vllm/vllm-openai:latest
   #   runtime: nvidia

3. เปลี่ยน model:
   LLM_MODEL=Qwen/Qwen2.5-32B-Instruct-AWQ

4. ต้องมี vLLM/OpenAI-compatible adapter ที่เลือกผ่าน LLM_BACKEND ก่อนขึ้น production
```
