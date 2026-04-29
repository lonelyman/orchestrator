# Production-Grade Roadmap

> **Created:** 2026-04-29
> **Source:** `07-CODE-REVIEW.md`
> **Target:** ยกระดับจาก PoC → production-grade enterprise (Docker deployment)
> **Phase 1 scope:** 10-15 users, 10 RAG docs, Ollama backend
> **Phase 2 scope:** vLLM + NVIDIA Blackwell

---

## 📐 Principle

แต่ละ sprint:
- **Independent** — merge ได้แยก, ไม่ block sprint อื่น
- **Reversible** — feature flag หรือ env toggle, rollback ง่าย
- **Tested** — มี unit + integration test ก่อน merge
- **Documented** — update doc ใน sprint เดียวกัน

**Sprint length:** ~1 สัปดาห์ (ปรับตามเวลาจริง)

---

## 🗺️ Roadmap Overview

```
Sprint 1 — Security Foundation        🔴 Blocker for prod
Sprint 2 — Reliability Hardening      🟠 Outage prevention
Sprint 3 — Observability Stack        🟠 Visibility before prod
Sprint 4 — Resilience Patterns        🟡 Tail latency / cascade
Sprint 5 — CI/CD + Quality Gates      🟡 Sustainability
Sprint 6 — RAG Quality (Phase 1.5)    🟢 Product value
Sprint 7 — Scale Prep (Phase 2)       🟢 vLLM migration ready
```

---

## Sprint 1 — Security Foundation 🔴

**Why first:** P0 items คือ **blocker จริง** สำหรับ enterprise — แก้ก่อนจึงจะเปิด prod ได้

### Tasks

#### 1.1 LDAP TLS hardening
- เพิ่ม env: `LDAP_TLS_CA_FILE`, `LDAP_TLS_SKIP_VERIFY` (default `false`)
- รองรับ `ldaps://` (port 636) ผ่าน config
- StartTLS error → fail hard เมื่อ `DEV_MODE=false`
- **Test:** unit test กับ ldap mock + manual test กับ AD จริง

#### 1.2 JWT hardening
- เพิ่ม `Audience: "orchestrator"`, `Subject: user.ID`, `JTI` (JWT ID)
- Verify `iss`, `aud` ตอน parse
- ตาราง `revoked_tokens(jti, user_id, revoked_at, expires_at)`
- Endpoint `POST /auth/logout` → revoke jti
- (Optional) Refresh token rotation flow

#### 1.3 Security headers + CORS
- เพิ่ม `fiber/middleware/helmet` (HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy)
- CORS policy: env `ALLOWED_ORIGINS` (default deny)
- CSP header สำหรับ WebUI

#### 1.4 Login brute-force protection
- Rate limit `/auth/login` แยก: 5 ครั้ง / 15 นาที per (IP + username)
- Track failed attempts ใน `auth_attempts` table
- Account lockout 30 นาทีถ้า fail ≥ 10 ครั้งใน 1 ชม.

#### 1.5 SSRF protection
- Validate `LLM_HOST`, `EMBED_HOST`, `OCR_HOST` ตอน config load
- Reject link-local (169.254.x.x), metadata services (169.254.169.254)
- Allow loopback เฉพาะ `DEV_MODE=true`

#### 1.6 Secrets out of compose
- ใช้ Docker secrets หรือ `.env` (ไม่ commit)
- Document key rotation procedure

### Deliverables
- [ ] PR #1: LDAP TLS
- [ ] PR #2: JWT v2 (audience, jti, revocation)
- [ ] PR #3: Security headers + CORS
- [ ] PR #4: Login rate limit + lockout
- [ ] PR #5: SSRF guard
- [ ] PR #6: Secrets management doc

### Decision needed (จากนโยบาย)
- ❓ มี enterprise PKI / CA bundle ไหม? (ต้องรอ infra team)
- ❓ JWT expiry policy: 8 ชม. + refresh 7 วัน OK ไหม?
- ❓ Allowed CORS origins มีกี่ domain?

---

## Sprint 2 — Reliability Hardening 🟠

**Why:** ปิดช่อง goroutine leak, race condition, graceful shutdown ที่ไม่สมบูรณ์

### Tasks

#### 2.1 Audit worker pool
- แทน `go func()` ด้วย channel-based worker (4 workers, queue 1000)
- Drop policy: log warning + counter เมื่อ queue เต็ม
- Drain ตอน shutdown (timeout 10s)

