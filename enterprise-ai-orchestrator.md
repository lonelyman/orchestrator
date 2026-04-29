# Enterprise AI Orchestrator — Living Document
> **Last Updated:** 2026-04-29
> **Status:** Phase 3 Done — เตรียม Phase 4 Production
> **วิธีใช้:** ทุกครั้งที่เปิด session ใหม่ ให้ paste document นี้ให้ Claude ดูก่อน แล้วจะ update ต่อได้เลย
> **GitHub:** https://github.com/lonelyman/orchestrator.git (branch: dev)

---

## 1. Project Overview

สร้าง **Enterprise AI Orchestrator** ด้วยภาษา Go เพื่อเป็นสมองกลางขององค์กร เชื่อมต่อข้อมูล (Multi-DB) และใช้เครื่องมืออัตโนมัติ (MCP) โดยมีความยืดหยุ่นจาก PoC ไปสู่ Production

---

## 2. Hardware

| Role | Machine | Spec |
|---|---|---|
| **PoC Server** | iMac | Apple M1, 16GB RAM |
| **Dev Client** | MacBook Pro | sysadmins-MacBook-Pro-2 |
| **Production (อนาคต)** | Ubuntu Server | NVIDIA Blackwell |

**การเข้าถึง iMac Server:**
```bash
ssh imac_testai@10.19.105.32
```

---

## 3. Architecture

```
MacBook (Dev/Client)
    │ SSH / Postman / curl
    ▼
iMac M1 16GB (PoC Server)
    │
    ├── Ollama (Native M1 Metal)
    │       ├── qwen2.5:7b        ← LLM หลัก (4.7GB)
    │       └── nomic-embed-text  ← Embedding (274MB)
    │
    └── Docker Compose
            ├── Go Orchestrator (Port 50000)
            ├── PostgreSQL + pgvector (Port 15433)
            └── Open WebUI (Port 53000) ← ปิดอยู่ชั่วคราว
```

### Request Flow

```
User → Postman / curl
    ↓
Go Orchestrator (Port 50000)
    ↓
Intent Classifier
    ↓
┌──────────┬──────────┬──────────┐
↓          ↓          ↓
RAG        MCP       Direct
(pgvector) (SQL Svr)  (ตรงๆ)
└──────────┴──────────┴──────────┘
                ↓
         Ollama (qwen2.5:7b)
                ↓
           คำตอบภาษาไทย
```

### Port Map

| Service | Port | หมายเหตุ |
|---|---|---|
| Ollama | 11434 (internal) | Native on M1 |
| Go Orchestrator | 50000 | Backend API |
| PostgreSQL | 15433 (external) / 5432 (internal) | Vector DB |
| Open WebUI | 53000 | ปิดอยู่ชั่วคราว |

### API Endpoints

| Method | Endpoint | หน้าที่ |
|---|---|---|
| GET | /health | ตรวจสอบสถานะ |
| POST | /v1/chat/completions | Chat (OpenAI-compatible) |
| POST | /v1/rag/ingest | อัพโหลด text ตรงๆ |
| POST | /v1/rag/upload | อัพโหลดไฟล์ (PDF/TXT/MD) |

---

## 4. Tech Stack

| ด้าน | PoC | Production |
|---|---|---|
| **Language** | Go 1.26 | Go 1.26 |
| **Framework** | Fiber v3 | Fiber v3 |
| **LLM Engine** | Ollama (Native M1) | vLLM (Docker, Ubuntu) |
| **Primary Model** | qwen2.5:7b | qwen2.5:32B MoE |
| **Embedding** | nomic-embed-text | nomic / OpenAI (สลับได้) |
| **Vector DB** | pgvector (PostgreSQL 16) | pgvector (PostgreSQL 16) |
| **Architecture** | Hexagonal Architecture | Hexagonal Architecture |
| **Session** | Isolated per user | Shared Knowledge Base (อนาคต) |

### Key Design Principles
- **Migration-Ready:** ทุก connection ผ่าน Interface → สลับ Ollama → vLLM ได้ใน `.env`
- **Zero-Trust MCP:** SQL Server → Read-only + Validation Layer เสมอ
- **Gateway-Ready:** รองรับ Header `X-User-ID`, `X-Session-ID` ตั้งแต่แรก
- **Standard Response:** `{"data":{}}` หรือ `{"error":{}}` เสมอ
- **Prompt Language:** English System Prompt / Thai Response

