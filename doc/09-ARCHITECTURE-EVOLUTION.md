# Architecture Evolution — PoC (Ollama) → Production (vLLM)

> **Created:** 2026-04-29
> **Goal:** วางสถาปัตยกรรมให้ย้าย Ollama → vLLM ได้โดย "แก้ไม่รื้อ"
> **Use cases:**
> 1. ถามตอบเรื่องทั่วไป (general chat)
> 2. ถามข้อมูลปัจจุบันในเน็ต (web search / live data)
> 3. ถามข้อมูลในองค์กร (RAG เอกสาร)
> 4. ระบบงานขาย (MCP / SQL Server)
> 5. ข้อมูลสำหรับผู้บริหาร (executive insights / analytics)

---

## 🎯 Design Principle: "Stable Interface, Swappable Implementation"

### Key insight ที่เก็บไว้ใช้ตลอด:

```
Domain (core business logic)
   ↓ depends on
Ports (interface contracts)   ← never change for swap
   ↑ implemented by
Adapters (Ollama / vLLM / Tavily / SQL Server)  ← swap freely
```

**กฎ 3 ข้อ:**
1. **Domain ห้ามรู้จัก HTTP/SQL/Vendor** — ห้าม import `net/http`, `database/sql`, vendor SDK
2. **Adapters ห้ามมี business logic** — แค่แปลง protocol → port interface
3. **Config drives wiring** — เปลี่ยน adapter ผ่าน env, ไม่ใช่ code change

ตอนนี้ codebase ทำได้ดีอยู่แล้วในเรื่องนี้ — เห็นได้จาก [llm.go](internal/domain/ports/llm.go) ที่ไม่ผูก vendor

---

## ✅ ส่วนที่ออกแบบไว้ดีแล้ว (เก็บไว้)

### 1. LLMPort design
[`internal/domain/ports/llm.go`](internal/domain/ports/llm.go) — interface ที่ครอบคลุม:
- `Chat()`, `ChatStream()`, `HealthCheck()`
- ทั้ง [`OllamaAdapter`](internal/infrastructure/llm/ollama.go) และ [`OpenAICompatibleAdapter`](internal/infrastructure/llm/openai_compatible.go) implement ครบ
- **vLLM = OpenAI-compatible** → adapter พร้อมใช้แล้ว เปลี่ยนแค่ env

### 2. MCPPort design
[`internal/domain/ports/mcp.go`](internal/domain/ports/mcp.go) — Tool calling abstraction:
- `ListTools()`, `Execute()`, `Ping()`
- Read-only validation ฝังใน adapter — ดี
- รองรับ multi-source (sales DB + executive DB ใช้ adapter คนละตัวได้)

### 3. Hexagonal layout
```
internal/
├── domain/           ← business model + ports (no I/O)
├── core/             ← orchestration (uses ports only)
└── infrastructure/   ← adapters (HTTP, SQL, LDAP)
```

โครงสร้างนี้ scale ได้ถึง production จริงโดยไม่ต้องรื้อ

---

## ❌ Gap ที่ต้องเติมเพื่อรองรับ 5 use cases

### Gap A — Web Search Port ยังไม่มี

**Use case:** "ถามข้อมูลปัจจุบันในเน็ต"

**ปัญหาปัจจุบัน:** Intent classifier มีแค่ `RAG`, `MCP`, `Direct` — ไม่มี `WebSearch`

**ออกแบบที่ควรเป็น:**

```go
// internal/domain/ports/web_search.go
type WebSearchPort interface {
    Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)
    HealthCheck(ctx context.Context) error
}

type SearchOptions struct {
    MaxResults int
    Recency    time.Duration  // เช่น "ข่าวใน 24 ชม."
    Region     string         // "TH" / "global"
    SafeSearch bool
}

type SearchResult struct {
    Title    string
    URL      string
    Snippet  string
    Source   string
    Date     time.Time
    Score    float64  // ranking score
}
```

