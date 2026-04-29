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

// Classify วิเคราะห์คำถามและตัดสินใจ Intent
func (c *Classifier) Classify(query string) models.IntentResult {
	query = strings.ToLower(query)

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

	// เช็ค MCP keywords
	for _, kw := range mcpKeywords {
		if strings.Contains(query, strings.ToLower(kw)) {
			return models.IntentResult{
				Intent:     models.IntentMCP,
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