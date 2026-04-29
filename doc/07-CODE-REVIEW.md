# Code Review — Production Readiness Assessment

> **Reviewed:** 2026-04-29
> **Reviewer:** Claude (Opus 4.7)
> **Scope:** ทั้ง codebase ตามสถานะ branch `dev`
> **Goal:** ประเมินว่าห่างจาก "production-grade BigTech standard" แค่ไหน + วางแผนปิด gap

---

## 📊 สรุปภาพรวม

**จุดแข็ง (เก็บไว้):**
- Hexagonal architecture (ports/adapters) แยกชัด → swap adapter ได้ (Ollama ↔ vLLM)
- Goose migrations แบบ embed → deploy ง่าย
- slog JSON logging ตั้งแต่แรก
- Config validation + DEV_MODE guard ที่ block ค่าอ่อนตอนขึ้น prod
- Test coverage บน middleware/handler หลายตัว
- Streaming SSE ครบ + Sliding window history
- pgvector + Goose + Fiber v3 — stack สมัยใหม่

**จุดที่ห่างจาก production-grade:**
- Security gaps ระดับ blocker (LDAP MITM, ไม่มี security headers, ไม่มี brute-force protection)
- Reliability — fire-and-forget goroutines, ไม่มี circuit breaker, graceful shutdown ไม่สมบูรณ์
- Observability — ไม่มี metrics/tracing, log ไม่มี request_id ทุกที่
- Multi-replica concerns — rate limiter in-memory, migration inline, no leader election

---

## 🔴 P0 — ต้องแก้ก่อนเปิด prod (Security / Correctness)

### 1. LDAP `InsecureSkipVerify: true`
**ที่:** `internal/infrastructure/auth/ldap.go:59`
**ปัญหา:** StartTLS ที่ skip cert verification เปิดช่อง MITM ขโมย AD credentials

**แก้:**
- โหลด CA bundle ของ enterprise PKI
- รองรับ `LDAPS://` (port 636)
- เพิ่ม `LDAP_TLS_CA_FILE`, `LDAP_TLS_SKIP_VERIFY` (default `false`)
- StartTLS error → fail hard เมื่อ `DEV_MODE=false`

### 2. JWT — ไม่มี revocation, refresh token, audience claim
**ที่:** `internal/infrastructure/auth/jwt.go`
**ปัญหา:**
- ไม่มี `Audience`, `Subject` → token reuse ข้าม service ได้
- ไม่มี blocklist → logout ไม่ได้จริง, token ที่หลุดใช้ได้จนหมดอายุ (8 ชม.)
- ไม่มี refresh token → re-login ทุก 8 ชม. UX แย่
- ไม่ verify `iss` ตอน parse

**แก้:**
- เพิ่ม `Audience: "orchestrator"`, `Subject: user.ID`
- ตาราง `revoked_tokens(jti, expires_at)` + คอลัมน์ `jti` ใน claim
- Refresh token แยก (long-lived, stored hashed in DB, rotation on use)

### 3. Rate limiter เป็น in-memory
**ที่:** `internal/api/router.go:64`
**ปัญหา:** Fiber default limiter เก็บใน memory — restart หาย, ไม่ทำงานข้าม instance ตอน scale

**แก้:**
- ใช้ Redis-backed limiter
- หรือ Postgres-based token bucket (เหมาะกับ scale Phase 1 ที่ user 10-15)

### 4. Audit fire-and-forget goroutine
**ที่:** `internal/api/middleware/audit.go:67`
**ปัญหา:**
- ไม่มี bound → traffic spike + DB ช้า → goroutine leak
- `context.Background()` → SIGTERM flush ไม่ทัน → log หาย (compliance issue)

**แก้:** buffered channel + worker pool (4 workers, queue 1000) + drain ตอน shutdown

### 5. SaveMessage fire-and-forget + race condition
**ที่:** `internal/core/orchestrator/orchestrator.go:107-108`
**สถานะ:** ✅ แก้แล้ว — save user/assistant ใน transaction เดียว และใช้ `sequence_number` สำหรับ history ordering
**ปัญหา:** user/assistant message save async parallel → ลำดับสลับได้ → history request ถัดไปอ่านไม่ครบ

**แก้:** save ใน transaction เดียว หรือใส่ `sequence_number` คอลัมน์

### 6. ไม่มี Security headers / CORS
**ที่:** `internal/api/router.go`
**ปัญหา:** ไม่มี HSTS, X-Frame-Options, X-Content-Type-Options, CSP, CORS policy

**แก้:** ใช้ `fiber/middleware/helmet` + `fiber/middleware/cors`

