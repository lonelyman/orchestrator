# Enterprise AI Orchestrator — Living Document

> **Last Updated:** 2026-04-28
> **Status:** Phase 0 — Foundation (In Progress)
> **วิธีใช้:** ทุกครั้งที่เปิด session ใหม่ ให้ paste document นี้ให้ Claude ดูก่อน แล้วจะ update ต่อได้เลย

---

## 1. Project Overview

สร้าง **Enterprise AI Orchestrator** ด้วยภาษา Go เพื่อเป็นสมองกลางขององค์กร เชื่อมต่อข้อมูล (Multi-DB) และใช้เครื่องมืออัตโนมัติ (MCP) โดยมีความยืดหยุ่นจาก PoC ไปสู่ Production

---

## 2. Hardware

| Role                         | Machine       | Spec                    |
| ---------------------------- | ------------- | ----------------------- |
| **Development / PoC Server** | iMac          | Apple M1, 16GB RAM      |
| **Client**                   | MacBook Pro   | sysadmins-MacBook-Pro-2 |
| **Production (อนาคต)**       | Ubuntu Server | NVIDIA Blackwell        |

**การเข้าถึง Server:**

```bash
ssh imac_testai@10.19.105.32
```

---

## 3. Architecture

```
MacBook (Client)
    │ SSH
    ▼
iMac M1 16GB (Server)
    │
    ├── Ollama (Native on M1 Metal)
    │       └── qwen2.5:7b        ← LLM หลัก (4.7GB)
    │       └── nomic-embed-text  ← Embedding (274MB)
    │
    └── Docker
            ├── Go Orchestrator (Port 50000)  ← สมองกลาง
            ├── PostgreSQL + pgvector (Port 5432)
            └── Open WebUI (Port 53000)
```

### Port Map

| Service         | Port                                | หมายเหตุ       |
| --------------- | ----------------------------------- | -------------- |
| Ollama          | 11434 (internal) / 51434 (external) | Native on M1   |
| Open WebUI      | 53000                               | Chat Interface |
| Go Orchestrator | 50000                               | Backend API    |
| PostgreSQL      | 5432                                | Vector DB      |

### API Design

- **Standard:** OpenAI-compatible (`/v1/chat/completions`)
- **Gateway-ready:** รองรับ Header `X-User-ID`, `X-Session-ID` ตั้งแต่แรก
- **Migration-ready:** ทุก LLM connection ผ่าน Interface → สลับ Ollama → vLLM ได้ใน `.env`

---

## 4. Tech Stack Decisions

| ด้าน              | PoC                      | Production               |
| ----------------- | ------------------------ | ------------------------ |
| **Language**      | Go (Golang)              | Go (Golang)              |
| **Framework**     | Fiber v3                 | Fiber v3                 |
| **LLM Engine**    | Ollama (Native M1)       | vLLM (Docker, Ubuntu)    |
| **Primary Model** | qwen2.5:7b               | qwen2.5:32B MoE          |
| **Embedding**     | nomic-embed-text         | nomic / OpenAI (สลับได้) |
| **Vector DB**     | pgvector (PostgreSQL 16) | pgvector (PostgreSQL 16) |
| **Architecture**  | Hexagonal Architecture   | Hexagonal Architecture   |
| **Session**       | Isolated per user        | Shared Knowledge Base    |

### Key Design Principles

- **Migration-Ready:** ทุก connection ผ่าน Interface
- **Zero-Trust MCP:** SQL Server → Read-only + Validation Layer เสมอ
- **Gateway-Ready:** ออกแบบรองรับ API Gateway (Kong/Traefik) ตั้งแต่แรก
- **Prompt Language:** English System Prompt / Thai Response

---

## 5. Codebase Structure (Hexagonal Architecture)

