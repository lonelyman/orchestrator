# Python Rebuild Spec — FastAPI + LangGraph
> **Created:** 2026-04-30
> **Purpose:** สเปกสำหรับ rebuild จาก PoC Go เป็น Python stack โดยไม่ตกหล่น feature ที่ทำไปแล้ว
> **Source:** `10-GREENFIELD-STACK.md`, `11-REBUILD-CHECKLIST.md`

---

## 1. Goal

สร้างระบบใหม่ที่ feature parity กับ Go PoC เดิม แต่ใช้ stack ที่เหมาะกับ Enterprise AI มากขึ้น:

- FastAPI เป็น API service
- LangGraph เป็น agent orchestration
- LiteLLM/OpenAI-compatible client เป็น LLM gateway
- PostgreSQL + pgvector เป็น data/vector store ระยะแรก
- Redis สำหรับ queue/cache/rate limit ระยะถัดไป
- Keycloak/OIDC เป็น auth target ระยะ production

เป้าหมายไม่ใช่ “เขียนใหม่ให้เหมือนเดิมทุกบรรทัด” แต่ต้องรักษา behavior สำคัญที่ PoC แก้ปัญหาไว้แล้ว

---

## 2. Non-Negotiable Feature Parity

ระบบใหม่ต้องมีรายการนี้ก่อนถือว่าแทน PoC เดิมได้:

- [ ] OpenAI-compatible `/v1/chat/completions`
- [ ] Streaming chat
- [ ] `/v1/models`
- [ ] JWT/OIDC protected API
- [ ] Dev auth mode สำหรับ local development
- [ ] Session management + `X-Session-ID`
- [ ] Conversation history แบบ ordered deterministic
- [ ] Save user + assistant messages แบบ atomic
- [ ] `/live`, `/ready`, `/health` semantics แยกชัด
- [ ] PDF upload + text extraction
- [ ] Scanned PDF OCR ด้วย AI OCR
- [ ] RAG ingest/search
- [ ] RAG ไม่ fallback เป็นเอกสารมั่วเมื่อ search ไม่เจอ
- [ ] Document list/delete API
- [ ] Audit log แบบ async queue + shutdown drain
- [ ] Security headers + CORS default deny
- [ ] Config validation fail fast
- [ ] Prompt file management
- [ ] LLM backend switch Ollama → vLLM ผ่าน config
- [ ] Tool calling contract
- [ ] Agent loop max iteration + timeout + allowlist
- [ ] `rag_search` tool
- [ ] `web_search` tool disabled-by-default
- [ ] Migration discipline
- [ ] Docker build/run
- [ ] Unit/integration/eval test baseline

---

## 3. Recommended Repo Structure

```text
app-api/
├── pyproject.toml
├── Dockerfile
├── docker-compose.yml
├── alembic.ini
├── src/
│   └── app_api/
│       ├── main.py
│       ├── api/
│       │   ├── routes/
│       │   │   ├── auth.py
│       │   │   ├── chat.py
│       │   │   ├── health.py
│       │   │   ├── rag.py
│       │   │   ├── documents.py
│       │   │   └── admin.py
│       │   └── middleware/
│       │       ├── request_id.py
│       │       ├── security.py
│       │       ├── auth.py
│       │       ├── session.py
│       │       ├── audit.py
│       │       └── errors.py
│       ├── domain/
│       │   ├── models.py
│       │   └── ports.py
│       ├── core/
│       │   ├── config.py
│       │   ├── prompts.py
│       │   ├── intent.py
│       │   ├── rag.py
│       │   └── agent/
│       │       ├── graph.py
│       │       ├── state.py
│       │       ├── registry.py
│       │       └── tools/
│       │           ├── rag_search.py
│       │           ├── web_search.py
│       │           └── analytics.py
│       ├── adapters/
│       │   ├── llm/
│       │   │   ├── openai_compatible.py
│       │   │   └── litellm_client.py
│       │   ├── vector/
│       │   │   └── pgvector.py
│       │   ├── db/
│       │   │   ├── session.py
│       │   │   ├── audit.py
│       │   │   └── migrations.py
│       │   ├── auth/
│       │   │   ├── dev.py
│       │   │   ├── ldap.py
│       │   │   └── oidc.py
│       │   ├── search/
│       │   │   └── tavily.py
│       │   └── ocr/
│       │       └── ollama_ocr.py
│       └── workers/
│           ├── audit_worker.py
│           └── ingest_worker.py
├── alembic/
│   └── versions/
├── prompts/
│   ├── system_general.j2
│   ├── system_rag.j2
│   └── system_web.j2
└── tests/
    ├── unit/
    ├── integration/
    └── eval/
```