---

## 5. Codebase Structure

```
orchestrator/
├── Dockerfile
├── docker-compose.yml
├── .env.example
├── .gitignore
├── README.md
├── go.mod / go.sum
├── cmd/server/main.go
├── config/config.go
├── docker/init/01-extensions.sql
└── internal/
    ├── api/handlers/
    │   ├── response.go
    │   ├── health.go
    │   ├── chat.go
    │   └── rag.go
    ├── domain/
    │   ├── models/
    │   │   ├── chat.go
    │   │   ├── document.go
    │   │   ├── mcp.go
    │   │   └── intent.go
    │   └── ports/
    │       ├── llm.go
    │       ├── embedder.go
    │       ├── vector.go
    │       └── mcp.go
    ├── core/
    │   ├── intent/classifier.go
    │   ├── orchestrator/orchestrator.go
    │   ├── rag/
    │   │   ├── rag.go
    │   │   ├── chunker.go
    │   │   └── parser.go
    │   └── mcp/executor.go
    └── infrastructure/
        ├── llm/ollama.go
        ├── embedder/nomic.go
        ├── vector/pgvector.go
        └── mcp/sqlserver.go
```

---

## 6. Intent Router

| Intent | Trigger Keywords | Action |
|---|---|---|
| `rag` | นโยบาย, ประกาศ, เอกสาร, OT, ล่วงเวลา, ลา ฯลฯ | ค้นหาจาก pgvector |
| `mcp` | ยอดขาย, สต็อก, รายงาน, เงินเดือน ฯลฯ | Query SQL Server |
| `direct` | ไม่เจอ keyword | ตอบตรงๆ |

**TODO Phase 3.5:** LLM-based classifier (แม่นยำกว่า rule-based)

---

## 7. Roadmap

| Phase | หัวข้อ | สถานะ |
|---|---|---|
| **Phase 0** | Foundation: Ollama + Docker + Go + WebUI | ✅ Done |
| **Phase 1** | RAG: Vector Ingestion + Search + PDF Upload | ✅ Done |
| **Phase 2** | MCP: SQL Server Bridge Structure | ✅ Structure Ready |
| **Phase 3** | Orchestration: Intent Router | ✅ Done |
| **Phase 3.5** | Session Management + Auth + WebUI integration | 🔜 Next |
| **Phase 4** | Production: Ubuntu + vLLM + Blackwell | ⏳ |

---

## 8. Known Issues & TODO

| Issue | Status | แนวทางแก้ |
|---|---|---|
| Scanned PDF อ่านไม่ออก | ✅ แก้แล้ว | tesseract OCR ใน Dockerfile |
| pgvector Search ได้ 0 docs | ✅ แก้แล้ว | fallback query ใน pgvector.go |
| SQL Server ยังไม่ได้เชื่อมต่อจริง | ⏳ รอ SQL Server | sqlserver.go พร้อมแล้ว |
| Session Management ยังไม่มี | 🔜 Phase 3.5 | ต้องสร้าง session store |
| Open WebUI ยังไม่ผ่าน Orchestrator | 🔜 Phase 3.5 | เปลี่ยน WebUI endpoint |
| Authentication ยังไม่มี | 🔜 Phase 3.5 | JWT หรือ API Key |

---

## 9. Configuration

### Ollama (Native on iMac)
```
Service: brew services start ollama
Env:     OLLAMA_FLASH_ATTENTION=1
         OLLAMA_KV_CACHE_TYPE=q8_0
Port:    11434
```

### .env
```env
API_PORT=50000
LLM_BACKEND=ollama
LLM_HOST=host.docker.internal
LLM_PORT=11434
LLM_MODEL=qwen2.5:7b
DB_HOST=orchestrator-postgres
DB_PORT=5432
DB_NAME=orchestrator
DB_USER=orchestrator
DB_PASS=changeme
SYSTEM_PROMPT=You are a helpful enterprise AI assistant. You must always respond in Thai language only.
```

### Docker Commands
```bash
docker compose up --build -d postgres orchestrator
docker logs orchestrator-api --tail 20
docker compose down
```

---

## 10. Decisions Log

