# Greenfield Stack — Industry-Standard Reference

> **Created:** 2026-04-29
> **Context:** ถ้าเริ่มใหม่จาก 0 โดยไม่ต้องห่วง legacy ของ PoC เดิม
> **Audience:** ผู้บริหาร (เน้น stack ที่ "ชื่อขายได้") + Engineering
> **Goal:** วาง stack ที่ดู professional, มี ecosystem ใหญ่, จ้าง engineer ได้ง่าย

---

## 🎯 Executive Summary (สำหรับนำเสนอผู้บริหาร)

**PoC ปัจจุบัน (Go + Fiber)** เป็น lean stack ที่ดี แต่:
- ❌ ไม่ใช่ stack มาตรฐานในวงการ AI/ML (ส่วนใหญ่ใช้ Python)
- ❌ Talent pool engineer สาย AI/Go เล็ก (Python มากกว่า 10 เท่า)
- ❌ Ecosystem AI library ใน Go ยังตามหลัง Python

**สิ่งที่วงการ Enterprise AI ใช้จริง (2026):**

```
Application Layer:    Python (FastAPI) | TypeScript (Next.js)
Agent Framework:      LangGraph | LlamaIndex | Pydantic AI
LLM Serving:          vLLM | TGI (Text Generation Inference)
Vector DB:            Qdrant | Weaviate | pgvector | Pinecone
Orchestration:        Kubernetes | Knative
Observability:        OpenTelemetry + Grafana LGTM stack
ML Tracking:          MLflow | Langfuse | LangSmith
Gateway:              LiteLLM | Portkey | AWS Bedrock
Data Pipeline:        Airflow | Dagster | Prefect
Feature Store:        Feast (optional)
```

**3 ทางเลือกที่แนะนำ:**

| Option | เหมาะกับ | Effort to rebuild |
|---|---|---|
| **A) Python + LangGraph** ⭐ | Standard enterprise AI | 2-3 เดือน |
| **B) TypeScript + Vercel AI SDK** | Fullstack เน้น UX | 2-3 เดือน |
| **C) Hybrid: Go API + Python AI service** | เก็บ infrastructure เดิม | 1-2 เดือน |

---

## 📊 Industry Landscape (สถานะ 2026)

### บริษัทที่ทำ Enterprise AI ใช้อะไร?

| บริษัท | Stack หลัก | Open source ที่ปล่อยออกมา |
|---|---|---|
| **OpenAI / ChatGPT Enterprise** | Python + Triton + K8s | tiktoken, evals |
| **Anthropic** | Python + JAX/PyTorch | (ไม่ปล่อยมาก) |
| **Microsoft Copilot** | Python (FastAPI) + .NET edge + Azure AI Foundry | Semantic Kernel, AutoGen |
| **Google (Gemini for Workspace)** | Python + Go + Vertex AI | ADK, genkit |
| **Meta** | Python (PyTorch) + Hack/PHP | Llama, FAISS |
| **Databricks** | Python + Scala + Spark | MLflow |
| **Bytedance (Doubao)** | Python + Go | eino (Go) |
| **Bloomberg, JP Morgan** | Python + Java + Spark | (internal) |

**Pattern ที่เห็น:**
- AI logic = **Python เกือบหมด**
- Infrastructure / scale = Go / Java / Rust
- Data pipeline = Python + Spark / Flink
- UI = TypeScript (Next.js / React)

---

## 🏆 Option A — Python + LangGraph (RECOMMENDED ⭐)

### Why
- **Industry standard 100%** — talent มาก, Stack Overflow / GitHub ตัวอย่างเยอะ
- **LangGraph จาก LangChain** — agent framework production-grade ที่ Anthropic, Microsoft, KLM ใช้
- **vLLM native Python** — integrate ง่ายสุด
- **Type-safe ผ่าน Pydantic** — แทบเทียบ Go ได้

### Stack เต็ม