**Adapter ที่ implement ได้:**
- **PoC:** `TavilyAdapter` (ฟรี 1000/เดือน, ออกแบบมาเพื่อ LLM พอดี)
- **Production option 1:** `BraveSearchAdapter` (จ่ายเป็น query, privacy-first)
- **Production option 2:** `SerperAdapter` (Google Search wrapper)
- **Enterprise:** self-host SearXNG + scraper

**Intent ใหม่:**
```go
const (
    IntentRAG       Intent = "rag"
    IntentMCP       Intent = "mcp"
    IntentWebSearch Intent = "web_search"  // ← เพิ่ม
    IntentDirect    Intent = "direct"
)
```

**Caching strategy:** web search results cache 1 ชม. ใน Redis/Postgres → ลด cost + เร็วขึ้น

---

### Gap B — Executive / Analytics Port

**Use case:** "ข้อมูลสำหรับผู้บริหาร"

ผู้บริหารถามแบบนี้ MCP SQL ดิบๆ ไม่พอ:
- "ยอดขายเดือนนี้เทียบเดือนที่แล้วเป็นอย่างไร?"
- "Top 5 product ที่กำไรเพิ่มขึ้นมากสุดไตรมาสนี้"
- "ทำไมรายได้สาขาอุดรลด?"

**ปัญหา:** SQL Server adapter ปัจจุบันให้ LLM เขียน SQL เอง → เสี่ยง:
- Hallucinated table names → query error
- ไม่รู้ business meaning ของ column
- ไม่รู้ว่า "ไตรมาส" บริษัทเริ่มเดือนไหน
- ตอบช้า เพราะ scan table ใหญ่

**ออกแบบที่ควรเป็น — เพิ่มชั้น "Semantic Layer":**

```go
// internal/domain/ports/analytics.go
type AnalyticsPort interface {
    // ListMetrics คืน metric ที่ pre-defined ไว้
    ListMetrics(ctx context.Context) ([]Metric, error)

    // Query ดึงข้อมูลตาม metric + filter (ไม่ใช่ SQL ดิบ)
    Query(ctx context.Context, req AnalyticsQuery) (AnalyticsResult, error)
}

type Metric struct {
    Name        string  // "monthly_revenue"
    Description string  // "ยอดขายรายเดือน รวม VAT"
    Dimensions  []string // ["branch", "product_category", "month"]
    Aggregation string  // "SUM" | "AVG" | "COUNT"
    Unit        string  // "THB"
}

type AnalyticsQuery struct {
    Metric     string
    Dimensions []string
    Filters    map[string]any  // {"branch": "udon", "month": ">=2026-01"}
    TimeRange  TimeRange
    OrderBy    string
    Limit      int
}
```

**Implementation strategy:**
- **PoC:** YAML/JSON file describe metrics + map ไป pre-built SQL templates
- **Production:** ใช้ semantic layer engine (Cube.js, Metabase API, dbt)
- **Best practice (BigTech):** materialized view ใน PostgreSQL + cache layer

**ตัวอย่าง flow:**
```
User: "ยอดขายอุดรเดือนนี้กี่บาท"
  ↓
Intent classifier → IntentAnalytics
  ↓
LLM → tool call: query(metric="monthly_revenue", filters={branch:"udon", month:"current"})
  ↓
AnalyticsPort.Query() → ใช้ pre-built query template (ปลอดภัย, เร็ว)
  ↓
LLM อธิบายตัวเลขเป็นภาษาไทย
```

---

### Gap C — Tool/Function Calling ใน LLMPort ยังไม่มี

**ปัญหา:** ปัจจุบัน `LLMPort.Chat()` รับแค่ messages ส่งกลับ string → LLM **เรียก tool ไม่ได้**

**สำคัญ เพราะ:**
- Web search, Analytics, MCP ทั้งหมดเป็น **tools ที่ LLM ต้องเรียก**
- Ollama (qwen2.5) รองรับ tool calling แล้ว
- vLLM รองรับ OpenAI-style function calling

**ออกแบบที่ควรเป็น:**

