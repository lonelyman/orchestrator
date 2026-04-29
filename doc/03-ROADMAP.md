# 03 — Roadmap & TODO
> **Last Updated:** 2026-04-29

---

## Roadmap

| Phase | หัวข้อ | สถานะ |
|---|---|---|
| **Phase 0** | Foundation: Ollama + Docker + Go + WebUI | ✅ Done |
| **Phase 1** | RAG: Vector Ingestion + Search + PDF Upload + OCR | ✅ Done |
| **Phase 2** | MCP: SQL Server Bridge Structure | ✅ Structure Ready |
| **Phase 3** | Orchestration: Intent Router | ✅ Done |
| **Phase 3.5** | Session + Auth + Logging + Streaming | ✅ Done |
| **Phase 4** | Production: Ubuntu + vLLM + Blackwell | ⏳ |

---

## TODO — ก่อน Production

### Priority สูง
- [ ] **Audit Log** — บันทึกทุก query/response (ตลาดหลักทรัพย์ต้องมี)
- [ ] **Role-based Access** — Admin vs Manager vs Employee
- [ ] **Rate Limiting** — จำกัด request per user
- [ ] **Config Validation** — Fail-fast เมื่อ .env ผิด

### Priority กลาง
- [ ] **WebUI Login** — เชื่อม Open WebUI กับ Auth
- [ ] **Document Management** — list/delete endpoints
- [ ] **Circuit Breaker** — ถ้า Ollama ล่ม
- [ ] **LLM-based Intent Classifier** — แม่นยำกว่า rule-based

### Priority ต่ำ (Phase 4)
- [ ] **Monitoring** — Prometheus + Grafana (commented ใน docker-compose)
- [ ] **golang-migrate** — แทน InitSchema()
- [ ] **OpenTelemetry** — Distributed Tracing
- [ ] **OCR Worker** — แยกเป็น Background Process
- [ ] **Horizontal Scaling** — Load balancer

---

## Known Issues

| Issue | Status | แนวทางแก้ |
|---|---|---|
| Scanned PDF อ่านไม่ออก | ✅ แก้แล้ว | tesseract OCR (tha+eng) |
| pgvector Search ได้ 0 docs | ✅ แก้แล้ว | fallback query |
| SQL Server ยังไม่เชื่อมต่อจริง | ⏳ รอ | sqlserver.go พร้อมแล้ว |
| WebUI ยังไม่มี Login | 🔜 | เชื่อม Auth ทีหลัง |
| nomic-embed-text-v2-moe context = 512 | ✅ แก้แล้ว | ChunkSize=200 |

---

## Enterprise Considerations

### สำหรับบริษัท 1,000 คน / ตลาดหลักทรัพย์

**Vector DB Scaling:**
```
ตอนนี้:  pgvector (โอเค < 100,000 chunks)
อนาคต:  Qdrant ถ้า > 500,000 chunks
```

**Security:**
```
ต้องมี:
- Document Access Control (ลับ vs ทั่วไป vs สาธารณะ)
- Role: Admin / Manager / Employee
- Audit Log ทุก query
- JWT expiry + refresh token
```

**Performance สำหรับ 1,000 users:**
```
Concurrent ~50-100 คน
iMac M1: PoC เท่านั้น
Production: Ubuntu + NVIDIA GPU
```

**Data Governance:**
```
- เอกสารไหน AI ห้ามตอบ?
- Document versioning + expiry
- AI Owner ในองค์กร
- Feedback mechanism
```
