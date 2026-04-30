package models

// Intent คือประเภทของคำถาม
type Intent string

const (
	// IntentRAG ใช้เมื่อต้องการค้นหาจากเอกสาร
	IntentRAG Intent = "rag"

	// IntentMCP ใช้เมื่อต้องการดึงข้อมูลจาก Database
	IntentMCP Intent = "mcp"

	// IntentWebSearch ใช้เมื่อต้องการข้อมูลปัจจุบันจากเว็บ
	IntentWebSearch Intent = "web_search"

	// IntentDirect ใช้เมื่อตอบตรงๆ ได้เลย
	IntentDirect Intent = "direct"
)

// IntentResult คือผลลัพธ์จาก Intent Classifier
type IntentResult struct {
	Intent     Intent  `json:"intent"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}
