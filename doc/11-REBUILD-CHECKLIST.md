# Rebuild Checklist — What This PoC Already Solved
> **Created:** 2026-04-30
> **Purpose:** ใช้เป็น checklist เวลา rebuild / greenfield เพื่อไม่ลืมสิ่งที่ PoC Go ทำไว้แล้ว
> **Scope:** สรุปเฉพาะสิ่งที่ implement แล้ว + สิ่งที่ต้องระวังถ้าทำใหม่

---

## วิธีใช้เอกสารนี้

ถ้าจะเริ่มระบบใหม่ด้วย Python/FastAPI/LangGraph หรือ Hybrid ให้ใช้เอกสารนี้เป็น checklist:

1. เช็คว่า feature เดิมมีครบหรือไม่
2. เช็ค behavior สำคัญที่ห้ามหาย
3. เช็ค config/env ที่ต้องรองรับ
4. เช็ค production guardrail ที่ทำไว้แล้ว
5. เช็คข้อควรระวังที่ PoC เจอแล้ว

---

## 1. Core Application Foundation

### ทำไปแล้ว
- [x] แยก bootstrap ออกจาก router: `cmd/server/main.go`, `internal/app/server.go`, `internal/api/router.go`
- [x] โครงสร้างใกล้ hexagonal architecture:
  - `domain` = models/ports
  - `core` = orchestration/RAG/intent
  - `infrastructure` = adapters DB/LLM/OCR/Auth
- [x] Router แยกจาก `main.go`
- [x] Fiber app มี body limit จาก config
- [x] JSON structured logging ด้วย `slog`
- [x] Request ID middleware:
  - รับ `X-Request-ID`
  - generate UUID ถ้าไม่มี
  - ส่งกลับใน response header
- [x] Recovery middleware กัน panic แล้วตอบ `500 SERVER_ERROR`
- [x] Request log middleware:
  - `request_id`
  - method/path/status
  - latency
  - ip
  - user_id/session_id ถ้ามี
- [x] Redaction helper สำหรับ log ที่มี token/password/authorization

### ถ้า rebuild ต้องไม่ลืม
- [ ] แยก `main` ให้สะอาด อย่าให้ wiring, routes, business logic อยู่รวมกัน
- [ ] ทุก request ต้องมี request id ตั้งแต่ middleware แรก
- [ ] ทุก log ที่เกี่ยวกับ request ต้องพ่วง `request_id`
- [ ] panic recovery ต้องไม่ leak stack trace หรือ secret กลับ user

### ระวัง
- อย่า log `Authorization`, JWT, password, API key
- ถ้าใช้ framework ใหม่ ต้องเช็คว่า context/request lifecycle ไม่หมดก่อน background task ทำงาน

---

## 2. Configuration / Environment

### ทำไปแล้ว
- [x] Central config loader ใน `config/config.go`
- [x] Validate config ตอน startup
- [x] Dev mode แยกจาก production mode
- [x] Reject weak production config:
  - weak `JWT_SECRET`
  - default `DB_PASS`
  - AD placeholder
- [x] Duration config:
  - `SESSION_EXPIRY`
  - `CHAT_TIMEOUT`
  - `RAG_INGEST_TIMEOUT`
  - `AUDIT_TIMEOUT`
  - `HEALTH_TIMEOUT`
  - `SHUTDOWN_TIMEOUT`
  - `MIGRATION_TIMEOUT`
  - `OCR_TIMEOUT`
  - `WEB_SEARCH_TIMEOUT`
- [x] Limit config:
  - `RATE_LIMIT_PER_MINUTE`
  - `MAX_UPLOAD_MB`
  - `OCR_MAX_PAGES`
  - `WEB_SEARCH_MAX_RESULTS`
- [x] Config tests ครอบ invalid ports, invalid limits, weak prod config, prompt file, web search config

### ถ้า rebuild ต้องไม่ลืม
- [ ] ใช้ typed settings เช่น Pydantic Settings ถ้าเป็น Python
- [ ] startup ต้อง fail fast ถ้า config production ไม่ปลอดภัย
- [ ] `.env.example` ต้องเป็น source of truth สำหรับ env ทั้งหมด
- [ ] secret จริงห้าม commit

