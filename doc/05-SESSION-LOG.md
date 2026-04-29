# 05 — Session Logs
> **Last Updated:** 2026-04-29

---

## 2026-04-28 (Day 1)

### สิ่งที่ทำ
- SSH MacBook → iMac setup
- Ollama + qwen2.5:7b + nomic-embed-text ติดตั้ง
- Ollama Optimization: FLASH_ATTENTION + KV_CACHE q8_0
- Docker Desktop + docker-compose (postgres + webui)
- Go 1.26 + VS Code Remote SSH
- Git + GitHub (branch: main/dev)
- Go Orchestrator Phase 0: Fiber v3, /health, /v1/chat/completions
- RAG Phase 1: pgvector, EmbedderPort, NomicAdapter, chunker, parser
- MCP Phase 2: SQLServerAdapter (structure only)
- Intent Router Phase 3: Rule-based classifier

---

## 2026-04-29 (Day 2 — Morning)

### สิ่งที่ทำ
- ย้าย docker-compose.yml → root
- สร้าง Dockerfile (multi-stage)
- สร้าง .env.example, .gitignore, README.md
- git merge dev → main (แก้ conflict)
- รัน Go ใน Docker (`docker compose up --build`)
- pgvector/schema init ปัจจุบันใช้ embedded goose migrations
- ทดสอบ MacBook → iMac Server ผ่าน Postman
- PDF Upload + Ollama AI OCR, with optional local Tesseract fallback
- Standard Response Format: `{"data":{}}` / `{"error":{}}`
- แยก handlers: health.go, chat.go, rag.go, response.go
- Upload PDF จริง "บค.002-2568 OT Policy" → 5 chunks → ตอบจากเอกสาร ✅

---

## 2026-04-29 (Day 2 — Afternoon)

### สิ่งที่ทำ
- Health Check ครบ: db + llm + embedder
- Upgrade Embedding: nomic-embed-text → nomic-embed-text-v2-moe
- EmbedModel อ่านจาก .env (configurable)
- ลด ChunkSize: 500 → 200 (v2-moe context limit 512)
- Clean whitespace ก่อน embed
- Graceful Shutdown (SIGTERM/SIGINT)
- Structured Logging: slog JSON format
- Streaming Support: SSE (Server-Sent Events)
- WebUI Integration: /v1/models endpoint
- Open WebUI → Go Orchestrator → RAG → Stream ✅
- SSH key (ed25519) บน iMac → GitHub

---

## 2026-04-29 (Day 2 — Evening)

### สิ่งที่ทำ

**AD/LDAP Authentication + JWT:**
- LDAPAdapter: StartTLS, bind username@nutrition.com
- JWTManager: HS256, 8h expiry
- POST /auth/login → ได้ JWT + user info จาก AD
- GET /auth/me → ดูข้อมูล user
- JWT Middleware: protect /v1/* routes
- ค้นพบ: Domain จริงคือ `nutrition.com` (ไม่ใช่ nutritionprofess.com)

**Session Management:**
- DB Schema: sessions + messages + indexes
- SessionPort interface
- PostgresAdapter: CreateSession, GetHistory, SaveMessage, SaveMessages
- Session Middleware: auto-create/restore session
- Sliding Window: 10 messages ล่าสุด เรียงด้วย `sequence_number`
- metadata JSONB: เก็บ intent
- Save user/assistant messages ใน transaction เดียว
- X-Session-ID ใน response header
- ทดสอบ: AI จำ context ข้าม request ได้ ✅

---

## Commits Summary

| Commit | Feature |
|---|---|
| `85243db` | phase 0 - go orchestrator |
| `5767615` | phase 1 - RAG engine |
| `1d378f6` | phase 2 - MCP bridge |
| `507ba05` | phase 3 - intent router |
| `9ffc70c` | streaming + WebUI + /v1/models |
| `feat` | health check all deps |
| `feat` | nomic-embed-text-v2-moe |
| `feat` | graceful shutdown |
| `feat` | slog structured logging |
| `feat` | AD/LDAP auth + JWT |
| `a2551a3` | session management |

---

## 2026-04-29 (Day 2 — Night)

### Audit Log ✅

**Files สร้างใหม่:**
| ไฟล์ | หน้าที่ |
|---|---|
| `internal/infrastructure/migrations/sql/00001_baseline.sql` | baseline schema รวม audit_logs + indexes |
| `internal/domain/models/audit.go` | AuditLog, AuditLogFilter |
| `internal/domain/ports/audit.go` | AuditPort interface |
| `internal/infrastructure/audit/postgres.go` | PostgreSQL adapter |
| `internal/api/middleware/audit.go` | Auto-record ทุก request |
| `internal/api/handlers/audit.go` | GET /v1/admin/logs |

**บันทึกข้อมูล:**
- user_id, username ✅
- session_id ✅
- method + path ✅
- query (คำถาม) ✅
- response_preview (200 chars) ✅
- latency_ms ✅
- status_code ✅
- ip_address ✅

**Endpoint:**
```
GET /v1/admin/logs?user_id=nipon.k&intent=rag
Authorization: Bearer TOKEN
```

**Git commit:** `aabcf34`

### TODO ที่ยังค้างอยู่
- [ ] Role-based Access Control
- [ ] intent field ใน audit_logs (ปัจจุบันอยู่ใน messages.metadata)
- [ ] Rate Limiting
- [ ] WebUI Login
- [ ] Document Management