```yaml
# Application
Language:           Python 3.12
Web framework:      FastAPI                          # เร็ว, async, OpenAPI auto
Validation:         Pydantic v2                      # type safety
Async runtime:      uvicorn (ASGI)
Package manager:    uv (10x faster than pip)         # by Astral

# AI / Agent
Agent framework:    LangGraph                        # stateful agent, ReAct
LLM library:        LangChain (light usage) + custom
LLM gateway:        LiteLLM                          # 100+ provider unified
Tracing/eval:       Langfuse (self-host) or LangSmith
Observability:      OpenTelemetry + Grafana Tempo

# LLM serving
PoC:                Ollama (เหมือนเดิม)
Production:         vLLM (OpenAI-compatible API)
Embedding:          BGE-M3 / Nomic via vLLM

# Storage
Relational:         PostgreSQL 17
Vector:             pgvector (start) → Qdrant (scale)
Cache:              Redis 7
Object store:       MinIO / S3 (ไฟล์ user upload)
Search (BM25):      PostgreSQL FTS หรือ OpenSearch

# Auth
Identity provider:  Keycloak (self-host) หรือ Authentik
Protocol:           OIDC (OpenID Connect)            # standard ที่ AD รองรับ
Tokens:             JWT (short-lived) + refresh

# Background jobs
Queue:              Celery + Redis    หรือ
                    ARQ + Redis       (async-native)
Scheduler:          APScheduler / Celery Beat

# Infrastructure
Container:          Docker
Orchestration:      Docker Compose (PoC) → Kubernetes (prod)
IaC:                Terraform / Pulumi
CI/CD:              GitHub Actions / GitLab CI
Registry:           Harbor (self-host) / GHCR

# Observability (LGTM stack — Grafana ecosystem)
Logs:               Loki
Metrics:            Mimir / Prometheus
Traces:             Tempo
Dashboard:          Grafana
APM:                OpenTelemetry collector

# AI-specific monitoring
LLM observability:  Langfuse (self-host) ⭐         # trace ทุก LLM call + cost + eval
Prompt management:  Langfuse Prompts หรือ PromptLayer
Eval framework:     DeepEval / Promptfoo

# Frontend (ถ้าทำ Web UI เอง)
Framework:          Next.js 15 (React 19)
UI library:         shadcn/ui + Tailwind CSS
Chat UI:            Vercel AI SDK / assistant-ui
State:              Zustand / TanStack Query
```

### Project Structure (FastAPI + LangGraph)

```
orchestrator/
├── pyproject.toml              # uv / poetry
├── docker-compose.yml
├── Dockerfile
│
├── src/
│   ├── api/
│   │   ├── main.py             # FastAPI app
│   │   ├── routes/
│   │   │   ├── chat.py
│   │   │   ├── auth.py
│   │   │   ├── rag.py
│   │   │   └── admin.py
│   │   └── middleware/
│   │       ├── auth.py
│   │       ├── rate_limit.py
│   │       └── tracing.py
│   │
│   ├── domain/                 # Pure business logic
│   │   ├── models.py           # Pydantic models
│   │   └── ports.py            # Protocol classes (interfaces)
│   │
│   ├── agents/                 # ⭐ LangGraph agents
│   │   ├── orchestrator.py     # Main agent graph
│   │   ├── nodes/
│   │   │   ├── intent.py
│   │   │   ├── retrieval.py
│   │   │   ├── tool_executor.py
│   │   │   └── responder.py
│   │   ├── tools/
│   │   │   ├── rag_tool.py
│   │   │   ├── web_search.py
│   │   │   ├── analytics.py
│   │   │   └── sales_db.py
│   │   └── state.py            # Graph state schema
│   │
│   ├── adapters/               # Infrastructure
│   │   ├── llm/
│   │   │   ├── litellm_client.py   # ⭐ unified gateway
│   │   │   └── ollama_direct.py    # fallback
│   │   ├── vector/
│   │   │   ├── pgvector_store.py
│   │   │   └── qdrant_store.py
│   │   ├── search/
│   │   │   ├── tavily.py
│   │   │   └── brave.py
│   │   ├── auth/
│   │   │   └── keycloak.py     # OIDC
│   │   └── db/
│   │       ├── session.py
│   │       └── audit.py
│   │
│   ├── core/
│   │   ├── config.py           # Pydantic Settings
│   │   ├── prompt/
│   │   │   ├── manager.py
│   │   │   └── templates/      # Jinja2 .j2
│   │   └── memory.py
│   │
│   └── workers/                # Celery tasks
│       ├── ingest.py
│       └── ocr.py
│
├── alembic/                    # DB migrations (เทียบ goose)
│   └── versions/
│
├── tests/
│   ├── unit/
│   ├── integration/
│   └── eval/                   # ⭐ LLM eval suite
│
└── prompts/                    # Version-controlled prompts
    ├── system_general.j2
    ├── system_rag.j2
    └── tools.j2
```