```go
// แทน Chat() เดิม → Chat() v2
type LLMPort interface {
    Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
    ChatStream(ctx context.Context, req ChatRequest, onEvent func(StreamEvent)) error
    HealthCheck(ctx context.Context) error
}

type ChatRequest struct {
    Messages    []ChatMessage
    Tools       []ToolDefinition  // ← เพิ่ม
    ToolChoice  string            // "auto" | "none" | "required"
    Temperature float32
    MaxTokens   int
}

type ChatResponse struct {
    Content   string
    ToolCalls []ToolCall   // ← LLM อาจขอเรียก tool
    Usage     TokenUsage
}

type StreamEvent struct {
    Type     string  // "content" | "tool_call" | "done"
    Content  string
    ToolCall *ToolCall
}
```

**Backward compat:** เก็บ method เดิมไว้เป็น helper ที่เรียก v2 ภายใน

---

### Gap D — Orchestrator ยังเป็น Simple Router

**ปัจจุบัน:** intent classify → ถ้า RAG เติม context → ส่ง LLM ครั้งเดียว

**ปัญหา:** flow แบบนี้ **agent หลาย step ไม่ได้** เช่น:
- ถาม "ยอดขายเดือนนี้เป็นอย่างไร เทียบกับ industry benchmark?"
- ต้อง: query DB → search web → รวมข้อมูล → ตอบ

**ออกแบบที่ควรเป็น — Agent Loop (ReAct pattern):**

```
┌─────────────────────────────────────────┐
│ Orchestrator.Run(query)                 │
│                                          │
│  ┌─────────────────────────────────┐    │
│  │ Loop (max N iterations):        │    │
│  │   1. LLM ตัดสินใจว่าใช้ tool ไหน  │    │
│  │   2. ถ้า tool_call → execute    │    │
│  │   3. ถ้า answer → return        │    │
│  │   4. Loop กลับไปข้อ 1            │    │
│  └─────────────────────────────────┘    │
└─────────────────────────────────────────┘
```

**Tools ที่ register ใน orchestrator:**
- `search_documents(query)` — RAG
- `query_database(metric, filters)` — Analytics
- `query_sales(query)` — MCP SQL Server
- `web_search(query)` — Web Search
- `get_current_time()` — utility

**ข้อดี:**
- LLM ตัดสินใจเอง (smart routing) แทน rule-based intent
- รองรับ multi-step reasoning
- Code orchestrator ไม่ต้องเปลี่ยนเมื่อเพิ่ม tool ใหม่

**ความเสี่ยง:**
- Loop ไม่หยุด → ต้องมี max iterations + timeout
- Cost สูงกว่า (LLM call หลายครั้ง)
- ต้อง guard tool args (LLM อาจ hallucinate)

**Hybrid approach (แนะนำ):**
- Simple query → keyword intent → 1-shot (เร็ว, ถูก)
- Complex query → agent loop (ช้า แต่ฉลาด)
- Threshold: ความยาวคำถาม / มีคำว่า "เปรียบเทียบ", "วิเคราะห์", "ทำไม"

---

### Gap E — Conversation Memory ตื้นเกินไป

**ปัจจุบัน:** Sliding window 10 messages

**ปัญหา:**
- ผู้ใช้ถามต่อ 30 รอบ → ลืม context รอบแรก
- ไม่มี long-term memory ("ผมเคยถามเรื่องนี้ไปแล้ว")

**ออกแบบที่ควรเป็น — 3 layers:**

```
┌─ Working memory (current request) ────┐
│  - ข้อความล่าสุด 10 messages          │
│  - Tool call history ใน turn นี้      │
└────────────────────────────────────────┘
            ↓
┌─ Session memory (current chat session)┐
│  - Summary ของ messages เก่ากว่า 10  │
│  - User preferences ใน session       │
└────────────────────────────────────────┘
            ↓
┌─ Long-term memory (cross-session) ────┐
│  - User profile (department, role)    │
│  - Past Q&A ที่ relevant (vector)     │
└────────────────────────────────────────┘
```

**Implementation roadmap:**
- **Phase 1:** เพิ่ม session summary — ทุกๆ 10 messages, LLM สรุปเป็น 1 paragraph เก็บใน sessions table
- **Phase 2:** Vector store สำหรับ user past queries — ใช้ pgvector ตัวเดิมได้

---

### Gap F — Prompt Management แบบ scale ไม่ได้

