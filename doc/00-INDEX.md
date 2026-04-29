# Enterprise AI Orchestrator — Index
> **Last Updated:** 2026-04-29
> **GitHub:** https://github.com/lonelyman/orchestrator.git (branch: dev)
> **วิธีใช้:** ทุกครั้งที่เปิด session ใหม่ ให้ paste ไฟล์ที่เกี่ยวข้องให้ Claude ดูก่อน

---

## 📁 ไฟล์ทั้งหมด

| ไฟล์ | หัวข้อ | สถานะ |
|---|---|---|
| `00-INDEX.md` | Index + วิธีใช้ | ✅ |
| `01-OVERVIEW.md` | Project Overview, Hardware, Architecture | ✅ |
| `02-TECH-STACK.md` | Tech Stack, Codebase Structure, API Endpoints | ✅ |
| `03-ROADMAP.md` | Roadmap, Known Issues, TODO | ✅ |
| `04-CONFIG.md` | Configuration, .env, Docker Commands | ✅ |
| `05-SESSION-LOG.md` | Installation Log, Session Logs ทั้งหมด | ✅ |
| `06-ENTERPRISE.md` | Enterprise Considerations (Security, Audit, Scale) | ✅ |
| `07-CODE-REVIEW.md` | Code Review — Production Readiness Gap Analysis | ✅ 2026-04-29 |
| `08-PRODUCTION-PLAN.md` | Production-Grade Roadmap (Sprint Plan) | ✅ 2026-04-29 |
| `09-ARCHITECTURE-EVOLUTION.md` | PoC Ollama → Production vLLM Architecture Plan | ✅ 2026-04-29 |
| `10-GREENFIELD-STACK.md` | Industry-Standard Stack (ถ้าเริ่มใหม่จาก 0) | ✅ 2026-04-29 |

---

## 🚀 สถานะปัจจุบัน

```
✅ Phase 0 — Foundation (Ollama + Docker + Go + WebUI)
✅ Phase 1 — RAG (pgvector + PDF Upload + OCR)
✅ Phase 2 — MCP Structure (SQL Server ready)
✅ Phase 3 — Intent Router (Rule-based)
✅ Streaming + WebUI Integration
✅ Health Check ครบทุก dependency
✅ Structured Logging (slog JSON)
✅ Graceful Shutdown
✅ AD/LDAP Authentication + JWT
✅ Session Management (Sliding Window History)

✅ Audit Log
🔜 Role-based Access Control
🔜 WebUI Login Integration
⏳ Phase 4 — Production (Ubuntu + vLLM + Blackwell)
```

---

## 🔑 Quick Reference

**Login:**
```bash
curl -X POST http://10.19.105.32:50000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"nipon.k","password":"password"}'
```

**Chat:**
```bash
curl -X POST http://10.19.105.32:50000/v1/chat/completions \
  -H "Authorization: Bearer TOKEN" \
  -H "X-Session-ID: SESSION_ID" \
  -H "Content-Type: application/json" \
  -d '{"model":"qwen2.5:7b","messages":[{"role":"user","content":"คำถาม"}]}'
```

**Upload PDF:**
```bash
curl -X POST http://10.19.105.32:50000/v1/rag/upload \
  -H "Authorization: Bearer TOKEN" \
  -F "file=@/path/to/file.pdf"
```

**Docker:**
```bash
docker compose up --build -d postgres orchestrator
docker logs orchestrator-api --tail 20
```