### Code style ที่ทำให้ "ดูสากล"

```python
# domain/ports.py — Protocol-based interface (เทียบ Go interface)
from typing import Protocol
from .models import ChatRequest, ChatResponse

class LLMPort(Protocol):
    async def chat(self, req: ChatRequest) -> ChatResponse: ...
    async def chat_stream(self, req: ChatRequest): ...
    async def health_check(self) -> bool: ...
```

```python
# agents/orchestrator.py — LangGraph
from langgraph.graph import StateGraph, END
from .state import AgentState

def build_agent_graph(llm, tools) -> StateGraph:
    graph = StateGraph(AgentState)

    graph.add_node("classify_intent", classify_node)
    graph.add_node("retrieve", retrieve_node)
    graph.add_node("call_tool", tool_node)
    graph.add_node("respond", responder_node)

    graph.set_entry_point("classify_intent")
    graph.add_conditional_edges(
        "classify_intent",
        route_by_intent,
        {"rag": "retrieve", "tool": "call_tool", "direct": "respond"},
    )
    graph.add_edge("retrieve", "respond")
    graph.add_edge("respond", END)

    return graph.compile(checkpointer=postgres_checkpointer)
```

```python
# adapters/llm/litellm_client.py — universal LLM gateway
import litellm

# ⭐ เปลี่ยน Ollama → vLLM → OpenAI → Anthropic ใน 1 บรรทัด
async def chat(messages: list[dict], model: str):
    return await litellm.acompletion(
        model=model,                    # "ollama/qwen2.5" → "openai/gpt-4" → "vllm/qwen-72b"
        messages=messages,
        api_base=settings.llm_base_url,
        timeout=settings.llm_timeout,
    )
```

### LiteLLM ทำไมสำคัญ
**LiteLLM = LLM gateway ที่ unify 100+ provider:**
- Ollama, vLLM, OpenAI, Anthropic, Bedrock, Vertex AI, Azure
- Built-in retry, fallback, load balancing
- Cost tracking + budget per user
- Caching layer
- ใช้ 1 codebase ส่งไปได้ทุก provider

ผู้บริหารถาม "ถ้าวันหนึ่งอยากใช้ ChatGPT Enterprise / Claude แทน?" → **เปลี่ยน 1 บรรทัด config**

---

## 🥈 Option B — TypeScript Fullstack (Vercel AI SDK)

### Why
- **Frontend + Backend ภาษาเดียว** — ทีมเล็กทำงานเร็ว
- **Vercel AI SDK** — DX ดีที่สุดในวงการสำหรับ chat UI
- **Type-safe end-to-end** — tRPC / Zod
- เหมาะถ้าต้องการ **product UX สวย** (ผู้บริหารชอบดู demo)

### Stack เต็ม

```yaml
# Backend
Runtime:            Node.js 22 หรือ Bun
Framework:          Next.js 15 (App Router) + Server Actions
                    หรือ Hono / NestJS (ถ้า separate API)
Validation:         Zod
ORM:                Drizzle ORM ⭐ หรือ Prisma
Auth:               Auth.js (NextAuth v5) หรือ Clerk

# AI
LLM SDK:            Vercel AI SDK ⭐  (streaming UI สบาย)
Agent:              LangGraph.js (ถ้าต้องการ stateful agent)
                    หรือ Mastra ⭐ (TS-native agent framework)
Vector:             pgvector + Drizzle
Embedding:          @ai-sdk/openai หรือ ollama-ai-provider

# Frontend
React:              19 + Server Components
UI:                 shadcn/ui + Tailwind
Chat:               assistant-ui หรือ Vercel AI SDK useChat
State:              Zustand + TanStack Query

# Infrastructure
Same as Option A   (Postgres + Redis + LGTM)
```