**ปัจจุบัน:** System prompt เป็น string เดียวใน config

**ปัญหา:**
- 5 use case ต้องการ prompt ต่างกัน (general/web/RAG/sales/exec)
- เปลี่ยน prompt ต้อง redeploy
- ไม่มี A/B test

**ออกแบบที่ควรเป็น:**

```go
// internal/core/prompt/
type PromptManager interface {
    Get(name string, vars map[string]any) (string, error)
    List() []PromptInfo
}

// File-based template (PoC)
// Database-backed (Production) — แก้ผ่าน admin UI ได้
```

**File structure:**
```
prompts/
├── system_general.tmpl
├── system_rag.tmpl
├── system_sales.tmpl
├── system_executive.tmpl
└── tool_descriptions.tmpl
```

**ข้อดี:**
- Edit prompt ไม่ redeploy
- Version control
- A/B test ผ่าน config

---

### Gap G — Document Management ยังไม่ครบ

**ปัจจุบัน:** ingest แล้วเก็บใน `documents` table — มี list/delete รอบแรกแล้ว แต่ยังไม่มี:
- Update document (re-ingest)
- Delete document
- Soft delete + version
- Tag / category / department

**ขาดสำหรับ enterprise:**
```go
type DocumentPort interface {
    Store(ctx, doc) error
    Get(ctx, id) (Document, error)
    List(ctx, filter) ([]Document, error)
    Update(ctx, id, doc) error
    Delete(ctx, id) error  // soft delete
}
```

ตาราง `documents` ต้องเพิ่ม:
- `uploaded_by` (user_id)
- `department` (สำหรับ filter ตอน search)
- `tags` (array)
- `version` + `deleted_at`
- `source_metadata` (JSONB) — original filename, file_hash, etc.

---

## 🏗️ Target Architecture (after evolution)

```
┌──────────────────────────────────────────────────────────────┐
│                      API Layer (Fiber)                        │
│  /v1/chat | /v1/rag | /v1/admin | /v1/documents | /metrics  │
└──────────────────────────────────────────────────────────────┘
                            ↓
┌──────────────────────────────────────────────────────────────┐
│                  Application Services                          │
│  ┌───────────────────────────────────────────────────────┐   │
│  │ Orchestrator (Agent Loop)                             │   │
│  │  • Intent Router (hybrid: keyword + LLM)             │   │
│  │  • Tool Executor                                      │   │
│  │  • Conversation Manager (3-layer memory)             │   │
│  └───────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
                            ↓ uses
┌──────────────────────────────────────────────────────────────┐
│                       Ports (Interfaces)                      │
│   LLMPort  │  EmbedderPort  │  VectorPort  │  SessionPort   │
│   AuditPort│  WebSearchPort │  AnalyticsPort  │  MCPPort   │
│              DocumentPort   │  PromptPort  │  CachePort     │
└──────────────────────────────────────────────────────────────┘
                            ↑ implemented by
┌──────────────────────────────────────────────────────────────┐
│                       Adapters                                │
│  ┌─────────────────────┐    ┌─────────────────────────┐     │
│  │ LLM:                │    │ Search:                  │     │
│  │  • Ollama (PoC)     │    │  • Tavily (PoC)         │     │
│  │  • vLLM (Prod)      │    │  • Brave (Prod)         │     │
│  └─────────────────────┘    └─────────────────────────┘     │
│  ┌─────────────────────┐    ┌─────────────────────────┐     │
│  │ Embedder:           │    │ Analytics:               │     │
│  │  • Nomic via Ollama │    │  • SQL Templates (PoC)  │     │
│  │  • vLLM embedding   │    │  • Cube.js (Prod)       │     │
│  └─────────────────────┘    └─────────────────────────┘     │
│  ┌─────────────────────┐    ┌─────────────────────────┐     │
│  │ Vector / Session /  │    │ MCP:                     │     │
│  │ Audit / Document:   │    │  • SQL Server (Sales)   │     │
│  │  • PostgreSQL       │    │  • Future: API/REST     │     │
│  └─────────────────────┘    └─────────────────────────┘     │
└──────────────────────────────────────────────────────────────┘
```

