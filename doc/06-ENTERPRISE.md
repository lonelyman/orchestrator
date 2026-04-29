# 06 — Enterprise Considerations
> **Last Updated:** 2026-04-29
> **บริษัท:** นูทริชั่น โปรเฟส — 1,000+ พนักงาน, ตลาดหลักทรัพย์

---

## Use Cases หลัก

```
กลุ่มที่ 1: คำถามทั่วไปองค์กร
→ นโยบาย, สวัสดิการ, ระเบียบ, ประกาศ HR
→ ใช้ RAG (เอกสาร PDF/Word)

กลุ่มที่ 2: ข้อมูลสินค้าและธุรกิจ
→ สินค้า, stock, ราคา, order, รายงานผู้บริหาร
→ ใช้ MCP (API/Database)
```

---

## RAG Flow เอกสารใหม่ (TODO)

```
1. มีเอกสารใหม่ (PDF/Word/Excel)
2. Upload ผ่าน /v1/rag/upload
3. Parse → chunk → embed → pgvector
4. metadata: ชื่อไฟล์, วันที่, ประเภท, access_level
5. AI พร้อมตอบทันที

TODO:
- [ ] Web UI สำหรับ Admin upload
- [ ] Batch ingest
- [ ] Document versioning
- [ ] Document expiry
- [ ] GET /v1/rag/documents
- [ ] DELETE /v1/rag/documents/:id
- [ ] รองรับ .docx และ .xlsx
```

---

## MCP / API Integration (TODO)

**Phase A — สินค้า (Read-only)**
```
"สินค้า X มี stock เหลือเท่าไหร่?"
→ Tool: get_product_stock
→ MCP → Product DB
```

**Phase B — รายงานผู้บริหาร (Read-only)**
```
"ยอดขายสัปดาห์นี้?"
→ Tool: get_sales_report
→ MCP → ERP/Sales DB
```

**Phase C — Action (Write — ต้อง confirm)**
```
"เพิ่ม order สินค้า X 100 ชิ้น"
→ Tool: create_order (ต้อง user confirm ก่อน)
```

---

## Security & Access Control (TODO)

```
Role ที่ต้องการ:
├── Admin     → อัพโหลดเอกสาร, จัดการระบบ
├── Manager   → ดูรายงาน, ถามข้อมูลธุรกิจ
└── Employee  → ถามนโยบาย HR, สวัสดิการ

Document Access Level:
├── Confidential → ผู้บริหารเท่านั้น
├── Internal     → พนักงานทุกคน
└── Public       → ทุกคน

TODO:
- [ ] role field จาก AD groups
- [ ] access_level ใน documents table
- [ ] Filter documents ตาม role ก่อน search
- [ ] Admin Panel จัดการสิทธิ์
```

---

## Audit Log (TODO — ตลาดหลักทรัพย์ต้องมี)

```
บันทึกทุก Query:
- user_id, username
- session_id
- คำถาม
- intent (rag/mcp/direct)
- เอกสารที่ดึงมา (sources)
- คำตอบ
- เวลา + IP

ทำไม:
✅ Compliance
✅ Security audit
✅ Improve AI quality
✅ Legal evidence

TODO:
- [ ] audit_logs table
- [ ] บันทึกทุก request/response
- [ ] GET /v1/admin/logs
- [ ] Log retention policy
```

---

## Vector DB Scaling Plan

| ช่วง | Chunks | แนวทาง |
|---|---|---|
| ตอนนี้ | < 50,000 | pgvector + IVFFlat Index |
| ระยะกลาง | 50,000-500,000 | Monitor + tune |
| ระยะยาว | > 500,000 | พิจารณา Qdrant |

**คาดการณ์:** เอกสาร ~1,700 ไฟล์ → ~20,000-50,000 chunks → pgvector โอเค

---

## Performance สำหรับ 1,000 Users

```
Concurrent users: ~50-100 คน
Request/minute:   ~200-500

iMac M1 16GB:
→ PoC เท่านั้น ช้าถ้าหลายคนพร้อมกัน

Production Ubuntu + GPU:
→ vLLM รองรับ concurrent ได้ดีกว่ามาก

TODO:
- [ ] Load testing
- [ ] Rate limiting per user
- [ ] Queue system
```

---

## Data Governance

```
ต้องกำหนดก่อน Deploy:
1. เอกสารไหน AI "ห้าม" ตอบ?
2. AI ตอบผิดแล้วทำยังไง? (Feedback)
3. เอกสารหมดอายุแล้วทำยังไง?
4. ใครเป็น AI Owner ในองค์กร?
```