### ข้อด้อย
- Async / streaming complex กว่า Python
- AI library ecosystem เล็กกว่า Python ~3x
- บาง LLM feature (function calling rare format) มาช้ากว่า

### เหมาะเมื่อ
- ทีม fullstack เล็ก 2-4 คน
- โฟกัส UX > AI capability
- มี Next.js / Vercel infra อยู่แล้ว

---

## 🥉 Option C — Hybrid: Go API + Python AI Service

### Why
- เก็บ infrastructure เดิมที่ทำมาแล้ว (auth, audit, session)
- AI logic แยกออกเป็น microservice Python
- ได้ทั้ง performance ของ Go + ecosystem ของ Python

### Architecture

```
┌─────────────┐         ┌──────────────────┐         ┌────────────┐
│  Frontend   │ ──HTTP──│  Go API Gateway  │──gRPC──>│ Python AI  │
│ (Next.js)   │         │  (Fiber)         │         │ Service    │
└─────────────┘         │                  │         │ (FastAPI)  │
                        │ • Auth (JWT/AD)  │         │            │
                        │ • Rate limit     │         │ • LangGraph│
                        │ • Audit          │         │ • RAG      │
                        │ • Session        │         │ • Tools    │
                        └──────────────────┘         └────────────┘
                                  ↓                          ↓
                          ┌──────────────┐         ┌────────────┐
                          │  PostgreSQL  │         │  vLLM /    │
                          │  Redis       │         │  Ollama    │
                          └──────────────┘         └────────────┘
```

### Stack
- **Go side:** ใช้ของเดิมจาก PoC (Fiber + pgx + JWT)
- **Python side:** FastAPI + LangGraph + LiteLLM
- **Communication:** gRPC (`buf` + `connectrpc`) หรือ REST
- **Shared types:** Protobuf → generate Go + Python

### ข้อด้อย
- 2 ภาษา = 2 ทีม / context switching
- Latency เพิ่ม (network hop)
- Deployment complex ขึ้น

### เหมาะเมื่อ
- ทีมมีทั้ง Go + Python skill
- ต้องการแยก concern ชัด (security boundary)
- Performance critical ที่ edge (Go), AI heavy lifting (Python)

---

## 🔍 Detailed Component Comparison

### LLM Serving

| Tool | License | Throughput | Use case |
|---|---|---|---|
| **Ollama** | MIT | Low | Dev / desktop |
| **vLLM** | Apache 2.0 | **Highest** ⭐ | GPU production |
| **TGI (HuggingFace)** | Apache 2.0 | High | HuggingFace ecosystem |
| **TensorRT-LLM** | Apache 2.0 | Highest (NVIDIA) | NVIDIA-only optimized |
| **SGLang** | Apache 2.0 | Highest (new) | Programmable inference |
| **llama.cpp / llama-server** | MIT | Med (CPU good) | Edge / no GPU |

**สำหรับ NVIDIA Blackwell:** vLLM หรือ TensorRT-LLM (NVIDIA optimize)

### Vector Database

| Tool | Type | Pros | Cons |
|---|---|---|---|
| **pgvector** ⭐ | Postgres ext | ไม่ต้องเพิ่ม service | Scale จำกัด ~10M vectors |
| **Qdrant** ⭐ | Standalone | Rust, super fast, payload filter | Operate เพิ่ม |
| **Weaviate** | Standalone | GraphQL, multimodal | Memory heavy |
| **Milvus** | Standalone | Scale ใหญ่สุด | Complex (3 services) |
| **Pinecone** | Managed cloud | Zero ops | Vendor lock-in, cost |
| **Chroma** | Embedded | DX ดี | ไม่เหมาะ prod scale |
| **Elasticsearch** | Search | มี hybrid พร้อม | Vector ไม่เร็วเท่า Qdrant |