#### 2.2 SaveMessage atomicity
- รวม user + assistant message ใน transaction เดียว
- หรือใส่ `sequence_number` คอลัมน์ + index `(session_id, sequence_number)`

#### 2.3 Graceful shutdown ที่จริง
- `cmd/server/main.go` refactor:
  1. Receive signal
  2. Stop accept new connections
  3. Drain in-flight requests (timeout)
  4. Flush audit queue
  5. Cancel all goroutines via root context
  6. Close DB pool
- `Listen` error → trigger shutdown channel

#### 2.4 Streaming context propagation
- `chat.go` ใช้ `c.Context()` แทน `context.Background()`
- `parser.go` `ocrImageWithOllama` รับ `context.Context` parameter

#### 2.5 DB pool tuning
- Override ใน `connectPostgres`:
  ```go
  poolConfig.MaxConns = 20
  poolConfig.MinConns = 2
  poolConfig.MaxConnLifetime = 30 * time.Minute
  poolConfig.MaxConnIdleTime = 5 * time.Minute
  poolConfig.HealthCheckPeriod = 1 * time.Minute
  ```
- ทำให้ override ผ่าน env ได้

#### 2.6 Health probe split
- `/live` — process alive only (ปัจจุบันถูก)
- `/ready` — DB connectable เท่านั้น
- `/health` — full dependency status (สำหรับ monitor, ไม่ใช่ probe)

### Deliverables
- [ ] PR #7: Audit worker pool
- [ ] PR #8: Message atomicity
- [ ] PR #9: Graceful shutdown v2
- [ ] PR #10: Streaming context
- [ ] PR #11: DB pool tuning
- [ ] PR #12: Health probe semantics

### Decision needed
- ❓ Audit log loss tolerance: drop หรือ block request เมื่อ queue เต็ม? (compliance impact)

---

## Sprint 3 — Observability Stack 🟠

**Why:** จะ debug prod ไม่ได้ถ้าไม่มี metrics/log/trace

### Tasks

#### 3.1 Prometheus metrics
- เพิ่ม `/metrics` endpoint (Prometheus format)
- HTTP RED metrics per endpoint (rate, errors, duration histogram)
- LLM metrics (latency, token count, model name label)
- DB pool metrics (in-use, idle, wait)
- Custom: intent classification distribution

#### 3.2 Structured logging completion
- slog handler middleware ที่ inject `request_id`, `user_id`, `session_id` ใน every log
- `LOG_LEVEL` env (debug/info/warn/error)
- Redaction: scrub `password`, `Authorization` header, JWT body จาก log

#### 3.3 Loki + Grafana stack
- Uncomment ใน `docker-compose.yml`
- Loki driver plugin install doc
- Pre-built dashboards:
  - Request rate / latency / error rate
  - LLM performance
  - DB pool
  - Audit log volume

#### 3.4 (Optional) Tempo + OpenTelemetry
- เพิ่ม OTel SDK + tempo exporter
- Auto-instrument fiber + pgx + http.Client
- Trace context propagation `traceparent` header

### Deliverables
- [ ] PR #13: Prometheus metrics
- [ ] PR #14: Structured logging middleware
- [ ] PR #15: Loki + Grafana compose
- [ ] PR #16: Pre-built Grafana dashboards (JSON)
- [ ] (Optional) PR #17: OTel + Tempo

### Decision needed
- ❓ Tempo จำเป็นใน Phase 1 ไหม? (ผมแนะนำข้ามไปก่อน)
- ❓ Monitoring server แยกจาก AI server? (compose file โน้ตไว้แล้ว)

---

## Sprint 4 — Resilience Patterns 🟡

**Why:** ป้องกัน cascade failure ตอน LLM/embedder/OCR ล่ม

### Tasks

#### 4.1 Circuit breaker
- ใส่ `sony/gobreaker` รอบ HTTP call ทุก external service
- Settings: 5 failures / 60s window → open 30s
- Metrics: breaker state changes

#### 4.2 Retry policy
- Exponential backoff สำหรับ idempotent calls (embed, health check)
- ไม่ retry สำหรับ chat completion (user คอย)
- Max 3 retries, jitter 100ms

#### 4.3 Distributed rate limiter
- ย้ายจาก in-memory → **Redis-backed** (sliding window log algorithm)
- หรือ Postgres-based (sufficient for Phase 1)
- Key: `user_id` แทน IP