```
enterprise-ai-orchestrator/
│
├── cmd/server/              ← main.go (Entrypoint)
│
├── internal/
│   ├── domain/              ← Core Business Logic (ไม่รู้จัก Fiber/Postgres)
│   │   ├── models/          ← ChatMessage, Document, Intent, Session
│   │   └── ports/           ← Interfaces: LLMPort, EmbedPort, VectorPort, MCPPort
│   │
│   ├── core/                ← Use Cases
│   │   ├── orchestrator/    ← Intent Router + Response Builder
│   │   ├── rag/             ← Ingest + Search Logic
│   │   └── mcp/             ← Tool Executor
│   │
│   └── infrastructure/      ← Adapters (implements ports)
│       ├── llm/             ← OllamaAdapter, vLLMAdapter, CloudAdapter
│       ├── embedder/        ← NomicAdapter, OpenAIEmbedAdapter
│       └── vector/          ← PgvectorAdapter
│
├── config/                  ← .env loader, AppConfig struct
└── docker/                  ← docker-compose files
```

### Core Interfaces

```go
// ports/llm.go
type LLMPort interface {
    Chat(ctx, messages, opts) (string, error)
    ChatStream(ctx, messages, opts, onChunk func(string)) error
    HealthCheck(ctx) error
}

// ports/embedder.go
type EmbedderPort interface {
    Embed(ctx, text string) ([]float32, error)
    Dimensions() int
}
```

---

## 6. Intent Router

| Phase     | วิธี                      | สถานะ    |
| --------- | ------------------------- | -------- |
| Phase 0-1 | Rule-based (keyword scan) | 🔜 Next  |
| Phase 3   | LLM-based classifier      | ⏳ อนาคต |

### Intent Types

- `direct` → LLM ตอบตรงๆ
- `rag` → ดึง context จาก pgvector ก่อน
- `mcp` → Query SQL Server ผ่าน MCP Bridge
- `cloud` → Escalate ไป Groq/OpenAI

---

## 7. Roadmap

| Phase       | หัวข้อ                                   | สถานะ                                   |
| ----------- | ---------------------------------------- | --------------------------------------- |
| **Phase 0** | Foundation: Ollama + Docker + Go + WebUI | ✅ Done                                 |
| **Phase 1** | RAG: Vector Ingestion + Search           | ✅ Done                                 |
| **Phase 2** | MCP: SQL Server Bridge                   | ✅ Structure Ready (SQL Server pending) |
| **Phase 3** | Orchestration: Intent Router รวมร่าง     | ⏳                                      |
| **Phase 4** | Production: Ubuntu + vLLM + Blackwell    | ⏳                                      |

---

## 8. Installation Log

### ✅ Completed

#### [2026-04-28] SSH Setup

- เปิด Remote Login บน iMac
- เชื่อมต่อ MacBook → iMac สำเร็จ
- Command: `ssh imac_testai@10.19.105.32`

#### [2026-04-28] Ollama Installation

- ติดตั้งผ่าน Homebrew: `brew install ollama`
- Version: `0.21.2_1`
- รันเป็น Background Service: `brew services start ollama`

#### [2026-04-28] Ollama Optimization

- เปิด Flash Attention: `OLLAMA_FLASH_ATTENTION=1`
   - ผล: เร็วขึ้น ~20%, คำนวณ Attention เป็น block แทน token
- เปิด KV Cache Quantization: `OLLAMA_KV_CACHE_TYPE=q8_0`
   - ผล: ประหยัด RAM ~30%, แทบไม่เสีย quality

#### [2026-04-28] Model Pull

- `qwen2.5:7b` — 4.7GB ✅
- `nomic-embed-text` — 274MB ✅

#### [2026-04-28] Model Test

- Thai language: ✅ ตอบภาษาไทยได้
- Speed: ✅ ผ่านการ optimize แล้ว
- M1 Metal: ✅ ทำงานได้จริง

---

#### [2026-04-28] Docker Desktop

- ติดตั้ง Docker Desktop version 29.4.1 ✅
- ทดสอบด้วย hello-world (arm64v8) ✅

#### [2026-04-28] docker-compose — PostgreSQL + Open WebUI

