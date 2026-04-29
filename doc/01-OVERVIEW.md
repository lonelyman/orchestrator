# 01 — Project Overview
> **Last Updated:** 2026-04-29

---

## Project Overview

สร้าง **Enterprise AI Orchestrator** ด้วยภาษา Go เพื่อเป็นสมองกลางขององค์กร เชื่อมต่อข้อมูล (Multi-DB) และใช้เครื่องมืออัตโนมัติ (MCP) โดยมีความยืดหยุ่นจาก PoC ไปสู่ Production

**บริษัท:** นูทริชั่น โปรเฟส จำกัด (มหาชน) — อาหารเสริม, 1,000+ พนักงาน, ตลาดหลักทรัพย์

---

## Hardware

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

## Architecture

```
MacBook (Dev/Client)
    │ SSH / Postman / curl
    ▼
iMac M1 16GB (PoC Server)
    │
    ├── Ollama (Native M1 Metal)
    │       ├── qwen2.5:7b        ← LLM หลัก (4.7GB)
    │       └── nomic-embed-text-v2-moe ← Embedding (957MB)
    │
    └── Docker Compose
            ├── Go Orchestrator (Port 50000)
            ├── PostgreSQL + pgvector (Port 15433)
            └── Open WebUI (Port 53000)
```

## Request Flow

```
User → Login (AD/LDAP) → JWT Token
    ↓
Go Orchestrator (Port 50000)
    ↓ JWT Middleware → Session Middleware
    ↓
Intent Classifier
    ↓
┌──────────┬──────────┬──────────┐
↓          ↓          ↓
RAG        MCP       Direct
(pgvector) (SQL Svr)  (ตรงๆ)
└──────────┴──────────┴──────────┘
                ↓
    Session History (Sliding Window 10 msgs)
                ↓
         Ollama (qwen2.5:7b)
                ↓
     Stream Response + Save to DB
                ↓
           คำตอบภาษาไทย ✅
```

## Port Map

| Service | Port | หมายเหตุ |
|---|---|---|
| Ollama | 11434 (internal) | Native on M1 |
| Go Orchestrator | 50000 | Backend API |
| PostgreSQL | 15433 (external) / 5432 (internal) | Vector + Session DB |
| Open WebUI | 53000 | Chat Interface |

## API Endpoints

| Method | Endpoint | Auth | หน้าที่ |
|---|---|---|---|
| GET | /live | ❌ | Liveness: process ยังตอบ HTTP |
| GET | /ready | ❌ | Readiness: DB + LLM + Embedder พร้อมรับ traffic |
| GET | /health | ❌ | Backward-compatible readiness |
| POST | /auth/login | ❌ | Login ด้วย AD account |
| GET | /auth/me | ✅ JWT | ดูข้อมูล user |
| GET | /v1/models | ❌ | Model list (OpenAI-compatible) |
| POST | /v1/chat/completions | ✅ JWT | Chat + Stream + Session |
| POST | /v1/rag/ingest | ✅ JWT | Upload text ตรงๆ |
| POST | /v1/rag/upload | ✅ JWT | Upload ไฟล์ (PDF/TXT/MD) |

ทุก response มี `X-Request-ID` สำหรับ trace log และ audit trail