#### 4.4 Request timeout cascade
- ตั้ง deadline บน root context ทุก request
- Inner call ใช้ deadline ที่เหลือ ไม่ใช่ตั้ง timeout ใหม่ที่อาจยาวกว่า

### Deliverables
- [ ] PR #18: Circuit breaker
- [ ] PR #19: Retry middleware
- [ ] PR #20: Distributed rate limiter
- [ ] PR #21: Context deadline propagation

### Decision needed
- ❓ Redis เพิ่มเข้า stack ไหม? (ต้องการสำหรับ rate limit + cache phase 2)
- ❓ Circuit breaker open → fallback to error message หรือ cached response?

---

## Sprint 5 — CI/CD + Quality Gates 🟡

**Why:** ป้องกัน regression, รัน security scan อัตโนมัติ

### Tasks

#### 5.1 GitHub Actions workflows
- `lint.yml`: golangci-lint
- `test.yml`: unit + integration test (testcontainers Postgres)
- `security.yml`: gosec + govulncheck + trivy (image scan)
- `build.yml`: multi-arch Docker build + push

#### 5.2 Pre-commit hooks
- `gofmt`, `go vet`, `golangci-lint --fast`
- License header check
- Secret scan (gitleaks)

#### 5.3 Integration test framework
- testcontainers สำหรับ Postgres + pgvector
- Mock LLM/embedder server (httptest)
- Test fixtures + factory

#### 5.4 Load test baseline
- k6 script: chat completion @ 10 RPS
- baseline metrics: p50/p95/p99 latency, error rate
- Fail CI ถ้า regression > 20%

#### 5.5 Dockerfile production-ready
- Multi-stage + non-root user
- HEALTHCHECK directive
- Pin base image SHA256
- `.dockerignore` ครบ

### Deliverables
- [ ] PR #22: GitHub Actions
- [ ] PR #23: Pre-commit
- [ ] PR #24: Integration test setup
- [ ] PR #25: k6 load test
- [ ] PR #26: Dockerfile hardening

### Decision needed
- ❓ Container registry: GHCR / Docker Hub / Internal Harbor?
- ❓ Image signing (cosign) จำเป็นไหม?

---

## Sprint 6 — RAG Quality 🟢

**Why:** Phase 1.5 — เพิ่มคุณภาพ retrieval ก่อนเปิดให้ user

### Tasks

#### 6.1 HNSW index ✅
- Baseline migration มี `CREATE INDEX ... USING hnsw (embedding vector_cosine_ops)` แล้ว
- Tune `m`, `ef_construction` ภายหลังเมื่อ doc count โตขึ้น

#### 6.2 Hybrid search (BM25 + vector)
- เพิ่ม `tsvector` คอลัมน์ใน `documents` (auto generated)
- Query: rrf (Reciprocal Rank Fusion) ของ vector + BM25 result
- Thai tokenizer (postgres `pg_jieba` หรือ custom)

#### 6.3 Ingest batch embedding
- Ollama embed รองรับ batch — เปลี่ยน loop เป็น 1 request
- Bulk insert ใช้ `COPY`

#### 6.4 Remove fallback DESC
- `pgvector.go` ลบ fallback ที่คืน irrelevant docs
- Return empty + ปรับ prompt ให้ LLM ตอบ "ไม่พบข้อมูลในเอกสาร"

#### 6.5 Embedding cache
- Hash content → cache ใน Redis/Postgres
- Hit rate metric

#### 6.6 Intent classifier v2
- ใช้ embedding similarity เทียบ intent prototype (5-10 ตัวอย่างต่อ intent)
- Threshold-based + fallback to LLM classifier

### Deliverables
- [x] PR #27: HNSW migration
- [ ] PR #28: Hybrid search
- [ ] PR #29: Batch embedding
- [ ] PR #30: Fallback removal
- [ ] PR #31: Embedding cache
- [ ] PR #32: Intent v2

### Decision needed
- ❓ Thai tokenizer choice: pg_jieba / pythainlp / สั่ง LLM ทำ?
- ❓ Embedding cache TTL: 30 วัน?

---

## Sprint 7 — Scale Prep (Phase 2) 🟢

**Why:** เตรียมตัวก่อน migrate ไป vLLM + NVIDIA Blackwell

### Tasks

#### 7.1 Migration as separate service
- `cmd/migrate/main.go` แยก
- compose: service `migrator` run before `orchestrator`
- Document rollback procedure