| วันที่ | การตัดสินใจ | เหตุผล |
|---|---|---|
| 2026-04-28 | Ollama Native | รีด M1 Metal GPU ได้เต็มที่ |
| 2026-04-28 | qwen2.5:7b | Sweet spot บน 16GB |
| 2026-04-28 | nomic-embed-text | เบา เร็ว Matryoshka Embeddings |
| 2026-04-28 | Hexagonal Architecture | สลับ Ollama → vLLM โดยไม่ rewrite |
| 2026-04-29 | Standard Response Format | มาตรฐานบริษัท: data/error เท่านั้น |
| 2026-04-29 | tesseract OCR ใน Docker | รองรับ Scanned PDF ภาษาไทย |
| 2026-04-29 | pgvector auto-init via SQL | ไม่ต้องรัน CREATE EXTENSION ด้วยมือ |
| 2026-04-29 | Rule-based Intent Classifier | เริ่มง่าย ค่อย upgrade เป็น LLM-based |

---

## 11. สิ่งที่ยังขาด — ก่อน Production

### บริษัท: นูทริชั่น โปรเฟส (อาหารเสริม)
Use Case หลัก 2 กลุ่ม:

```
กลุ่มที่ 1: คำถามทั่วไปองค์กร
→ นโยบาย, สวัสดิการ, ระเบียบ, ประกาศ HR
→ ใช้ RAG (เอกสาร PDF/Word)

กลุ่มที่ 2: ข้อมูลสินค้าและธุรกิจ
→ สินค้า, stock, ราคา, order, รายงานผู้บริหาร
→ ใช้ MCP (API/Database)
```

---

### 11.1 RAG — Flow เอกสารใหม่ (ยังขาด)

**ปัจจุบัน:** Upload ทีละไฟล์ผ่าน API เท่านั้น

**ที่ต้องเพิ่ม:**

```
Flow เอกสารใหม่เข้าระบบ:

1. มีเอกสารใหม่ (PDF/Word/Excel)
        ↓
2. วางไว้ใน folder ที่กำหนด
   หรือ Upload ผ่าน Web UI
        ↓
3. ระบบ parse → chunk → embed
        ↓
4. บันทึกลง pgvector พร้อม metadata:
   - ชื่อไฟล์
   - วันที่อัพโหลด
   - ประเภทเอกสาร (HR/Product/Policy)
   - version
        ↓
5. แจ้ง Admin ว่า ingest สำเร็จ
        ↓
6. AI พร้อมตอบจากเอกสารใหม่ทันที
```

**TODO:**
- [ ] Web UI สำหรับ Admin อัพโหลดเอกสาร
- [ ] Batch ingest (อัพหลายไฟล์พร้อมกัน)
- [ ] Document versioning (เอกสารเก่า vs ใหม่)
- [ ] Document management (ลบ/แก้ไข/ดูรายการ)
- [ ] Endpoint: `GET /v1/rag/documents` ดูรายการเอกสารทั้งหมด
- [ ] Endpoint: `DELETE /v1/rag/documents/:id` ลบเอกสาร
- [ ] รองรับ Word (.docx) และ Excel (.xlsx)

---

### 11.2 MCP / API Integration (ยังขาดทั้งหมด)

**ปัจจุบัน:** SQLServerAdapter โครงสร้างพร้อม แต่ยังไม่ได้เชื่อมต่อจริง

**Use Cases ที่ต้องเพิ่ม:**

#### Phase A — ข้อมูลสินค้า (Read-only)
```
"สินค้า X มี stock เหลือเท่าไหร่?"
"ราคาสินค้า Y คืออะไร?"
"สินค้าตัวไหนขายดีที่สุดเดือนนี้?"
→ MCP → Product API / DB
```

#### Phase B — รายงานผู้บริหาร (Read-only)
```
"ยอดขายสัปดาห์นี้เป็นยังไง?"
"สรุปยอด order เดือนนี้"
"เปรียบเทียบยอดขาย Q1 vs Q2"
→ MCP → ERP / Sales DB → สร้าง Report
```

#### Phase C — Action (Write — ระวัง)
```
"เพิ่ม order สินค้า X จำนวน 100 ชิ้น"
→ MCP → Order API (ต้อง confirm ก่อนทุกครั้ง)
```

**TODO:**
- [ ] เชื่อมต่อ Product API / DB จริง
- [ ] เชื่อมต่อ ERP / Sales DB
- [ ] Tool: `get_product_stock` — ดู stock สินค้า
- [ ] Tool: `get_sales_report` — ดูรายงานยอดขาย
- [ ] Tool: `get_product_info` — ดูข้อมูลสินค้า
- [ ] Tool: `create_order` — สร้าง order (Phase C, ต้อง confirm)
- [ ] Confirmation Layer — ก่อน write ต้องให้ user confirm