---

## 4. Domain Contracts

### LLM Port

ต้องรองรับทั้ง chat ธรรมดา, streaming, tool calling:

```python
from typing import Protocol, AsyncIterator

class LLMPort(Protocol):
    async def chat(self, req: LLMChatRequest) -> LLMChatResponse: ...
    async def chat_stream(self, req: LLMChatRequest) -> AsyncIterator[StreamEvent]: ...
    async def health_check(self) -> None: ...
```

### Tool Registry

```python
class ToolExecutor(Protocol):
    async def __call__(self, ctx: ToolContext, args: dict) -> ToolResult: ...

class ToolRegistry:
    def register(self, tool: ToolDefinition, executor: ToolExecutor) -> None: ...
    def definitions(self, names: list[str] | None = None) -> list[ToolDefinition]: ...
    async def execute(self, call: ToolCall) -> ToolResult: ...
```

ต้องมี guardrail:

- [ ] allowlist tool names
- [ ] timeout ต่อ tool
- [ ] max result size
- [ ] structured error เมื่อ tool fail
- [ ] log tool usage

### Web Search Port

```python
class WebSearchPort(Protocol):
    async def search(self, query: str, opts: WebSearchOptions) -> list[WebSearchResult]: ...
    async def health_check(self) -> None: ...
```

### Vector Store Port

```python
class VectorStorePort(Protocol):
    async def store(self, doc: Document) -> None: ...
    async def search(self, embedding: list[float], limit: int) -> list[Document]: ...
    async def delete_source(self, source: str) -> int: ...
    async def list_documents(self) -> list[DocumentSummary]: ...
```

---

## 5. API Contract

### Public / Health

- `GET /live`
- `GET /ready`
- `GET /health`
- `GET /v1/models`
- `POST /auth/login` หรือ OIDC callback equivalent

### Protected

- `GET /auth/me`
- `POST /v1/chat/completions`
- `POST /v1/rag/upload`
- `POST /v1/rag/ingest`
- `GET /v1/rag/documents`
- `DELETE /v1/rag/documents/{source}`
- `DELETE /v1/rag/documents?source=...`

### Admin

- `GET /v1/admin/logs`

---

## 6. Config Spec

ใช้ Pydantic Settings และ validate ตอน startup

### Required Groups

- Server/runtime
- Security/CORS
- DB/pool
- LLM
- Embedder
- OCR
- Web search
- Auth
- Prompt

### Important Defaults

```env
WEB_SEARCH_ENABLED=false
WEB_SEARCH_PROVIDER=tavily
WEB_SEARCH_MAX_RESULTS=5
ALLOWED_ORIGINS=
CORS_ALLOW_CREDENTIALS=false
DEV_MODE=true
```

Production mode ต้อง reject:

- [ ] default DB password
- [ ] weak JWT secret ถ้ายังใช้ JWT local
- [ ] missing OIDC issuer/client config
- [ ] AD placeholder
- [ ] web search enabled without API key
- [ ] wildcard CORS with credentials

---

## 7. Agent Flow

### Non-Streaming Phase 1

```text
request
  -> validate
  -> load session history
  -> classify intent
  -> select tool definitions
  -> LangGraph / agent loop
      -> LLM decides tool call
      -> execute tool
      -> append tool result
      -> final answer
  -> save messages atomically
  -> audit
```

### Limits

- max iterations: `3`
- tool timeout: `15s`
- tool result max chars: `8000`
- fallback for RAG if model does not call tool

### Required Tools

- [ ] `rag_search(query, limit)`
- [ ] `web_search(query, max_results, topic)`

### Later Tools