### ระวัง
- Default ที่เหมาะกับ dev อาจไม่เหมาะกับ prod
- Env ที่เป็น timeout/limit ต้อง parse และ validate ไม่ใช่ปล่อย string ดิบ

---

## 3. Docker / Runtime

### ทำไปแล้ว
- [x] Docker multi-stage build
- [x] Runtime image ไม่ bundle Tesseract แล้ว
- [x] ยังเก็บ `poppler-utils` เพราะต้องใช้:
  - `pdftotext`
  - `pdftoppm`
- [x] Copy `prompts/` เข้า image
- [x] Docker compose override host สำหรับ Ollama service:
  - `LLM_HOST=host.docker.internal`
  - `EMBED_HOST=host.docker.internal`
  - `OCR_HOST=host.docker.internal`
- [x] PostgreSQL container พร้อม healthcheck

### ถ้า rebuild ต้องไม่ลืม
- [ ] Docker image ต้องมี binary/tool ที่ parser ต้องใช้จริง
- [ ] อย่าใส่ OCR engine ที่ไม่ได้ใช้ใน production image
- [ ] prompts/templates ต้องถูก copy เข้า image
- [ ] compose dev ต้องเชื่อม Ollama/vLLM host ได้

### ระวัง
- ถ้าเปลี่ยนจาก Ollama เป็น vLLM ต้องไม่ hard-code `/api/chat`
- `poppler-utils` ยังจำเป็นแม้ใช้ AI OCR เพราะต้อง render PDF เป็น image

---

## 4. Database / Migration

### ทำไปแล้ว
- [x] PostgreSQL + pgvector
- [x] Embedded goose migrations
- [x] Migration path fix แล้ว:
  - ใช้ embedded FS subdir ถูกต้อง
  - rebuild container แล้ว migration สร้างตารางกลับมาได้
- [x] Dev DB reset เป็น clean baseline เพราะ DB ยังว่าง
- [x] Baseline schema includes:
  - sessions
  - messages
  - documents
  - audit_logs
  - goose_db_version
- [x] HNSW index สำหรับ documents embedding
- [x] `messages.sequence_number` สำหรับ order conversation history
- [x] DB pool tuning:
  - `DB_POOL_MAX_CONNS`
  - `DB_POOL_MIN_CONNS`
  - `DB_POOL_MAX_CONN_LIFETIME`
  - `DB_POOL_MAX_CONN_IDLE_TIME`
  - `DB_POOL_HEALTH_CHECK_PERIOD`

### ถ้า rebuild ต้องไม่ลืม
- [ ] Migration ต้อง run ตอน startup หรือใน deploy job ที่ชัดเจน
- [ ] Migration ต้อง embedded/package ไปกับ artifact
- [ ] ห้ามแก้ migration เก่าหลังมี production data
- [ ] ต้องมี test ว่า migration files ถูก embed/package จริง
- [ ] message history ต้องมี ordering deterministic

### ระวัง
- ถ้าใช้ Alembic แทน goose ต้องมี process ชัดว่า migration run ที่ไหน
- ถ้า DB ยังว่าง reset baseline ได้ แต่หลัง prod ต้อง append-only migration
- pgvector ใช้ได้ดีตอนเริ่ม แต่ถ้า vectors โตมากอาจต้อง Qdrant

---

## 5. Health / Readiness / Shutdown

### ทำไปแล้ว
- [x] `/live` = process alive
- [x] `/ready` = DB connectable เท่านั้น
- [x] `/health` = full dependency status:
  - DB
  - LLM
  - Embedder
- [x] Health timeout ผ่าน config
- [x] Graceful shutdown v2:
  - รับ SIGTERM/SIGINT
  - stop accepting requests
  - drain HTTP ด้วย `SHUTDOWN_TIMEOUT`
  - close audit worker pool
  - close DB pool
- [x] Listen error trigger shutdown path
- [x] ทดสอบ SIGTERM container แล้ว stop cleanly