- ไฟล์: ~/orchestrator/docker/docker-compose.yml
- PostgreSQL 16 + pgvector: port 5432 ✅
- Open WebUI: port 53000 ✅
- เชื่อมต่อ Ollama ผ่าน host.docker.internal:11434 ✅

#### [2026-04-28] WebUI Configuration

- ตั้ง Default System Prompt: English instruction / Thai response ✅
- Disable nomic-embed-text และ qwen:latest ออกจาก model list ✅
- Enable เฉพาะ qwen2.5:7b ✅

#### [2026-04-28] Phase 0 ทดสอบสำเร็จ

- Chat ภาษาไทยผ่าน WebUI → Ollama ✅
- M1 Metal GPU 100% ✅
- System Prompt บังคับภาษาไทยได้ ✅

---

#### [2026-04-28] Go Installation

- Go 1.26.2 darwin/arm64 ✅
- go mod init: `github.com/enterprise-ai/orchestrator` ✅

#### [2026-04-28] VS Code Remote SSH

- Extension: Remote - SSH ✅
- เชื่อมต่อ MacBook → iMac โดยตรง ✅
- เปิด folder: `/Users/imac_testai/orchestrator` ✅

#### [2026-04-28] Git Setup

- git init + push ขึ้น GitHub ✅
- Branch: main (protected), dev (ทำงาน) ✅
- Remote: https://github.com/lonelyman/orchestrator.git ✅

#### [2026-04-28] Go Files Created (Phase 0)

| ไฟล์                                         | Package      | หน้าที่                                         |
| -------------------------------------------- | ------------ | ----------------------------------------------- |
| `internal/domain/models/chat.go`             | models       | ChatMessage, ChatRequest, ChatResponse, Session |
| `internal/domain/ports/llm.go`               | ports        | LLMPort interface                               |
| `internal/infrastructure/llm/ollama.go`      | llm          | OllamaAdapter (implements LLMPort)              |
| `internal/core/orchestrator/orchestrator.go` | orchestrator | สมองกลาง รับ request ส่งต่อ LLM                 |

#### [2026-04-28] Go Orchestrator Phase 0 — สำเร็จ ✅

| ไฟล์                 | Package | หน้าที่              |
| -------------------- | ------- | -------------------- |
| `config/config.go`   | config  | โหลด env variables   |
| `cmd/server/main.go` | main    | Fiber v3 HTTP Server |

**ทดสอบผ่านทั้งหมด:**

- `GET /health` → `{"status":"healthy"}` ✅
- `POST /v1/chat/completions` → ตอบภาษาไทย ✅
- Git commit: `feat: phase 0 - go orchestrator with ollama adapter` ✅

---

#### [2026-04-28] Phase 1 RAG Engine — สำเร็จ ✅

| ไฟล์                                         | Package  | หน้าที่                          |
| -------------------------------------------- | -------- | -------------------------------- |
| `internal/domain/ports/embedder.go`          | ports    | EmbedderPort interface           |
| `internal/domain/ports/vector.go`            | ports    | VectorPort interface             |
| `internal/domain/models/document.go`         | models   | Document, SearchResult           |
| `internal/infrastructure/embedder/nomic.go`  | embedder | NomicAdapter (nomic-embed-text)  |
| `internal/infrastructure/vector/pgvector.go` | vector   | PgvectorAdapter + fallback logic |
| `internal/core/rag/rag.go`                   | rag      | Ingest + Search + BuildContext   |

**Key Fix — pgvector Search:**

- ปัญหา: similarity search ได้ 0 แถวทั้งที่มีข้อมูลใน DB
- แก้ด้วย: parameterized query + fallback ดึงเอกสารล่าสุดเมื่อ similarity = 0

**ทดสอบผ่านทั้งหมด:**

- `POST /v1/rag/ingest` → บันทึกเอกสารลง pgvector ✅
- `POST /v1/chat/completions` → AI ตอบโดยอ้างอิงเอกสารจริง ✅
- RAG context อ้าง source: HR-Policy-2024.pdf ✅
- Git commit: `feat: phase 1 - RAG engine with pgvector` ✅