### 7. ไม่มี brute-force protection ที่ login
**ที่:** `internal/api/handlers/auth.go`
**ปัญหา:** เปิดทาง credential stuffing เข้า AD → AD account ถูก lock ได้ (DoS)

**แก้:** rate limit `/auth/login` แยกต่างหาก (เช่น 5 ครั้ง/15 นาที per IP+username) + delay backoff

### 8. SSRF risk จาก env
**ที่:** `LLM_HOST`, `EMBED_HOST`, `OCR_HOST` ไม่ validate
**ปัญหา:** config ผิดพลาดชี้ internal metadata service → leak

**แก้:** validate ว่าไม่ใช่ link-local (169.254.x.x), loopback (ยกเว้น dev mode), หรือใช้ allowlist hostname

---

## 🟠 P1 — Reliability / Observability

### 9. Graceful shutdown ไม่สมบูรณ์
**ที่:** `cmd/server/main.go:35-50`
**ปัญหา:**
- `Listen` error log แล้วไม่ trigger shutdown
- ไม่ drain in-flight requests
- ไม่ wait audit/session goroutines
- ไม่ปิด LLM streaming connections

**แก้:** signal handler trigger ปิดทุก subsystem ตามลำดับ (stop accept → drain → flush goroutines → close DB)

### 10. ไม่มี Prometheus metrics / OpenTelemetry tracing
**ที่:** ทั้งระบบ
**ที่ควรมี:**
- HTTP RED metrics (rate/errors/duration) per endpoint
- LLM latency histogram + token usage counter
- DB pool stats (acquire wait, in-use, idle)
- OTel trace context (`traceparent`) propagate ไป LLM/embedder/DB

### 11. Logging ขาด structured fields
**ปัญหา:**
- หลาย log ไม่มี `request_id` (เช่น `ldap.go:81`)
- ไม่มี `LOG_LEVEL` env (default Info)
- ไม่มี redaction → password/token หลุดเข้า log ได้

**แก้:** middleware ใส่ slog handler ที่ inject request_id + user_id ทุก log ภายใน request

### 12. ไม่มี circuit breaker / retry
**ที่:** `internal/infrastructure/llm/`, `internal/infrastructure/embedder/`
**ปัญหา:** LLM ค้าง 30 วิ → request ทั้งหมดค้าง → exhaust pool → cascade failure

**แก้:** `sony/gobreaker` รอบ HTTP client + exponential backoff retry สำหรับ idempotent calls

### 13. DB pool ไม่ tune
**ที่:** `internal/app/server.go:125`
**ปัญหา:** default `max(4, NumCPU)` — น้อยเกินไปสำหรับ prod

**แก้:** override `MaxConns=20`, `MinConns=2`, `MaxConnLifetime=30m`, `MaxConnIdleTime=5m` (ปรับตามโหลด)

### 14. Readiness probe รวม external dependency
**ที่:** `internal/api/handlers/health.go:51`
**สถานะ:** ✅ แก้แล้ว — `/ready` เช็ก DB เท่านั้น, `/health` เช็ก full dependencies
**ปัญหา:** LLM down → readiness fail → K8s remove pod → recovery ยาก

**แก้:**
- Liveness = process alive
- Readiness = DB connectable เท่านั้น (สิ่งที่ block การรับ request)
- Health (สถานะแยก) = ทุก dependency, ใช้สำหรับ monitor ไม่ใช่ probe

### 15. Streaming ใช้ `context.Background()`
**ที่:** `internal/api/handlers/chat.go:76`, `parser.go:351`
**สถานะ:** 🟡 แก้แล้วสำหรับ chat request context; OCR parser context ยังเหลือไว้ใน backlog
**ปัญหา:** client cancel ไม่ propagate → LLM request ทำต่อ → เปลือง GPU

**แก้:** ใช้ `c.UserContext()` หรือ derive จาก request context

---

## 🟡 P2 — Code Quality

### 16. Constructor signature ขยายไม่ดี
**ที่:** `NewChatHandlerWithTimeout(orch, timeout, modelAndOwner ...string)`
**ปัญหา:** `...string` แทน struct → fragile, ลำดับสลับไม่รู้

**แก้:** functional options pattern หรือ config struct

### 17. Intent classifier เป็น hardcoded keyword
**ที่:** `internal/core/intent/classifier.go`
**ปัญหา:** confidence 0.8/0.6 ไม่มีความหมาย, คำใหม่ต้อง redeploy

**แก้ (Phase 1):** ใช้ embedding similarity เทียบ intent prototype
**แก้ (Phase 2):** small LLM classifier + cache