### ถ้า rebuild ต้องไม่ลืม
- [ ] แยก liveness/readiness/health ให้ชัด
- [ ] Kubernetes/readiness probe ไม่ควรเช็ค LLM หนักๆ
- [ ] Shutdown ต้อง drain queue/background worker
- [ ] Listen/server startup error ต้องทำให้ process exit non-zero

### ระวัง
- `/health` ที่เช็ค LLM/embedder อาจช้า ไม่ควรใช้เป็น readiness probe
- Background worker ถ้าไม่ drain อาจทำ audit/log หาย

---

## 6. Auth / Session / RBAC

### ทำไปแล้ว
- [x] LDAP adapter
- [x] Dev mode bypass AD
- [x] JWT manager
- [x] JWT middleware
- [x] Session middleware:
  - create session ถ้าไม่มี
  - validate owner
  - return `X-Session-ID`
- [x] Session history sliding window
- [x] Save user + assistant messages ใน transaction เดียว
- [x] RBAC middleware มี role guard สำหรับ admin logs

### ถ้า rebuild ต้องไม่ลืม
- [ ] Auth ต้องมี dev mode แยกชัด และปิดใน prod
- [ ] Session owner mismatch ต้องสร้างใหม่หรือ reject ห้าม reuse ข้าม user
- [ ] Conversation history ต้องจำกัดจำนวน message เพื่อคุม token
- [ ] Admin endpoint ต้องมี role guard

### ยังต้องทำ / policy needed
- [ ] LDAP TLS hardening จริง:
  - CA file
  - StartTLS fail hard เมื่อ `DEV_MODE=false`
  - `LDAP_TLS_SKIP_VERIFY=false` by default
- [ ] JWT audience/subject/jti
- [ ] Token revocation + logout
- [ ] Refresh token rotation
- [ ] Login brute-force protection / lockout

### ระวัง
- AD/LDAP policy ต้องถาม infra ก่อน อย่าเดา
- JWT expiry/refresh policy ต้องให้ business/security ตัดสิน

---

## 7. Security Headers / CORS / Rate Limit

### ทำไปแล้ว
- [x] Helmet/security headers:
  - `X-Frame-Options=DENY`
  - `X-Content-Type-Options=nosniff`
  - Referrer policy
  - CSP
  - Permissions policy
- [x] CORS default deny:
  - `ALLOWED_ORIGINS=`
  - `CORS_ALLOW_CREDENTIALS=false`
- [x] Validate invalid CORS origin
- [x] Reject wildcard origin with credentials
- [x] Rate limiter per user_id ถ้ามี claims, fallback IP

### ถ้า rebuild ต้องไม่ลืม
- [ ] Default CORS ต้อง deny ไม่ใช่ wildcard
- [ ] Web UI domain ต้องตั้งใน env
- [ ] CSP ต้องเข้ากับ frontend จริง
- [ ] Rate limit key ควรเป็น user_id ไม่ใช่ IP อย่างเดียว

### ระวัง
- ถ้าเปิด `ALLOWED_ORIGINS=*` พร้อม credentials จะเป็นช่องโหว่
- CSP เข้มเกินไปอาจทำ WebUI พัง ต้อง test browser

---

## 8. Audit / Compliance

### ทำไปแล้ว
- [x] Audit log table
- [x] Audit middleware เก็บ:
  - request_id
  - user_id/username
  - session_id
  - method/path
  - intent
  - query
  - response preview
  - latency/status/ip
- [x] Audit worker pool:
  - `AUDIT_WORKERS`
  - `AUDIT_QUEUE_SIZE`
  - queue full = drop + warning + counter
  - drain ตอน shutdown
- [x] Admin audit log endpoint

### ถ้า rebuild ต้องไม่ลืม
- [ ] Audit write ห้าม block request นาน
- [ ] ต้องมี queue/backpressure policy
- [ ] ต้อง drain queue ตอน shutdown
- [ ] ต้อง redact sensitive data ก่อนลง log/audit
- [ ] ต้องมี admin endpoint/filter สำหรับ audit

### ระวัง
- Drop policy ต้องตกลงกับ compliance ถ้า audit ห้ามหายอาจต้อง block หรือ persistent queue
- Query/response preview อาจมีข้อมูลส่วนบุคคล ต้องกำหนด retention