---

#### [2026-04-28] MacBook Setup — สำเร็จ ✅

- Clone repo จาก GitHub ✅
- สร้าง Dockerfile (multi-stage build) ✅
- สร้าง .env.example, .gitignore, README.md ✅
- ย้าย docker-compose.yml ไป root ✅
- รัน PostgreSQL + pgvector ผ่าน Docker ✅
- Build และรัน Go Server ใน Docker ✅
- ทดสอบ Chat ผ่าน qwen3.5:cloud (ชั่วคราว) ✅

**หมายเหตุ:** ใช้ qwen3.5:cloud ชั่วคราว รอ qwen2.5:7b download เสร็จบน iMac

| ไฟล์                                       | Package | หน้าที่                                 |
| ------------------------------------------ | ------- | --------------------------------------- |
| `internal/domain/models/mcp.go`            | models  | Tool, ToolCall, ToolResult              |
| `internal/domain/ports/mcp.go`             | ports   | MCPPort interface                       |
| `internal/core/mcp/executor.go`            | mcp     | Tool Registry + Executor                |
| `internal/infrastructure/mcp/sqlserver.go` | mcp     | SQLServerAdapter (Zero-Trust Read-only) |

**Zero-Trust Validation:**

- SELECT เท่านั้น ✅
- Block: INSERT, UPDATE, DELETE, DROP, CREATE, ALTER, TRUNCATE, EXEC ✅
- Driver: `github.com/microsoft/go-mssqldb` ✅
- Git commit: `feat: phase 2 - MCP bridge structure` ✅

**หมายเหตุ:** ยังไม่ได้เชื่อมต่อ SQL Server จริง รอ Phase ถัดไป

---

### 🔜 Next Steps (Phase 3 — Intent Router)

1. **Intent Classifier** — ตัดสินใจว่าจะใช้ RAG / MCP / Direct
2. **อัพเดต Orchestrator** — รวม RAG + MCP เข้าด้วยกัน
3. **ทดสอบ** — AI ตัดสินใจเองว่าจะใช้ path ไหน

---

## 9. Configuration

### Ollama Service

```
Location: Native on iMac M1
Service:  brew services (homebrew.mxcl.ollama)
Env:      OLLAMA_FLASH_ATTENTION=1
          OLLAMA_KV_CACHE_TYPE=q8_0
Internal Port: 11434
```

### .env Template (จะสร้างเมื่อเริ่ม Go)

```env
# LLM Backend
LLM_BACKEND=ollama
LLM_HOST=host.docker.internal
LLM_PORT=11434
LLM_MODEL=qwen2.5:7b

# Embedding
EMBED_BACKEND=ollama
EMBED_MODEL=nomic-embed-text

# Database
DB_HOST=postgres
DB_PORT=5432
DB_NAME=orchestrator
DB_USER=orchestrator
DB_PASS=changeme

# Server
API_PORT=50000
```

---

## 10. Notes & Decisions

| วันที่     | การตัดสินใจ                       | เหตุผล                                        |
| ---------- | --------------------------------- | --------------------------------------------- |
| 2026-04-28 | ใช้ Ollama Native (ไม่ใช่ Docker) | รีด M1 Metal GPU ได้เต็มที่                   |
| 2026-04-28 | qwen2.5:7b สำหรับ PoC             | Sweet spot: ไว + คุณภาพดี บน 16GB             |
| 2026-04-28 | nomic-embed-text                  | เบา เร็ว รองรับ Matryoshka Embeddings         |
| 2026-04-28 | English Prompt / Thai Response    | AI เข้าใจ logic ดีกว่า, ทีมใช้งานง่ายกว่า     |
| 2026-04-28 | Hexagonal Architecture            | สลับ Ollama → vLLM ได้โดยไม่ rewrite          |
| 2026-04-28 | Gateway-ready Headers             | รองรับ Kong/Traefik ในอนาคตโดยไม่ต้องแก้ code |