---

### 11.3 Session Management (ยังขาด)

**ปัจจุบัน:** AI จำการสนทนาไม่ได้เลย ทุก message = เริ่มใหม่

**TODO:**
- [ ] เก็บ history ใน PostgreSQL
- [ ] Session expire (เช่น 30 นาที)
- [ ] Header `X-Session-ID` ที่ออกแบบไว้ตั้งแต่แรก

---

### 11.4 Authentication (ยังขาด)

**ปัจจุบัน:** ใครก็เรียก API ได้

**TODO:**
- [ ] API Key สำหรับ Service-to-Service
- [ ] JWT สำหรับ User login
- [ ] Role-based: Admin (อัพเอกสาร) vs User (แค่ถาม)

---

### 11.5 Open WebUI → Orchestrator (ยังไม่เชื่อมกัน)

**ปัจจุบัน:** WebUI คุยกับ Ollama ตรงๆ ไม่ผ่าน RAG/Intent Router

**TODO:**
- [ ] เปลี่ยน WebUI ให้ชี้มาที่ Go Orchestrator port 50000
- [ ] ทดสอบ Chat ผ่าน WebUI → RAG → คำตอบจากเอกสาร

---

### 11.6 Production Readiness (Phase 4)

**TODO:**
- [ ] ย้ายไป Ubuntu Server + NVIDIA Blackwell
- [ ] เปลี่ยนจาก Ollama → vLLM (แค่แก้ .env)
- [ ] เปลี่ยน Model: qwen2.5:7b → qwen2.5:32B
- [ ] Monitoring: Prometheus + Grafana (commented ใน docker-compose)
- [ ] golang-migrate แทน InitSchema()
- [ ] Load testing ก่อน production

---

## 12. Enterprise Considerations (บริษัท 1,000 คน / ตลาดหลักทรัพย์)

### 12.1 Vector DB Scaling Plan

```
ขนาดข้อมูลที่คาดการณ์:
เอกสาร HR + Product + Finance + Legal ≈ 1,700+ ไฟล์
Vector chunks ≈ 20,000-50,000 chunks
```

| ช่วง | Vector Size | แนวทาง |
|---|---|---|
| ตอนนี้ | < 50,000 | pgvector + IVFFlat Index |
| ระยะกลาง | 50,000-500,000 | pgvector + tune index + monitor |
| ระยะยาว | > 500,000 | พิจารณาย้าย Qdrant / Weaviate |

**TODO:**
- [ ] เพิ่ม IVFFlat Index ใน pgvector schema
- [ ] Monitor query time เมื่อข้อมูลเพิ่มขึ้น
- [ ] ทดสอบ performance ที่ 10,000 / 50,000 / 100,000 chunks

---

### 12.2 Security & Access Control (สำคัญมาก)

**ปัจจุบัน:** ใครก็เรียก API ได้ ไม่มีการแบ่งสิทธิ์

**ที่ต้องมี:**

```
Role ที่ต้องการ:
├── Admin     → อัพโหลดเอกสาร, จัดการระบบ
├── Manager   → ดูรายงาน, ถามข้อมูลธุรกิจ
└── Employee  → ถามนโยบาย HR, สวัสดิการ
```

**Document Access Control:**
```
เอกสารลับ (Confidential):
→ ผู้บริหารเท่านั้น
→ เช่น รายงานการเงิน, กลยุทธ์บริษัท

เอกสารทั่วไป (Internal):
→ พนักงานทุกคน
→ เช่น นโยบาย HR, สวัสดิการ

เอกสารสาธารณะ (Public):
→ ทุกคน
→ เช่น ข้อมูลสินค้า, แคตตาล็อก
```

**TODO:**
- [ ] เพิ่ม `role` field ใน Document model
- [ ] เพิ่ม `access_level` ใน pgvector schema
- [ ] JWT Authentication พร้อม role claims
- [ ] Filter documents ตาม role ก่อน Search
- [ ] Admin Panel สำหรับจัดการสิทธิ์เอกสาร

---

### 12.3 Audit Log (บริษัทตลาดหลักทรัพย์ต้องมี)

**ปัจจุบัน:** ไม่มี log เลย

