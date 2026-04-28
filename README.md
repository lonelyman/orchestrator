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

| Method | Endpoint             | หน้าที่                  |
| ------ | -------------------- | ------------------------ |
| GET    | /health              | ตรวจสอบสถานะ             |
| POST   | /v1/chat/completions | Chat (OpenAI-compatible) |
| POST   | /v1/rag/ingest       | อัพโหลดเอกสาร            |

## Stack

- Language: Go 1.26
- Framework: Fiber v3
- LLM PoC: Ollama + qwen2.5:7b
- LLM Prod: vLLM + qwen2.5:32b
- Vector DB: PostgreSQL + pgvector
- Embedding: nomic-embed-text