---

## 📦 Recommended Folder Structure (target)

```
internal/
├── domain/
│   ├── models/
│   └── ports/
│       ├── llm.go                ✅ มีแล้ว
│       ├── embedder.go           ✅ มีแล้ว
│       ├── vector.go             ✅ มีแล้ว
│       ├── session.go            ✅ มีแล้ว
│       ├── audit.go              ✅ มีแล้ว
│       ├── mcp.go                ✅ มีแล้ว
│       ├── web_search.go         ❌ เพิ่ม
│       ├── analytics.go          ❌ เพิ่ม
│       ├── document.go           ❌ เพิ่ม (CRUD)
│       ├── prompt.go             ❌ เพิ่ม
│       └── cache.go              ❌ เพิ่ม
│
├── core/
│   ├── orchestrator/
│   │   ├── orchestrator.go       ✅ มี (refactor → agent loop)
│   │   ├── tool_registry.go      ❌ เพิ่ม
│   │   └── memory.go             ❌ เพิ่ม (3-layer)
│   ├── intent/
│   │   ├── classifier.go         ✅ มี (keep + add hybrid)
│   │   └── llm_classifier.go     ❌ เพิ่ม (fallback)
│   ├── rag/
│   │   ├── rag.go                ✅ มี
│   │   ├── chunker.go            ✅ มี
│   │   ├── parser.go             ✅ มี
│   │   ├── reranker.go           ❌ เพิ่ม
│   │   └── hybrid_search.go      ❌ เพิ่ม
│   ├── prompt/
│   │   ├── manager.go            ❌ เพิ่ม
│   │   └── templates/            ❌ เพิ่ม (.tmpl files)
│   └── tools/                    ❌ เพิ่ม (folder ใหม่)
│       ├── rag_tool.go
│       ├── analytics_tool.go
│       ├── web_search_tool.go
│       └── mcp_tool.go
│
├── infrastructure/
│   ├── llm/
│   │   ├── ollama.go             ✅ PoC
│   │   ├── openai_compatible.go  ✅ vLLM-ready
│   │   └── http_client.go        ❌ shared (retry/breaker)
│   ├── embedder/
│   │   ├── nomic.go              ✅ มี
│   │   └── openai.go             ❌ เพิ่ม (vLLM embeddings)
│   ├── vector/
│   │   └── pgvector.go           ✅ มี
│   ├── search/
│   │   ├── tavily.go             ❌ เพิ่ม
│   │   └── brave.go              ❌ เพิ่ม (production)
│   ├── analytics/
│   │   ├── sql_template.go       ❌ เพิ่ม (PoC)
│   │   └── cube.go               ❌ เพิ่ม (production)
│   ├── mcp/
│   │   ├── sqlserver.go          ✅ มี
│   │   └── rest_api.go           ❌ เพิ่ม (อนาคต)
│   ├── document/
│   │   └── postgres.go           ❌ เพิ่ม
│   ├── cache/
│   │   ├── redis.go              ❌ Phase 2
│   │   └── memory.go             ❌ PoC (in-process LRU)
│   ├── auth/                     ✅ มี
│   ├── audit/                    ✅ มี
│   ├── session/                  ✅ มี
│   └── migrations/               ✅ มี
│
└── api/
    ├── handlers/
    │   ├── chat.go               ✅ มี
    │   ├── auth.go               ✅ มี
    │   ├── rag.go                ✅ มี
    │   ├── audit.go              ✅ มี
    │   ├── health.go             ✅ มี
    │   ├── documents.go          ❌ เพิ่ม (CRUD)
    │   └── admin.go              ❌ เพิ่ม (prompt mgmt)
    └── middleware/               ✅ มี
```

---

## 🚀 Migration Path: Ollama → vLLM (zero-downtime)

### Step 1: ก่อน vLLM hardware มา (ตอนนี้)
**ทำได้เลย:**
- ทุก use case ใช้ `LLM_BACKEND=ollama` ตามเดิม
- Test `OpenAICompatibleAdapter` กับ vLLM container บน dev machine
- ใส่ feature flag `LLM_USE_TOOLS=true/false` (ค่อยๆเปิด)
- เพิ่ม web search adapter + analytics layer (ไม่กระทบ LLM)