---

## 9. LLM Adapter / Ollama → vLLM

### ทำไปแล้ว
- [x] `LLMPort`
- [x] Ollama adapter
- [x] OpenAI-compatible adapter สำหรับ vLLM
- [x] Config:
  - `LLM_BACKEND=ollama|vllm|openai-compatible`
  - `LLM_HOST`
  - `LLM_PORT`
  - `LLM_MODEL`
  - `LLM_API_KEY`
- [x] `Chat`
- [x] `ChatStream`
- [x] `HealthCheck`
- [x] Tool calling foundation:
  - `LLMChatRequest`
  - `LLMChatResponse`
  - `ToolCalls`
  - `TokenUsage`
- [x] Ollama tool calling parse/send
- [x] OpenAI/vLLM style tool calling parse/send

### ถ้า rebuild ต้องไม่ลืม
- [ ] LLM interface ต้องไม่ผูก Ollama
- [ ] vLLM ต้องใช้ OpenAI-compatible API เป็น primary path
- [ ] Tool calling contract ต้องเป็นกลาง
- [ ] Streaming กับ non-streaming ต้อง test แยก

### ระวัง
- Ollama กับ OpenAI-compatible tool call format ไม่เหมือนกัน
- บาง local model อาจไม่เรียก tool แม้ส่ง tool definition ไปแล้ว ต้องมี fallback
- อย่า hard-code model name ใน business logic

---

## 10. Prompt Management

### ทำไปแล้ว
- [x] `SYSTEM_PROMPT_FILE`
- [x] `SYSTEM_PROMPT` fallback
- [x] Default system prompt
- [x] Prompt file included in Docker image
- [x] Test:
  - load prompt from file
  - reject missing file
- [x] `prompts/system_general.tmpl`

### ถ้า rebuild ต้องไม่ลืม
- [ ] Prompt ต้อง version-controlled
- [ ] Prompt file ต้อง package เข้า image
- [ ] Missing prompt file ต้อง fail fast
- [ ] แยก prompt ตาม use case:
  - general
  - RAG
  - web search
  - analytics
  - sales

### ระวัง
- Prompt ใน env ยาวๆ maintain ยาก
- Prompt registry/admin UI ยังไม่มี

---

## 11. RAG / Document Processing

### ทำไปแล้ว
- [x] PDF upload endpoint
- [x] Text extraction ด้วย `pdftotext`
- [x] PDF-to-image rendering ด้วย `pdftoppm`
- [x] AI OCR via Ollama:
  - `OCR_ENGINE=ollama`
  - `OCR_MODEL=scb10x/typhoon-ocr1.5-3b:latest`
- [x] OCR disabled mode
- [x] Tesseract code path ยัง optional แต่ Docker ไม่ bundle แล้ว
- [x] Max upload size
- [x] OCR max pages
- [x] RAG ingest timeout
- [x] Embedder adapter
- [x] pgvector document store
- [x] Search no longer falls back to newest irrelevant docs
- [x] Document management endpoints:
  - `GET /v1/rag/documents`
  - `DELETE /v1/rag/documents/:source`
  - `DELETE /v1/rag/documents?source=...`

### ถ้า rebuild ต้องไม่ลืม
- [ ] PDF text extraction ก่อน OCR เพื่อลด cost
- [ ] OCR ต้องมี max pages
- [ ] Upload ต้องมี size limit
- [ ] RAG search ห้าม fallback เป็นเอกสารล่าสุดแบบมั่ว
- [ ] Document delete/list ต้องมี API ตั้งแต่ต้น
- [ ] Chunking strategy ต้องชัด

### ระวัง
- OCR ผ่าน LLM ช้าและใช้ resource สูง
- ถ้าไม่มี `poppler-utils` pipeline PDF จะพัง
- RAG quality ยังต้องปรับ:
  - chunking
  - metadata
  - rerank
  - hybrid search

---

## 12. Chat API / Validation / OpenAI Compatibility

