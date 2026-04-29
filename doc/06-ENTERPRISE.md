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
- [x] GET /v1/rag/documents
- [x] DELETE /v1/rag/documents/:source
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

### Document Access Control (Deferred)

สถานะปัจจุบัน:
- ระบบ RAG ยังไม่มี `access_level`, owner, department, document group, version, หรือ expiry ใน schema
- Search ยังไม่ได้ filter เอกสารตาม role/department ของ user
- ยังไม่ควรเดานโยบายสิทธิ์เองในโค้ด เพราะอาจทำให้พนักงานเห็นเอกสารผิดกลุ่มหรือบล็อกเอกสารที่ควรเห็น

Policy ที่ต้องตัดสินใจก่อนลงโค้ด:
- Role ไหนเห็นเอกสารระดับ `Public`, `Internal`, `Confidential` ได้บ้าง
- ใช้ role อย่างเดียวพอไหม หรือต้องใช้ department/site/company ด้วย
- เอกสาร HR, ผู้บริหาร, การเงิน, สินค้า, และประกาศทั่วไปแยกสิทธิ์อย่างไร
- ต้องมี document owner/approver หรือ expiry/review date หรือไม่
- ถ้าเอกสารไม่มี metadata ให้ default เป็น allow หรือ deny

แนวทาง technical เมื่อ policy พร้อม:
- เพิ่ม metadata ใน `documents` table เช่น `access_level`, `department`, `owner`, `version`, `expires_at`
- ตอน ingest ต้องรับ metadata และ validate ค่า
- ตอน search ต้อง filter ก่อนส่ง context เข้า LLM
- เพิ่ม audit field สำหรับ sources ที่ถูกใช้ตอบ

### LDAP / Active Directory TLS (Deferred)

สถานะปัจจุบัน:
- ตอนนี้ระบบใช้งานผ่าน `DEV_MODE` เป็นหลัก
- ยังไม่สามารถเข้าถึง AD server จริงเพื่อทดสอบ authentication path ได้
- จึงยังไม่แก้โค้ด LDAP/TLS ในรอบนี้ เพื่อลดความเสี่ยงจากการเปลี่ยน behavior โดยไม่มี environment ยืนยันผล

Production risk ที่ต้องกลับมาแก้ก่อนใช้ AD จริง:
- LDAP connection ต้องใช้ TLS ที่ verify certificate ได้จริง
- ห้ามใช้ `InsecureSkipVerify` ใน production
- ถ้า `StartTLS` หรือ LDAPS fail ต้อง fail closed ไม่ควร bind ต่อแบบไม่ปลอดภัย
- ควรแยก config สำหรับ dev/test/prod เช่น `AD_USE_TLS`, `AD_TLS_INSECURE_SKIP_VERIFY`, และ `AD_CA_CERT_PATH`

เงื่อนไขก่อนเริ่มแก้:
- มี AD/LDAP test environment หรือช่องทางเข้าถึง server จริง
- รู้ port/protocol ที่องค์กรใช้จริง: LDAP+StartTLS หรือ LDAPS
- มี CA certificate หรือ trust chain ที่ server ใช้
- มี test account สำหรับตรวจ login success/failure

---

## Audit Log (TODO — ตลาดหลักทรัพย์ต้องมี)

```
บันทึกทุก Query:
- request_id
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

หมายเหตุเรื่อง log:
- Application log (`slog`/stdout) ไม่ควรเก็บ prompt, document text, หรือ response เต็ม
- Application log ควรเก็บเฉพาะ metadata เช่น user/session id, intent, latency, status, text length
- Audit log เป็น compliance store แยกต่างหาก ต้องมี RBAC, retention policy, และการจำกัดสิทธิ์เข้าถึง

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