### Step 2: vLLM มาถึง — Canary deployment
```yaml
# docker-compose
orchestrator-v1:
  environment:
    LLM_BACKEND: ollama
    LLM_HOST: ollama-server

orchestrator-v2:  # canary 10% traffic
  environment:
    LLM_BACKEND: vllm
    LLM_HOST: vllm-server

# nginx/traefik split traffic 90/10 → 50/50 → 100/0
```

**สิ่งที่เปลี่ยน:** แค่ env variables — code ไม่แตะ

### Step 3: เก็บ Ollama ไว้เป็น fallback
```
Primary: vLLM (fast, big model)
Fallback: Ollama (ถ้า vLLM ตาย)
```

ใช้ pattern **multi-LLM router**:
```go
// internal/infrastructure/llm/router.go
type RouterAdapter struct {
    primary  ports.LLMPort
    fallback ports.LLMPort
    breaker  *gobreaker.CircuitBreaker
}
```

### Step 4: Multi-model routing (Phase 2.5)
```go
type ModelRouter interface {
    SelectModel(intent Intent, complexity float64) ports.LLMPort
}

// Cheap intent classify → small model
// Generation → big model
// Embedding → Nomic
```

---

## 🔧 What's missing for PoC (ทำเพิ่มก่อน prod)

จัดลำดับตาม **business value × effort**:

### Must-have ก่อนเปิด user (ตามลำดับ):

1. **Web Search Port + Tavily adapter** (effort: S, value: H)
   - ปลด use case "ข้อมูลปัจจุบันในเน็ต"
   - ใช้ Tavily ฟรี tier ก่อน

2. **Tool calling ใน LLMPort** (effort: M, value: H)
   - ปลด agent capabilities
   - Backward compat ผ่าน optional field

3. **Document CRUD** (effort: S, value: H)
   - ✅ list/delete ตาม source ทำแล้ว
   - ยังเหลือ update/re-ingest, versioning, soft delete

4. **Prompt management** (effort: S, value: M)
   - 5 use case prompt ต่างกัน
   - ไม่ต้อง redeploy ตอนปรับคำพูด

5. **Analytics Port + SQL templates** (effort: M, value: H)
   - ปลอดภัยกว่าให้ LLM เขียน SQL ดิบ
   - ตอบเร็วกว่า (cached materialized view)

### Nice-to-have (Phase 1.5):

6. **Hybrid search (BM25 + vector)** (effort: M, value: M)
7. **Reranker** (effort: M, value: M)
8. **Session summary** (effort: S, value: M)
9. **Multi-LLM router** (effort: M, value: L) — รอจน vLLM พร้อม

### Phase 2 (Production):

10. **Agent loop** (effort: L, value: H)
11. **Long-term memory (vector)** (effort: M, value: M)
12. **Cube.js / semantic layer** (effort: L, value: H)
13. **Multi-tenant isolation** (effort: M, value: H)

---

## 🎯 Use Case → Component Mapping

| Use case | Required components | Status |
|---|---|---|
| ถามทั่วไป | LLM | ✅ พร้อม |
| ข้อมูลในเน็ต | LLM + WebSearchPort + Cache | ❌ ขาด search |
| ข้อมูลองค์กร | LLM + RAG + Document CRUD | 🟡 RAG พร้อม, มี list/delete, ยังขาด versioning |
| ระบบงานขาย | LLM + Tool calling + MCP | 🟡 MCP พร้อม, ขาด tool calling |
| ผู้บริหาร | LLM + Tool calling + Analytics + RAG + WebSearch | 🟡 ครึ่งทาง |

---

## 🛡️ Anti-patterns ที่ต้องระวัง (production lessons)

### 1. ❌ ใส่ vendor SDK ใน core
```go
// BAD — core import openai SDK
package orchestrator
import "github.com/sashabaranov/go-openai"
```
```go
// GOOD — core ใช้ port
import "internal/domain/ports"
func (o *Orchestrator) Chat(... ports.LLMPort ...)
```