### ทำไปแล้ว
- [x] `/v1/chat/completions`
- [x] OpenAI-like request/response
- [x] Streaming response
- [x] `/v1/models`
- [x] Request validation:
  - model default
  - reject unsupported model
  - messages required
  - max 64 messages
  - max 20,000 chars/message
  - max 60,000 chars total
  - supported roles: system/user/assistant
  - last message must be user
- [x] Chat timeout
- [x] Audit query/response preview

### ถ้า rebuild ต้องไม่ลืม
- [ ] Validate request ก่อนเรียก LLM
- [ ] จำกัดข้อความและ token/cost
- [ ] Streaming ต้อง propagate context/cancel
- [ ] API shape ถ้าใช้ OpenAI-compatible ควรรักษาให้ WebUI ใช้ได้

### ระวัง
- Streaming agent loop ยังไม่ได้ทำใน PoC
- ถ้ารองรับ tool calls ใน API public ต้องออกแบบ response schema เพิ่ม

---

## 13. Intent Router / Agent Loop / Tools

### ทำไปแล้ว
- [x] Rule-based intent classifier:
  - RAG
  - MCP
  - Web Search
  - Direct
- [x] MCP intent placeholder
- [x] Tool registry
- [x] Agent loop สำหรับ non-streaming:
  - max 3 iterations
  - execute tool calls
  - append tool result กลับให้ LLM
  - final summarization
- [x] `rag_search(query, limit)` tool
- [x] `web_search(query, max_results, topic)` tool
- [x] RAG fallback ถ้า model ไม่เรียก tool
- [x] Web search disabled message ถ้ายังไม่ได้เปิด env

### ถ้า rebuild ต้องไม่ลืม
- [ ] Tool registry ต้องมี allowlist
- [ ] Tool execution ต้อง timeout
- [ ] Tool result ต้อง truncate
- [ ] Unknown tool ต้อง reject
- [ ] Agent loop ต้องมี max iterations
- [ ] Tool usage ต้อง log/audit

### ระวัง
- Rule-based classifier ยังหยาบ
- คำว่า "วันนี้" อาจหมายถึง web หรือ internal DB ต้องให้ internal keywords ชนะก่อน
- Model บางตัวอาจ hallucinate tool args
- Streaming agent ยังไม่ได้ทำ

---

## 14. Web Search

### ทำไปแล้ว
- [x] `WebSearchPort`
- [x] Tavily adapter
- [x] Config:
  - `WEB_SEARCH_ENABLED`
  - `WEB_SEARCH_PROVIDER=tavily`
  - `WEB_SEARCH_API_KEY`
  - `WEB_SEARCH_BASE_URL`
  - `WEB_SEARCH_TIMEOUT`
  - `WEB_SEARCH_MAX_RESULTS`
- [x] Register `web_search` tool เฉพาะเมื่อ enabled
- [x] ถ้า disabled ตอบชัดเจน ไม่ call LLM/tool
- [x] Tests สำหรับ adapter/config/intent/tool

### ถ้า rebuild ต้องไม่ลืม
- [ ] Web search provider ต้องอยู่หลัง port/interface
- [ ] API key ต้องอยู่ใน secret/env
- [ ] Result ต้องมี URL/source เพื่อ cite
- [ ] ต้องมี disabled mode
- [ ] ต้องมี timeout

### ยังต้องทำ
- [ ] Cache search result 1 ชม.
- [ ] Provider อื่น:
  - Brave
  - Serper
  - SearXNG
- [ ] Recency/region mapping จริง

### ระวัง
- Web search มี cost ต่อ query
- ข้อมูลเว็บอาจผิด ต้องให้ LLM cite source และระบุเมื่อไม่พบ
- Tavily API key ห้าม commit

---

## 15. MCP / Analytics / Sales DB

### ทำไปแล้ว
- [x] `MCPPort`
- [x] MCP executor skeleton
- [x] SQL Server adapter structure
- [x] Read-only validation concept
- [x] Intent MCP placeholder

### ถ้า rebuild ต้องไม่ลืม
- [ ] อย่าให้ LLM ยิง SQL raw โดยตรงใน production
- [ ] ต้องมี semantic layer / metric catalog
- [ ] Tool SQL ต้อง read-only และ parameterized
- [ ] ต้องมี allowlist table/view
- [ ] ต้องมี query timeout/row limit

