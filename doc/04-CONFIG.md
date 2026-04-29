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

# Check
ollama list
ollama ps
```

---

## .env

```env
# Server
API_PORT=50000

# LLM
LLM_BACKEND=ollama
LLM_HOST=host.docker.internal
LLM_PORT=11434
LLM_MODEL=qwen2.5:7b

# Embedding
EMBED_MODEL=nomic-embed-text-v2-moe

# Database
DB_HOST=orchestrator-postgres
DB_PORT=5432
DB_NAME=orchestrator
DB_USER=orchestrator
DB_PASS=changeme

# Active Directory
AD_SERVER=192.168.2.1
AD_PORT=389
AD_BASE_DN=DC=nutrition,DC=com
AD_DOMAIN=nutrition.com

# JWT
JWT_SECRET=change-this-to-random-string-in-production
JWT_EXPIRY=8h

# System Prompt
SYSTEM_PROMPT=You are a helpful enterprise AI assistant. You must always respond in Thai language only. Never use Chinese, English, or any other language. Thai language only.
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

2. เปิด vLLM (uncomment ใน docker-compose.yml):
   # vllm:
   #   image: vllm/vllm-openai:latest
   #   runtime: nvidia

3. เปลี่ยน model:
   LLM_MODEL=Qwen/Qwen2.5-32B-Instruct-AWQ

4. ไม่ต้องแก้ code อะไรเลย ✅
```