### 2. ❌ Pass adapter ลงลึกแทน port
```go
// BAD
func newRAG(ollama *llm.OllamaAdapter) ...
```
```go
// GOOD
func newRAG(llm ports.LLMPort) ...
```

### 3. ❌ Config struct ใหญ่ส่งผ่านทุก layer
```go
// BAD
func NewHandler(cfg *AppConfig) — handler ไม่ควรรู้ทั้ง config
```
```go
// GOOD — pick เฉพาะที่ต้องใช้
func NewHandler(timeout time.Duration, model string)
```

### 4. ❌ HTTP client สร้างใหม่ทุก request
ปัจจุบันใน `parser.go:252` สร้าง `http.Client{}` ใหม่ทุกหน้า OCR — leak connection
**แก้:** shared client ผ่าน adapter constructor

### 5. ❌ Hardcoded prompts ในหลายที่
ปัจจุบัน system prompt อยู่ใน `config.go`, `orchestrator.go`, `parser.go` (OCR prompt)
**แก้:** centralize ใน prompt manager

### 6. ❌ Sync I/O ใน hot path
RAG ingest → embed sync — request ค้างได้นาน
**แก้:** async job queue + status polling

---

## 📋 Decision Matrix

### Web search vendor
| Option | Free tier | Cost (1k queries) | Privacy | Quality |
|---|---|---|---|---|
| **Tavily** | 1000/mo | $5 | Good | LLM-optimized ⭐ |
| **Brave** | 2000/mo | $5 | Excellent | Good |
| **Serper** | 2500 trial | $0.30 | Avg | Google-grade ⭐ |
| **SearXNG** | unlimited | free (self-host) | Best | Avg |

**Recommendation:** Tavily สำหรับ PoC, Brave สำหรับ prod (privacy-conscious enterprise)

### Analytics layer
| Option | Setup | Maintenance | Power |
|---|---|---|---|
| **YAML SQL templates** | S | S | Low |
| **Cube.js** | M | M | High ⭐ |
| **dbt + materialized views** | L | M | High |
| **Metabase API** | M | S | Med |

**Recommendation:** YAML templates ก่อน → Cube.js เมื่อ metric > 20 ตัว

### LLM library
| Option | Pros | Cons |
|---|---|---|
| **Pure HTTP (current)** | No deps, full control ⭐ | Write boilerplate |
| **langchaingo** | Many integrations | Heavy, breaking changes บ่อย |
| **eino (Bytedance)** | Production-grade | New, English doc น้อย |

**Recommendation:** เก็บ pure HTTP — โครงสร้างปัจจุบันดีอยู่แล้ว

---

## 🔚 Summary

**สถาปัตยกรรมเดิมดีพอ — ไม่ต้องรื้อ ทำ 6 อย่างนี้พอ:**

1. ✅ **เก็บ:** ports/adapters layout, hexagonal structure
2. ➕ **เพิ่ม port ใหม่:** WebSearch, Analytics, Document, Prompt, Cache
3. 🔧 **ขยาย LLMPort:** เพิ่ม tool calling support (backward compat)
4. 🔧 **Refactor Orchestrator:** เป็น agent loop (เก็บ rule-based intent ไว้เป็น fast path)
5. ➕ **Tool registry:** map tool name → port + execution
6. ➕ **Memory layers:** working / session / long-term

**Migration Ollama → vLLM:**
- เปลี่ยน 2 env vars (`LLM_BACKEND`, `LLM_HOST`)
- Code 0 บรรทัด
- Canary deploy ได้ผ่าน docker-compose / load balancer

**Time to "production v1":** ~6 sprints (ดู `08-PRODUCTION-PLAN.md` + เพิ่ม sprint เฉพาะ web search/analytics)

---

## 🔗 Related docs

- `07-CODE-REVIEW.md` — gap analysis ระดับ code
- `08-PRODUCTION-PLAN.md` — sprint roadmap (security/reliability/observability)
- เอกสารนี้เน้น **architecture evolution** — เสริม sprint plan ด้วย sprint ใหม่:
  - **Sprint 6.5 — Web Search & Analytics**
  - **Sprint 7.5 — Agent Loop Refactor**