**ที่ต้องมี:**
```
ทุก Query ต้องบันทึก:
- User ID ที่ถาม
- คำถามที่ถาม
- Intent ที่ตัดสินใจ (RAG/MCP/Direct)
- เอกสารที่ดึงมาใช้ (source)
- คำตอบที่ AI ให้
- เวลา + IP Address
```

**ทำไมต้องมี:**
```
✅ Compliance — ตลาดหลักทรัพย์กำหนด
✅ Security — ตรวจสอบถ้ามีการรั่วไหล
✅ Improve — ดูว่า AI ตอบผิดบ่อยไหม
✅ Legal — หลักฐานถ้ามีข้อพิพาท
```

**TODO:**
- [ ] สร้างตาราง `audit_logs` ใน PostgreSQL
- [ ] บันทึกทุก request/response
- [ ] Endpoint: `GET /v1/admin/logs` ดู log
- [ ] Log retention policy (เก็บกี่เดือน?)

---

### 12.4 Data Governance

**ที่ต้องกำหนดก่อน Deploy จริง:**

```
1. เอกสารไหนที่ AI "ห้าม" ตอบ?
   → เช่น ข้อมูลส่วนตัวพนักงาน, เงินเดือน

2. AI ตอบผิดแล้วทำยังไง?
   → มี Feedback mechanism

3. เอกสารหมดอายุแล้วทำยังไง?
   → ระบบ expire เอกสารเก่า

4. ใครเป็น AI Owner ในองค์กร?
   → คนรับผิดชอบ approve เอกสารที่ AI ใช้
```

**TODO:**
- [ ] กำหนด Data Classification policy
- [ ] สร้าง Feedback endpoint (`POST /v1/feedback`)
- [ ] Document expiry system
- [ ] กำหนด AI Owner role

---

### 12.5 Performance สำหรับ 1,000 Users

```
ถ้าพนักงาน 1,000 คนใช้พร้อมกัน:

Concurrent users: ~50-100 คน (ในเวลาเดียวกัน)
Request/minute:   ~200-500 requests

iMac M1 16GB รับได้ไหม?
→ PoC: ได้ แต่จะช้าถ้าหลายคนพร้อมกัน
→ Production: ต้องย้าย Ubuntu + vLLM + GPU
```

**TODO:**
- [ ] Load testing ก่อน deploy จริง
- [ ] Rate limiting per user
- [ ] Queue system ถ้า request เยอะ
- [ ] Horizontal scaling plan

---

## 13. Session Log — 2026-04-29

### สิ่งที่ทำสำเร็จ

#### Streaming Support ✅
- เพิ่ม Stream mode ใน `/v1/chat/completions`
- แก้ context canceled bug ใน Fiber SendStreamWriter
- WebUI เห็นตัวอักษรทยอยออกมา real-time

#### WebUI Integration ✅
- เพิ่ม `/v1/models` endpoint (OpenAI-compatible)
- เชื่อม Open WebUI → Go Orchestrator (port 50000)
- WebUI → Intent Router → RAG → Stream → คำตอบ ✅
- แก้ WebUI chat_history format parsing

#### RAG + Stream ✅
- ChatStream ใช้ RAG context เหมือน Chat
- Intent Classification ทำงานใน Stream mode
- ทดสอบ: "นโยบาย OT?" → RAG docs=3 → ตอบจากเอกสารจริง ✅

#### SSH Key บน iMac ✅
- สร้าง ed25519 key
- เพิ่มใน GitHub
- push ผ่าน SSH ได้แล้ว

### API Route ทั้งหมดตอนนี้

| Method | Endpoint | Format | หน้าที่ |
|---|---|---|---|
| GET | /health | Standard | ตรวจสถานะ |
| GET | /v1/models | OpenAI | Model list สำหรับ WebUI |
| POST | /v1/chat/completions | OpenAI | Chat + Stream |
| POST | /v1/rag/ingest | Standard | Upload text |
| POST | /v1/rag/upload | Standard | Upload file (PDF/TXT) |

### TODO ที่ยังค้างอยู่ (Phase 3.5)

- [ ] Session Management — จำประวัติการสนทนา
- [ ] Authentication — JWT / API Key
- [ ] Graceful Shutdown
- [ ] Structured Logging (slog)
- [ ] Health Check ครบทุก dependency
- [ ] Config Validation (Fail-fast)
- [ ] Audit Log
- [ ] Document Management UI