### ยังต้องทำ
- [ ] AnalyticsPort
- [ ] Metric definitions
- [ ] SQL templates
- [ ] Sales DB connection policy
- [ ] Executive insight flow

### ระวัง
- คำถามผู้บริหารต้องการ business meaning ไม่ใช่ SQL ตรงๆ
- LLM hallucinate table/column ได้ง่าย
- Data permission ตาม department/role ต้องชัด

---

## 16. Tests / Quality Gates

### ทำไปแล้ว
- [x] Unit tests หลายส่วน:
  - config
  - middleware
  - handlers
  - LLM adapters
  - OCR helper
  - migrations embed
  - audit async worker
  - orchestrator agent loop
  - Tavily adapter
- [x] Manual verification หลังงานสำคัญ:
  - `go test ./...`
  - `go vet ./...`
  - `go build`
  - `docker compose build orchestrator`
  - `/ready`
  - `/health`
  - SIGTERM graceful shutdown

### ถ้า rebuild ต้องไม่ลืม
- [ ] มี unit tests สำหรับ ports/adapters
- [ ] มี integration tests สำหรับ DB/migration
- [ ] มี eval tests สำหรับ RAG/agent answers
- [ ] CI ต้องรัน lint/test/build
- [ ] Docker image ต้อง build ใน CI

### ระวัง
- LLM behavior ต้องมี eval ไม่ใช่ unit test อย่างเดียว
- Mock LLM ช่วย test flow แต่ไม่แทน real model testing

---

## 17. Current Env Checklist

ถ้า rebuild ต้องมี env กลุ่มนี้หรือ equivalent:

### Server / Runtime
- [ ] `API_PORT`
- [ ] `RATE_LIMIT_PER_MINUTE`
- [ ] `SESSION_EXPIRY`
- [ ] `CHAT_TIMEOUT`
- [ ] `RAG_INGEST_TIMEOUT`
- [ ] `AUDIT_TIMEOUT`
- [ ] `AUDIT_WORKERS`
- [ ] `AUDIT_QUEUE_SIZE`
- [ ] `HEALTH_TIMEOUT`
- [ ] `SHUTDOWN_TIMEOUT`
- [ ] `MIGRATION_TIMEOUT`
- [ ] `MAX_UPLOAD_MB`

### Security
- [ ] `ALLOWED_ORIGINS`
- [ ] `CORS_ALLOW_CREDENTIALS`
- [ ] `CSP_POLICY`
- [ ] `JWT_SECRET`
- [ ] `JWT_EXPIRY`

### LLM / Embedding / OCR
- [ ] `LLM_BACKEND`
- [ ] `LLM_HOST`
- [ ] `LLM_PORT`
- [ ] `LLM_MODEL`
- [ ] `LLM_API_KEY`
- [ ] `EMBED_HOST`
- [ ] `EMBED_PORT`
- [ ] `EMBED_MODEL`
- [ ] `OCR_ENGINE`
- [ ] `OCR_HOST`
- [ ] `OCR_PORT`
- [ ] `OCR_MODEL`
- [ ] `OCR_TIMEOUT`
- [ ] `OCR_MAX_PAGES`
- [ ] `OCR_PROMPT`

### Web Search
- [ ] `WEB_SEARCH_ENABLED`
- [ ] `WEB_SEARCH_PROVIDER`
- [ ] `WEB_SEARCH_API_KEY`
- [ ] `WEB_SEARCH_BASE_URL`
- [ ] `WEB_SEARCH_TIMEOUT`
- [ ] `WEB_SEARCH_MAX_RESULTS`

### Database
- [ ] `DB_HOST`
- [ ] `DB_PORT`
- [ ] `DB_NAME`
- [ ] `DB_USER`
- [ ] `DB_PASS`
- [ ] `DB_POOL_MAX_CONNS`
- [ ] `DB_POOL_MIN_CONNS`
- [ ] `DB_POOL_MAX_CONN_LIFETIME`
- [ ] `DB_POOL_MAX_CONN_IDLE_TIME`
- [ ] `DB_POOL_HEALTH_CHECK_PERIOD`