### 18. RAG ยังไม่มี hybrid search / reranking
**ที่:** `internal/infrastructure/vector/pgvector.go`
**ปัญหา:**
- vector search อย่างเดียว → recall ดีแต่ precision ต่ำ
- HNSW index แก้แล้วใน baseline migration

**แก้:**
- เพิ่ม BM25 (`tsvector`) → hybrid scoring
- (Phase 2) Cross-encoder reranker

### 19. RAG fallback คืน irrelevant docs
**ที่:** `pgvector.go:86-91`
**สถานะ:** ✅ แก้แล้ว — vector search คืน empty เมื่อไม่พบผลลัพธ์ และ prompt ห้ามเดาจาก general knowledge
**ปัญหา:** เมื่อ vector search ไม่พบ → คืน docs ตาม `created_at DESC` → LLM ใช้ context ที่ไม่เกี่ยว → hallucinate

**แก้:** คืน empty + ให้ LLM ตอบ "ไม่พบข้อมูลในเอกสาร"

### 20. RAG ingest ไม่ batch embedding
**ที่:** `internal/core/rag/rag.go:37`
**ปัญหา:** 100 chunks = 100 HTTP calls → ช้า

**แก้:** batch embed (Ollama รองรับ) + bulk insert

### 21. Migration inline ใน app startup
**ที่:** `internal/app/server.go:46`
**ปัญหา:** multi-replica → goose advisory lock ทำให้ pod อื่นต้องรอ → SLO bust

**แก้:** แยก migration เป็น `cmd/migrate/main.go` → run เป็น init container / Job ใน K8s
**สำหรับ Docker (target ปัจจุบัน):** ทำเป็น service แยก `migrator` ที่ run ก่อน orchestrator start

### 22. Magic numbers / hardcoded fallbacks
- `historyLimit = 10` ใน orchestrator
- `"qwen2.5:7b"` ใน chat handler default
- `confidence: 0.8` ใน classifier

**แก้:** ย้ายเข้า config + ตั้งชื่อ constant ให้ชัด

### 23. ไม่มี request body validation
**ที่:** `chat.go`, `rag.go`
**ปัญหา:** validate แค่ "messages required" — ไม่ check role, length, content size

**แก้:** ใช้ `go-playground/validator` หรือ explicit validation function

### 24. Type-unsafe `c.Locals("session_id")`
**ปัญหา:** string key กระจาย → typo ไม่ compile error

**แก้:** typed context helpers (`SessionIDFromContext(c) (string, bool)`)

### 25. Dockerfile / compose ปรับปรุงได้
- ไม่มี non-root user
- ไม่มี HEALTHCHECK
- compose hardcode `POSTGRES_PASSWORD: changeme`
- image base `alpine:latest` → ควร pin SHA256

---

## 🟢 P3 — Architecture for scale

### 26. Single-service mono — ไม่แยก async work
RAG ingest, OCR เป็น CPU/IO heavy → ทำใน request lifecycle = tail latency แย่
**แนวคิด:** worker process แยก + job queue (Postgres `pg_notify` หรือ Redis Streams)

### 27. ไม่มี caching layer
- Embedding cache (เนื้อหาเดิม → embedding เดิม) — ประหยัด GPU
- Session history cache
- LLM response cache สำหรับ FAQ

### 28. Multi-tenant isolation
ดู section "Multi-tenant — ตอบคำถามจากที่คุณถามมา" ด้านล่าง

---

## ❓ Multi-tenant — คำอธิบายและคำถามเจาะ

**Multi-tenant** = ระบบเดียวให้บริการหลาย "ลูกค้า/องค์กร/แผนก" ที่ต้องแยกข้อมูลกัน

### Pattern ที่นิยม (เรียงตาม isolation strength):

| Pattern | คำอธิบาย | เหมาะกับ |
|---|---|---|
| **Pool model** | Schema เดียว, ใส่ `tenant_id` ทุก table, filter ที่ query | SaaS หลายลูกค้า, scale ดี, isolation อ่อน |
| **Bridge model** | Schema เดียว, แต่ละ tenant มี schema/database แยก | enterprise ที่ต้องการ data residency |
| **Silo model** | แต่ละ tenant deploy แยก (instance/cluster) | regulated industry (banking, health) |

### คำถามเจาะที่ผมต้องรู้เพื่อแนะนำ:

1. **ระบบนี้จะให้ใครใช้?**
   - แผนกเดียว (HR เท่านั้น)?
   - ทั้งบริษัท แต่หลายแผนก (HR, Finance, IT, Sales)?
   - หลายบริษัท / หลายสาขา?

