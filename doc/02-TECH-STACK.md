# 02 — Tech Stack & Codebase
> **Last Updated:** 2026-04-29

---

## Tech Stack

| ด้าน | PoC | Production |
|---|---|---|
| **Language** | Go 1.26 | Go 1.26 |
| **Framework** | Fiber v3 | Fiber v3 |
| **LLM Engine** | Ollama (Native M1) | vLLM (Docker, Ubuntu) |
| **Primary Model** | qwen2.5:7b | qwen2.5:32B MoE |
| **Embedding** | nomic-embed-text-v2-moe | nomic / OpenAI (สลับได้) |
| **Vector DB** | pgvector (PostgreSQL 16) | pgvector (PostgreSQL 16) |
| **Session DB** | PostgreSQL | PostgreSQL |
| **Auth** | AD/LDAP + JWT | AD/LDAP + JWT |
| **Architecture** | Hexagonal Architecture | Hexagonal Architecture |

## Key Design Principles
- **Migration-Ready:** Core เรียกผ่าน `LLMPort` → เพิ่ม vLLM adapter ได้โดยไม่แตะ orchestrator logic
- **Zero-Trust MCP:** SQL Server → Read-only + Validation Layer เสมอ
- **Gateway-Ready:** รองรับ Header `X-User-ID`, `X-Session-ID`
- **Standard Response:** `{"data":{}}` หรือ `{"error":{}}` เสมอ
- **OpenAI-Compatible:** `/v1/*` endpoints ใช้กับ WebUI ได้เลย

## LLM Adapter Plan

สถานะปัจจุบัน:
- Development ใช้ Ollama adapter (`internal/infrastructure/llm/ollama.go`)
- Production target คือ vLLM ผ่าน OpenAI-compatible adapter (`internal/infrastructure/llm/openai_compatible.go`)
- `LLM_BACKEND=ollama|vllm|openai-compatible` ใช้เลือก adapter ตอน startup
- Embedding endpoint แยกด้วย `EMBED_HOST`/`EMBED_PORT` เพราะ vLLM chat endpoint ไม่จำเป็นต้อง serve Ollama embeddings API

แนวทางที่ต้องรักษา:
- อย่าใส่ vLLM/OpenAI schema ปนใน Ollama adapter
- ให้ adapter ใหม่ implement `domain/ports.LLMPort`
- HTTP hardening เช่น timeout, status handling, retry/circuit breaker ควรอยู่ที่ adapter boundary

---

## Codebase Structure

```
orchestrator/
├── Dockerfile                          ← Multi-stage (Go + poppler + tesseract)
├── docker-compose.yml                  ← postgres + orchestrator + webui
├── .env / .env.example
├── go.mod / go.sum
├── cmd/server/main.go                  ← Entrypoint + graceful shutdown
├── config/config.go                    ← AppConfig struct
├── internal/infrastructure/migrations/ ← Embedded goose baseline migration
└── internal/
    ├── app/
    │   └── server.go                   ← DB, migrations, adapters, core wiring
    ├── api/
    │   ├── router.go                   ← Public/protected/admin route registration
    │   ├── handlers/
    │   │   ├── response.go             ← OK() / Fail() standard response
    │   │   ├── health.go               ← GET /live, /ready, /health
    │   │   ├── auth.go                 ← POST /auth/login, GET /auth/me
    │   │   ├── chat.go                 ← POST /v1/chat/completions + stream
    │   │   └── rag.go                  ← POST /v1/rag/ingest + upload
    │   └── middleware/
    │       ├── auth.go                 ← JWT verification
    │       └── session.go              ← Session create/restore
    ├── domain/
    │   ├── models/
    │   │   ├── chat.go                 ← ChatMessage, ChatRequest, ChatResponse
    │   │   ├── document.go             ← Document, SearchResult
    │   │   ├── mcp.go                  ← Tool, ToolCall, ToolResult
    │   │   ├── intent.go               ← Intent: RAG, MCP, Direct
    │   │   ├── auth.go                 ← User, LoginRequest, Claims
    │   │   └── session.go              ← Session, Message, SessionHistory
    │   └── ports/
    │       ├── llm.go                  ← LLMPort
    │       ├── embedder.go             ← EmbedderPort
    │       ├── vector.go               ← VectorPort
    │       ├── mcp.go                  ← MCPPort
    │       └── session.go              ← SessionPort
    ├── core/
    │   ├── intent/classifier.go        ← Rule-based keyword classifier
    │   ├── orchestrator/orchestrator.go ← Intent + RAG + Session + LLM
    │   ├── rag/
    │   │   ├── rag.go                  ← Ingest + Search + BuildContext
    │   │   ├── chunker.go              ← chunk size 200, overlap 20
    │   │   └── parser.go               ← PDF(pdftotext→GoLib→Ollama/Tesseract OCR) + TXT + MD
    │   └── mcp/executor.go             ← Tool Registry
    └── infrastructure/
        ├── llm/ollama.go               ← Chat + Stream + HealthCheck
        ├── embedder/nomic.go           ← nomic-embed-text-v2-moe
        ├── vector/pgvector.go          ← Store + Search + fallback
        ├── session/postgres.go         ← CreateSession + GetHistory + SaveMessage
        ├── auth/
        │   ├── ldap.go                 ← AD Authentication (nutrition.com)
        │   └── jwt.go                  ← Generate + Verify JWT
        └── mcp/sqlserver.go            ← Read-only SQL Server adapter
```

---

## Intent Router

| Intent | Trigger Keywords | Action |
|---|---|---|
| `rag` | นโยบาย, ประกาศ, เอกสาร, OT, ล่วงเวลา, ลา ฯลฯ | ค้นหาจาก pgvector |
| `mcp` | ยอดขาย, สต็อก, รายงาน, เงินเดือน ฯลฯ | Query SQL Server |
| `direct` | ไม่เจอ keyword | ตอบตรงๆ |

**TODO:** Phase 3.5 — LLM-based classifier

---

## Session Management

```
Sliding Window: ดึง 10 messages ล่าสุด
Session Expiry: 30 นาที
Storage: PostgreSQL (sessions + messages tables)
metadata JSONB: เก็บ intent + RAG sources

Flow:
JWT → user_id → Session (create/restore)
→ GetHistory(limit=10) → LLM
→ SaveMessage (async goroutine)
→ X-Session-ID header
```