- [ ] `analytics_query(metric, filters, dimensions, time_range)`
- [ ] `sales_query(query_name, parameters)`
- [ ] `get_current_time(timezone)`

---

## 8. RAG Pipeline

### Upload Path

```text
PDF upload
  -> validate size/type
  -> pdftotext
  -> if text empty: pdftoppm -> AI OCR
  -> normalize text
  -> chunk
  -> embed
  -> store pgvector
```

### Required Guards

- [ ] max upload size
- [ ] max OCR pages
- [ ] timeout
- [ ] reject unsupported type
- [ ] no irrelevant fallback
- [ ] document source metadata
- [ ] list/delete documents

### Later Improvements

- [ ] better chunking
- [ ] metadata tags
- [ ] hybrid search
- [ ] reranker
- [ ] document versioning
- [ ] soft delete

---

## 9. Audit and Logging

### Request Log

ต้องมี:

- request_id
- method/path/status
- latency
- ip
- user_id
- session_id

### Audit Log

ต้องเก็บ:

- request_id
- user_id/username
- session_id
- method/path
- intent
- query
- response preview
- latency/status/ip
- created_at

### Worker Rule

- [ ] audit write ต้องผ่าน queue
- [ ] queue full policy ต้องชัด
- [ ] shutdown ต้อง drain
- [ ] admin log endpoint ต้อง filter/paginate

---

## 10. Database Schema Baseline

Minimum tables:

- users หรือ external identity mapping
- sessions
- messages
- documents
- audit_logs
- revoked_tokens ถ้ายังใช้ local JWT
- auth_attempts ถ้าทำ lockout

Important indexes:

- `messages(session_id, sequence_number)`
- `documents` vector index
- `audit_logs(created_at)`
- `audit_logs(user_id)`
- `audit_logs(request_id)`

---

## 11. Testing Plan

### Unit Tests

- config validation
- intent classifier
- prompt loading
- tool registry
- RAG tool
- web search tool
- Tavily adapter
- LLM adapter tool call parsing
- request validation

### Integration Tests

- migration run
- DB session save atomic
- RAG ingest/search
- document list/delete
- health endpoints

### Eval Tests

ต้องมี eval dataset อย่างน้อย:

- RAG answer found
- RAG answer not found
- Thai OCR text
- web search requires source citation
- tool disabled response
- sales/analytics permission denied

---

## 12. Cutover Criteria

ก่อนย้ายจาก Go PoC ไป Python rebuild:

- [ ] API contract ใช้กับ WebUI เดิมได้
- [ ] migration/schema พร้อม
- [ ] auth flow พร้อม
- [ ] RAG upload/search เทียบผลกับ PoC ได้
- [ ] OCR ใช้งานได้จริง
- [ ] agent loop ไม่ runaway
- [ ] web search disabled-by-default
- [ ] audit logs ครบ
- [ ] load test ผ่านสำหรับ target users
- [ ] backup/restore ทดสอบแล้ว
- [ ] runbook พร้อม

---

## 13. Things To Avoid

- อย่าให้ LLM ยิง SQL raw ใน production
- อย่าเปิด CORS wildcard พร้อม credentials
- อย่า log secrets
- อย่าใช้ `/health` หนักๆ เป็น readiness probe
- อย่าทำ RAG fallback เป็นเอกสารล่าสุด
- อย่า hard-code Ollama endpoint ใน core logic
- อย่าให้ agent loop ไม่มี max iteration
- อย่าเปิด web search โดยไม่มี API budget/cost control
- อย่าเก็บ prompt แค่ใน env

---

## 14. Recommended First Sprint

ถ้าจะเริ่ม Python rebuild จริง Sprint แรกควรทำแค่นี้:

1. FastAPI skeleton
2. Pydantic Settings
3. `/live`, `/ready`, `/health`
4. Request ID + structured log
5. Alembic baseline
6. OpenAI-compatible LLM adapter
7. Simple `/v1/chat/completions`
8. Docker compose
9. CI: lint + test + build

ยังไม่ควรทำ LangGraph/RAG ในวันแรก จนกว่า foundation ข้างบนจะนิ่ง