**Recommendation by scale:**
- < 10M vectors: **pgvector** (เหมือนเดิม)
- 10M - 100M: **Qdrant**
- > 100M: **Milvus** หรือ Pinecone

### Agent Framework

| Framework | Language | Maturity | Best for |
|---|---|---|---|
| **LangGraph** ⭐ | Python/TS | Mature | Stateful agent, production |
| **LlamaIndex** | Python | Mature | RAG-heavy, document QA |
| **Pydantic AI** | Python | New, growing fast | Type-safe, simple agents |
| **Semantic Kernel** | C# / Python | Mature (Microsoft) | .NET shop |
| **AutoGen** | Python | Mature (Microsoft) | Multi-agent conversation |
| **CrewAI** | Python | Trendy | Role-based teams |
| **Mastra** | TypeScript | New | TS-first agent |
| **Eino** | Go | New (Bytedance) | Go shop, production-grade |
| **DSPy** | Python | Research | Prompt optimization |

**Recommendation:**
- General purpose enterprise: **LangGraph**
- Type-safety obsessed: **Pydantic AI**
- Microsoft ecosystem: **Semantic Kernel**
- ถ้าเก็บ Go: **Eino** (ใหม่มาก แต่ Bytedance ใช้ prod จริง)

### LLM Observability / Tracing

| Tool | Self-host | License | Features |
|---|---|---|---|
| **Langfuse** ⭐ | ✅ | MIT | Trace, eval, prompt mgmt, cost tracking |
| **LangSmith** | ❌ (cloud only) | Commercial | Best UX, LangChain native |
| **Arize Phoenix** | ✅ | Elastic | Open source, evals |
| **Helicone** | ✅ | Apache 2.0 | Proxy-based, simple |
| **Traceloop** | ✅ | Apache 2.0 | OpenTelemetry-native |

**Recommendation:** **Langfuse** (self-host) — มี prompt management + eval ในตัว

### Auth / Identity

| Tool | Self-host | Use case |
|---|---|---|
| **Keycloak** ⭐ | ✅ | Enterprise standard, OIDC + SAML, AD federation |
| **Authentik** | ✅ | Modern UX, Python-based |
| **Auth0** | ❌ | SaaS, ง่าย, แพง |
| **Clerk** | ❌ | Modern, dev-friendly |
| **Ory Kratos + Hydra** | ✅ | Cloud-native, Go-based |

**Recommendation:** **Keycloak** — มาตรฐาน enterprise, federate AD ได้, OIDC standard

### CI/CD

| Tool | Use case |
|---|---|
| **GitHub Actions** ⭐ | ใช้ GitHub อยู่แล้ว |
| **GitLab CI** | ใช้ GitLab self-host |
| **Argo CD** | GitOps for K8s |
| **Tekton** | K8s-native |

### Container / K8s

```
Dev:        Docker Compose
Staging:    Docker Swarm หรือ K3s (lightweight K8s)
Prod:       Kubernetes (k3s / k8s / OpenShift)
Helm:       chart packaging
Operator:   for stateful (Postgres operator: CloudNativePG)
```

---

## 📐 Reference Architecture (Production-grade)

