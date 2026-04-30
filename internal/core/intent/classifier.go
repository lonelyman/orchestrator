package intent

import (
	"strings"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

// Classifier ตัดสินใจว่าจะใช้ Intent ไหน
type Classifier struct{}

// New สร้าง Classifier ใหม่
func New() *Classifier {
	return &Classifier{}
}

// ragKeywords คำที่บ่งบอกว่าต้องการค้นหาจากเอกสาร
var ragKeywords = []string{
	"นโยบาย", "ประกาศ", "เอกสาร", "ระเบียบ", "ข้อบังคับ",
	"แนวทาง", "วิธี", "ขั้นตอน", "กฎ", "สิทธิ์", "สวัสดิการ",
	"OT", "ล่วงเวลา", "ลา", "หยุด", "สัญญา",
}

// mcpKeywords คำที่บ่งบอกว่าต้องการข้อมูล Real-time จาก DB
var mcpKeywords = []string{
	"ยอดขาย", "สต็อก", "สินค้า", "คงเหลือ", "รายงาน",
	"ข้อมูล", "ตัวเลข", "จำนวน", "รายการ", "บัญชี",
	"พนักงาน", "เงินเดือน", "โบนัส", "ค่าจ้าง",
}

// strongWebSearchKeywords คำที่บ่งบอกชัดว่าต้องค้นเว็บ
var strongWebSearchKeywords = []string{
	"ข่าว",
	"อินเทอร์เน็ต", "internet", "เว็บ", "website", "web",
	"ค้นหา", "search", "google",
}

// recencyWebSearchKeywords คำที่บ่งบอกข้อมูลสด แต่ต้องให้ internal DB/RAG keywords ชนะก่อน
var recencyWebSearchKeywords = []string{
	"ล่าสุด", "วันนี้", "ตอนนี้", "ปัจจุบัน",
}

// Classify วิเคราะห์คำถามและตัดสินใจ Intent
func (c *Classifier) Classify(query string) models.IntentResult {
	query = strings.ToLower(query)

	// เช็คคำที่บอกชัดว่าต้องค้นเว็บก่อน
	for _, kw := range strongWebSearchKeywords {
		if strings.Contains(query, strings.ToLower(kw)) {
			return models.IntentResult{
				Intent:     models.IntentWebSearch,
				Confidence: 0.8,
				Reason:     "matched keyword: " + kw,
			}
		}
	}

	// เช็ค MCP keywords ก่อน recency เพื่อให้คำถาม internal เช่น "ยอดขายวันนี้" ยังเข้าฐานข้อมูล
	for _, kw := range mcpKeywords {
		if strings.Contains(query, strings.ToLower(kw)) {
			return models.IntentResult{
				Intent:     models.IntentMCP,
				Confidence: 0.8,
				Reason:     "matched keyword: " + kw,
			}
		}
	}

	// เช็ค RAG keywords
	for _, kw := range ragKeywords {
		if strings.Contains(query, strings.ToLower(kw)) {
			return models.IntentResult{
				Intent:     models.IntentRAG,
				Confidence: 0.8,
				Reason:     "matched keyword: " + kw,
			}
		}
	}

	// เช็ค recency keywords หลัง internal keywords
	for _, kw := range recencyWebSearchKeywords {
		if strings.Contains(query, strings.ToLower(kw)) {
			return models.IntentResult{
				Intent:     models.IntentWebSearch,
				Confidence: 0.8,
				Reason:     "matched keyword: " + kw,
			}
		}
	}

	// Default → Direct
	return models.IntentResult{
		Intent:     models.IntentDirect,
		Confidence: 0.6,
		Reason:     "no keywords matched",
	}
}