### Auth / Dev
- [ ] `AD_SERVER`
- [ ] `AD_PORT`
- [ ] `AD_BASE_DN`
- [ ] `AD_DOMAIN`
- [ ] `DEV_MODE`
- [ ] `DEV_USERNAME`
- [ ] `DEV_PASSWORD`

### Prompt
- [ ] `SYSTEM_PROMPT_FILE`
- [ ] `SYSTEM_PROMPT`

---

## 18. Rebuild Minimum Feature Parity Checklist

ถ้าจะบอกว่า rebuild ใหม่ feature parity กับ PoC เดิม ต้องผ่าน checklist นี้:

- [ ] Login dev mode ได้
- [ ] JWT auth ใช้กับ protected API ได้
- [ ] Session create/reuse ได้
- [ ] Chat non-stream ได้
- [ ] Chat stream ได้
- [ ] Conversation history ถูก save เป็น user + assistant atomic
- [ ] `/live`, `/ready`, `/health` semantics เหมือนเดิม
- [ ] Upload PDF แล้ว extract text ได้
- [ ] Scanned PDF ผ่าน AI OCR ได้
- [ ] Ingest เอกสารเข้า vector store ได้
- [ ] RAG search ไม่ตอบมั่วจากเอกสารที่ไม่เกี่ยวข้อง
- [ ] List/delete RAG documents ได้
- [ ] Audit log ถูกเขียนผ่าน queue/worker
- [ ] Graceful shutdown drain worker ได้
- [ ] Security headers/CORS default deny
- [ ] LLM backend เปลี่ยน Ollama → vLLM ผ่าน config ได้
- [ ] Tool calling contract รองรับ OpenAI-compatible
- [ ] Agent loop มี max iteration/timeout
- [ ] `rag_search` tool ใช้งานได้
- [ ] `web_search` tool disabled-by-default และเปิดด้วย env ได้
- [ ] Config validation fail fast
- [ ] Docker image build/run ได้
- [ ] Migration package/run ได้
- [ ] Tests + CI gates ครบ

---

## 19. Important Tradeoffs Learned

### Go PoC ดีตรงไหน
- Fast, simple deploy, single binary
- Good for API gateway, auth, audit, session, DB access
- Type safety ดี
- Resource footprint ต่ำ

### Go PoC เสียเปรียบตรงไหน
- AI/agent ecosystem เล็กกว่า Python
- Agent loop/tooling ต้องเขียนเองเยอะ
- Eval/prompt/LLM observability ecosystem น้อยกว่า
- Hiring AI engineer ที่ถนัด Go ยากกว่า

### ถ้าเริ่มใหม่
- Python + FastAPI + LangGraph เหมาะกับ AI logic
- Go ยังเหมาะกับ gateway/security/audit service
- Hybrid ใช้ได้ถ้าต้องการเก็บ Go foundation

---

## 20. Biggest Risks ถ้าทำใหม่

- [ ] ลืม security hardening เพราะรีบทำ AI feature
- [ ] ไม่มี migration discipline ตั้งแต่แรก
- [ ] ทำ RAG แล้ว fallback มั่วเมื่อ search ไม่เจอ
- [ ] ไม่มี audit queue/drain
- [ ] Tool calling ไม่มี allowlist/timeout/max iteration
- [ ] เปิด web search โดยไม่มี cost control/cache
- [ ] Prompt ไม่ version-controlled
- [ ] ไม่มี eval suite ทำให้ตอบดีเฉพาะ demo
- [ ] ไม่แยก `/ready` กับ `/health`
- [ ] ผูก code กับ Ollama แล้ว migrate vLLM ยาก

---

## 21. Recommended Next Docs ถ้าจะ rebuild จริง

- [ ] `12-PYTHON-REBUILD-SPEC.md` — spec สำหรับ FastAPI + LangGraph feature parity
- [ ] `13-API-CONTRACT.md` — OpenAI-compatible API + admin/RAG endpoints
- [ ] `14-DATA-MODEL.md` — schema/session/audit/document/tool run
- [ ] `15-EVAL-PLAN.md` — eval dataset สำหรับ RAG/agent/web search