```
                            ┌──────────────────┐
                            │  Cloudflare /    │
                            │  Internal CDN    │
                            └────────┬─────────┘
                                     │
                   ┌─────────────────┴─────────────────┐
                   │                                   │
            ┌──────▼──────┐                    ┌──────▼──────┐
            │ Web (Next)  │                    │ Admin UI    │
            └──────┬──────┘                    └──────┬──────┘
                   │                                   │
                   └───────────────┬───────────────────┘
                                   │
                          ┌────────▼─────────┐
                          │  API Gateway     │
                          │  (Kong / Traefik)│
                          │  - Rate limit    │
                          │  - WAF           │
                          │  - mTLS          │
                          └────────┬─────────┘
                                   │
                  ┌────────────────┼────────────────┐
                  │                │                │
          ┌───────▼──────┐ ┌──────▼──────┐ ┌──────▼──────┐
          │ Auth Service │ │  AI Service │ │ Admin Service│
          │ (Keycloak)   │ │  (FastAPI + │ │  (FastAPI)   │
          │              │ │   LangGraph)│ │              │
          └──────────────┘ └──────┬──────┘ └──────────────┘
                                   │
              ┌────────────────────┼────────────────────┐
              │                    │                    │
       ┌──────▼──────┐    ┌───────▼─────┐    ┌────────▼────────┐
       │ LiteLLM     │    │  Worker     │    │  Tool Services  │
       │ Gateway     │    │  (Celery)   │    │  - Web Search   │
       │             │    │  - Ingest   │    │  - Analytics    │
       │             │    │  - OCR      │    │  - Sales DB     │
       └──────┬──────┘    └─────────────┘    └─────────────────┘
              │
       ┌──────┴────────────────────┐
       │                            │
  ┌────▼─────┐              ┌──────▼──────┐
  │ Ollama   │              │ vLLM        │
  │ (PoC/dev)│              │ (Production)│
  └──────────┘              └─────────────┘

  ─────────────────── Data layer ───────────────────
  ┌──────────────┐  ┌──────────┐  ┌──────────┐  ┌─────────┐
  │ PostgreSQL   │  │ Qdrant   │  │  Redis   │  │  MinIO  │
  │ - app data   │  │ - vector │  │ - cache  │  │ - files │
  │ - audit      │  │          │  │ - queue  │  │         │
  └──────────────┘  └──────────┘  └──────────┘  └─────────┘

  ──────────── Observability ────────────
  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
  │  Loki    │  │ Mimir    │  │  Tempo   │  │ Grafana  │
  │ (logs)   │  │ (metric) │  │ (trace)  │  │ (UI)     │
  └──────────┘  └──────────┘  └──────────┘  └──────────┘
                       ↑                          ↑
                       └────── OTel Collector ────┘
                                   ↑
                              All services emit OTel

  ──────────── AI Observability ────────────
  ┌──────────────────┐
  │   Langfuse       │ ← LLM traces, prompts, eval, cost
  │   (self-host)    │
  └──────────────────┘
```

---

## 🚦 Migration Roadmap (ถ้าตัดสินใจ rebuild)

### Phase 0 — Decision (1 สัปดาห์)
- [ ] นำเสนอ 3 options + recommendation ผู้บริหาร
- [ ] เลือก stack
- [ ] กำหนด team + budget
- [ ] Setup Linear / Jira project

### Phase 1 — Foundation (2 สัปดาห์)
- [ ] Repo skeleton: monorepo (Turborepo / Nx) หรือ poly-repo
- [ ] CI/CD pipeline พื้นฐาน (lint + test + build)
- [ ] Local dev environment (devcontainer / docker-compose)
- [ ] Observability stack (LGTM) deploy first
- [ ] Keycloak setup + AD federation

### Phase 2 — Core API (3 สัปดาห์)
- [ ] FastAPI skeleton + Pydantic Settings
- [ ] LiteLLM gateway integrate
- [ ] Auth middleware (OIDC verify)
- [ ] Postgres schema + Alembic migration
- [ ] Health endpoints + metrics

### Phase 3 — Agent Layer (3 สัปดาห์)
- [ ] LangGraph orchestrator graph
- [ ] Tools: RAG, Web Search, Analytics, Sales DB
- [ ] Prompt manager (Langfuse)
- [ ] Streaming support

### Phase 4 — Workers & Pipelines (2 สัปดาห์)
- [ ] Celery + Redis setup
- [ ] Document ingest pipeline (parse → chunk → embed → store)
- [ ] OCR pipeline
- [ ] Background eval runs

### Phase 5 — Frontend (3 สัปดาห์)
- [ ] Next.js 15 setup + shadcn/ui
- [ ] Login flow (NextAuth + Keycloak)
- [ ] Chat UI (Vercel AI SDK / assistant-ui)
- [ ] Document management UI
- [ ] Admin dashboard