2. **เอกสาร RAG ต้องแยกการเข้าถึงไหม?**
   - HR upload เอกสารเงินเดือน → Sales ห้ามเห็น?
   - หรือทุกคนเห็นทุกเอกสาร?
   - มี classification level (Public / Internal / Confidential) ไหม?

3. **Audit log ต้องแยกการดูไหม?**
   - HR Manager ดูได้แค่ log ของแผนก HR?
   - หรือมี admin คนเดียวดูทั้งหมด?

4. **Chat history เป็นของใคร?**
   - User เห็นแค่ของตัวเอง (ปัจจุบันเป็นแบบนี้แล้ว)?
   - Manager ดูของลูกน้องได้?

**Recommendation เบื้องต้น (Phase 1):**
- User 10-15 คน, docs 10 ไฟล์ → **ยังไม่ต้องทำ multi-tenant** ก็ได้ ใช้ pool model เผื่อไว้: เพิ่มคอลัมน์ `org_id` / `department` ใน `documents`, `audit_logs`, `sessions` ตอนนี้ → ขยายทีหลังโดยไม่ต้อง migrate ใหญ่
- ใช้ AD `department` field ที่ดึงมาอยู่แล้ว เป็น tenant boundary ตามธรรมชาติ
- RAG search ใส่ filter `WHERE department = ANY($user_departments)` — ทำได้ถูกหลัง phase 1

---

## 🔧 Stack ที่เสนอ (Redis / Kafka / Observability)

### สถานะปัจจุบัน:
- ❌ ไม่มี Redis
- ❌ ไม่มี Message Queue
- ❌ ไม่มี metrics/tracing
- ✅ มี slog JSON + comment ใน compose ว่าจะใช้ Loki + Grafana (ยัง commented)

### Recommendation ตาม scale Phase 1 (10-15 user, 10 docs):

#### **Tier 1 — ต้องมีก่อน prod (low effort, high value)**

| Component | เลือก | เหตุผล |
|---|---|---|
| **Log aggregation** | **Loki + Grafana** (มีใน compose แล้ว) | Free, lightweight, query log ผ่าน Grafana, เหมาะกับ slog JSON |
| **Metrics** | **Prometheus + Grafana** | Standard, scrape `/metrics` endpoint, dashboard สำเร็จเยอะ |
| **Tracing** | **ข้ามไปก่อน** หรือ **Grafana Tempo** | Phase 1 user น้อย, log ดีพอ debug |

→ Stack เดียว `Grafana LGTM` (Loki + Grafana + Tempo + Mimir) ติดตั้งทีเดียวได้ทั้งหมด

#### **Tier 2 — เพิ่มเมื่อโตขึ้น (Phase 2)**

| Component | เลือก | เหตุผล |
|---|---|---|
| **Cache** | **Redis (single instance)** | Rate limit + session cache + embedding cache |
| **Job queue** | **Postgres `pg_notify` + table-based queue** หรือ **River** (Go-native) | RAG ingest async โดยไม่ต้องเพิ่ม Kafka |

#### **Tier 3 — ไม่จำเป็นจนกว่า scale ใหญ่**

- **Kafka / NATS** — overkill สำหรับ user 10-15 คน
- **Service mesh (Istio)** — ไม่จำเป็นเลยใน Docker single-host

### Recommendation Summary:

```
Phase 1 (POC, Ollama, 10-15 users):
  ✅ Loki + Grafana + Prometheus  (uncomment + setup)
  ✅ Redis single instance         (rate limit + cache)
  ❌ ไม่ต้อง Kafka, ไม่ต้อง Tempo

Phase 2 (Production, vLLM):
  ✅ + Tempo (tracing)
  ✅ + River (Go job queue บน Postgres) สำหรับ RAG ingest
  ✅ + Redis sentinel (HA)

Phase 3 (Scale ใหญ่):
  ✅ + พิจารณา Kafka ถ้ามี event streaming requirement
  ✅ + Multi-region replication
```

---

## 📌 Summary table

| ลำดับ | หัวข้อ | จำนวน items | Effort | Risk if skip |
|---|---|---|---|---|
| P0 | Security/Correctness | 8 | M | 🔴 Cannot go prod |
| P1 | Reliability/Observability | 7 | M-L | 🟠 Outage risk |
| P2 | Code quality | 10 | S-M | 🟡 Tech debt |
| P3 | Scale architecture | 3 | L | 🟢 Phase 2+ |

**ดูแผน sprint แบบละเอียดที่:** `08-PRODUCTION-PLAN.md`