#### 7.2 Async RAG ingest
- Job queue (River — Go-native, Postgres-backed)
- Worker process แยกจาก API
- Status tracking: `documents.status` (pending/processing/done/failed)

#### 7.3 Multi-tenant foundation (ตามนโยบาย)
- เพิ่ม `org_id` / `department_id` ใน `documents`, `audit_logs`, `sessions`
- RAG search filter ตาม user department
- Admin scope (cross-department) แยก role

#### 7.4 LLM router
- Cheap model (intent classify) vs big model (generate)
- Config-driven routing rules

#### 7.5 vLLM adapter validation
- มี `openai_compatible.go` แล้ว — verify ทำงานกับ vLLM จริง
- Test: streaming, function calling, system prompt
- Performance test vs Ollama baseline

### Deliverables
- [ ] PR #33: Migration service
- [ ] PR #34: River job queue
- [ ] PR #35: Multi-tenant schema
- [ ] PR #36: LLM router
- [ ] PR #37: vLLM validation suite

### Decision needed (รอนโยบาย)
- ❓ Multi-tenant model: pool/bridge/silo?
- ❓ Department mapping: AD `department` field พอไหม?
- ❓ vLLM hardware ETA?

---

## 📊 Effort Estimate (rough)

| Sprint | Complexity | Items | Est. effort | Order |
|---|---|---|---|---|
| 1 — Security | M-L | 6 | 1 sprint | **First** |
| 2 — Reliability | M | 6 | 1 sprint | Second |
| 3 — Observability | M | 4-5 | 1 sprint | Third |
| 4 — Resilience | M | 4 | 0.5-1 sprint | Fourth |
| 5 — CI/CD | M | 5 | 1 sprint | Anytime parallel |
| 6 — RAG quality | L | 6 | 1-2 sprint | Phase 1.5 |
| 7 — Scale prep | L | 5 | 2 sprint | Before Phase 2 |

**Total to "production-ready Phase 1":** ~5 sprints (Sprint 1-5)
**Total to "vLLM ready":** +2 sprints (6-7)

---

## 🎯 Recommendation

### ทำตามลำดับนี้ถ้าเริ่มได้เลย:

1. **Sprint 1 (Security)** — required สำหรับเปิด prod
2. **Sprint 5 (CI/CD)** บางส่วน parallel — ตั้ง lint/test ก่อน sprint อื่นเพื่อรับ PR
3. **Sprint 2 (Reliability)** — กัน outage
4. **Sprint 3 (Observability)** — ดู prod ได้
5. **Sprint 4 (Resilience)** — ทน failure
6. **Sprint 6 (RAG)** — ก่อนเปิด user phase 1
7. **Sprint 7 (Scale)** — เมื่อจะขึ้น vLLM

### ทำคู่ขนานได้:
- Sprint 5 (CI/CD) ทำพร้อม sprint อื่นได้
- Doc updates ทุก sprint

### ไม่แนะนำ skip:
- Sprint 1 — ปิด security gap คือ priority สูงสุด
- Sprint 3 — ไม่มี observability = แก้ prod ไม่ได้

---

## 📝 ข้อมูลรอตัดสินใจ (รวม)

จาก review ทั้งหมด เรื่องที่ต้องรอนโยบาย:

| ลำดับ | คำถาม | กระทบ sprint |
|---|---|---|
| 1 | Enterprise PKI / CA bundle มีไหม? | 1 |
| 2 | JWT expiry policy + refresh token strategy | 1 |
| 3 | Allowed CORS origins | 1 |
| 4 | Audit log loss tolerance (drop vs block) | 2 |
| 5 | Tracing จำเป็น phase 1 ไหม? | 3 |
| 6 | Monitoring server แยกเครื่องไหม? | 3 |
| 7 | Redis เข้า stack? | 4, 6 |
| 8 | Container registry choice | 5 |
| 9 | Image signing required? | 5 |
| 10 | Multi-tenant model + boundary | 7 |
| 11 | Department mapping source | 7 |
| 12 | vLLM hardware ETA | 7 |

---

## 🔗 Related docs

- `07-CODE-REVIEW.md` — Detailed gap analysis
- `06-ENTERPRISE.md` — Enterprise requirements (existing)
- `03-ROADMAP.md` — Original roadmap (สามารถ merge เข้ากับ doc นี้ภายหลัง)