### Phase 6 — Production hardening (2 สัปดาห์)
- [ ] Security review + pentest
- [ ] Load test (Locust / k6)
- [ ] Chaos testing (Litmus / Toxiproxy)
- [ ] Backup + DR plan
- [ ] Runbook documentation

**Total: ~16 สัปดาห์ (4 เดือน) สำหรับทีม 3-4 คน**

---

## 💰 Cost & Resource Comparison

### Open source self-host (ทั้งหมดที่แนะนำ)

| Component | Monthly cost (เครื่องเอง) | Note |
|---|---|---|
| Postgres + pgvector | $0 | container |
| Redis | $0 | container |
| Qdrant | $0 | container |
| MinIO | $0 | container |
| Keycloak | $0 | container |
| Langfuse | $0 | container |
| Grafana LGTM | $0 | container |
| LiteLLM | $0 | container |
| **Total infra software** | **$0** | |
| Hardware | varies | 1 server 32GB RAM พอเริ่ม |
| Web search API (Tavily) | $5-50 | per 10k queries |

### Managed alternative (ถ้าไม่อยาก ops)

| Component | Cloud option | Cost |
|---|---|---|
| Postgres | Supabase / Neon | $25-100/mo |
| Vector | Qdrant Cloud / Pinecone | $25-100/mo |
| Redis | Upstash | $0-50/mo |
| Auth | Auth0 / Clerk | $25-150/mo |
| LLM observability | LangSmith Cloud | $39-150/mo |
| Object store | S3 / R2 | $5-50/mo |

---

## 🎓 Talent / Hiring perspective

### หา engineer ง่าย/ยาก (Thailand 2026)

| Stack | Engineer count | เงินเดือน expected |
|---|---|---|
| Python + FastAPI + AI | **มาก** ⭐⭐⭐ | 60-150k |
| TypeScript + Next.js | **มาก** ⭐⭐⭐ | 60-130k |
| Go + ML | น้อยมาก ⭐ | 80-180k |
| Java + Spring + AI | ปานกลาง ⭐⭐ | 70-150k |

### ที่บริษัทใหญ่หา resume

> "Python, FastAPI, LangChain/LangGraph, RAG, vector database, vLLM, AWS Bedrock"

ถ้าเขียน "Go + Fiber + custom RAG" → resume ตกง่าย เพราะ recruiter scan keyword

---

## 📚 Learning Resources

### หนังสือ / Course (ภาษาอังกฤษ standard)
- **"Building LLM Powered Applications"** — Manning
- **"Designing Machine Learning Systems"** — Chip Huyen ⭐
- **"AI Engineering"** — Chip Huyen (2025)
- **DeepLearning.AI** — Short courses (LangGraph, RAG, Agents)
- **Anthropic Cookbook** — github.com/anthropics/anthropic-cookbook

### Reference codebase ที่อ่านได้
- **OpenWebUI** — github.com/open-webui/open-webui (Python + Svelte)
- **Khoj** — github.com/khoj-ai/khoj (Python + Next.js)
- **Dify** — github.com/langgenius/dify (LLMOps platform, Python + Next.js)
- **Continue** — github.com/continuedev/continue (TypeScript)
- **Quivr** — github.com/QuivrHQ/quivr (Python + Next.js)

ดูโครงสร้าง repo เหล่านี้ก่อนเขียนเอง — จะเห็น pattern ที่ industry ใช้จริง

---

## 🎯 Recommendation สำหรับนำเสนอผู้บริหาร

### Slide 1 — Problem
"PoC พิสูจน์แล้วว่าทำได้ แต่ stack ปัจจุบัน (Go) **ไม่ใช่มาตรฐาน enterprise AI** หา engineer ยาก, AI library ตามหลัง Python"

### Slide 2 — Industry Standard
"บริษัทระดับ Fortune 500 และ AI native ใช้: **Python (FastAPI) + LangGraph + vLLM + Postgres + Redis + Keycloak + Grafana + Langfuse** — เป็น stack ที่ Microsoft, Anthropic, Databricks ใช้จริง"

### Slide 3 — Recommendation
**"Option A: Rebuild ด้วย Python + LangGraph"**
- Time: 4 เดือน, ทีม 3-4 คน
- Cost: hardware $0 (self-host all open source) + Tavily $50/mo
- Outcome: stack สากล, scale ได้, talent หาง่าย

### Slide 4 — Why not stay with Go
- Talent pool 1/10 ของ Python
- vLLM ecosystem = Python first-class
- Agent framework production-grade ใน Python (LangGraph) มี maturity > Go (Eino) 2 ปี
- เก็บ Go ไว้ทำ infrastructure / data pipeline ได้ ไม่ใช่ AI logic

### Slide 5 — Migration plan
"PoC ปัจจุบันเก็บไว้ใช้งานต่อ 3 เดือน (เป็น production จริง)
→ ทีมใหม่เริ่ม greenfield คู่ขนาน
→ Cutover เมื่อ feature parity"

### Slide 6 — Risk
- ⚠️ Time-to-market ช้าลง 4 เดือน
- ⚠️ ต้องจ้าง / train ทีม Python AI
- ✅ แต่ได้ stack ที่ scale ถึง 10x และ pitch กับ vendor / partner ได้

### Slide 7 — Decision points
1. Go all-in Python rebuild?
2. Stay with Go + ขยาย eino framework?
3. Hybrid Go + Python AI service?

---

## 🤔 ความเห็นส่วนตัวของผม (Engineer perspective)

ถ้า**ผมเป็นคนเลือก:**

**สำหรับ scale 10-15 user, 10 docs (Phase 1):**
- เก็บ Go ไว้ก็ได้ — overhead rebuild ไม่คุ้ม
- เน้น production hardening (sprint 1-3 ใน `08-PRODUCTION-PLAN.md`)

**สำหรับ scale 100+ user, multi-department, multi-use-case:**
- **Rebuild ด้วย Python + LangGraph** ⭐
- Reason: agent capability + ecosystem maturity
- Go ทำได้แต่ต้องเขียนเองเยอะกว่า 5x

**Sweet spot ที่ผมจะเลือกถ้าเริ่มใหม่ในปี 2026:**

```
Frontend:    Next.js 15 + shadcn/ui + Vercel AI SDK
Backend:     Python 3.12 + FastAPI + LangGraph
LLM:         vLLM + LiteLLM gateway
Vector:      pgvector → Qdrant
Auth:        Keycloak (OIDC + AD federation)
Observability: Grafana LGTM + Langfuse
Infra:       Docker → K3s
CI:          GitHub Actions
```

**Tools ที่ขาดไม่ได้:**
- `uv` — Python package manager (faster than poetry)
- `ruff` — linter + formatter
- `mypy` / `pyright` — type check
- `pre-commit` — git hooks
- `pytest` + `pytest-asyncio`

---

## ✅ Action items ถ้าตัดสินใจ rebuild

1. **POC Python skeleton** (1 สัปดาห์) — ทำ minimum FastAPI + LangGraph + Ollama เพื่อ feel
2. **Side-by-side comparison** — ทำ feature เดียวกันทั้ง Go เดิม + Python ใหม่ → ดู effort
3. **Demo ผู้บริหาร** — show LangGraph studio, Langfuse dashboard
4. **Hiring plan** — JD Python AI engineer 2-3 ตำแหน่ง
5. **Decision deadline** — ตั้งวันชัดเจน

---

## 🔗 Related docs

- `07-CODE-REVIEW.md` — gap analysis ของ PoC ปัจจุบัน
- `08-PRODUCTION-PLAN.md` — sprint plan สำหรับ evolve PoC
- `09-ARCHITECTURE-EVOLUTION.md` — เก็บ Go แล้ว evolve
- เอกสารนี้ (`10-GREENFIELD-STACK.md`) — เริ่มใหม่จาก 0 ด้วย industry standard
