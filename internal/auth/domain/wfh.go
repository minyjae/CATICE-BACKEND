package domain

// WFHRequest = คำขอ work-from-home 1 วัน ของ user คนหนึ่ง
//   - Date       : วันเดียว (ขอทีละวัน — ต้องการหลายวันก็ยื่นหลายใบ)
//   - ApproverID : snapshot ค่า User.ManagerID ของผู้ยื่น ณ ตอนสร้างคำขอ
type WFHRequest struct {
	ID         string        `json:"id"`
	UserID     string        `json:"user_id"`
	Date       string        `json:"date"`
	Reason     string        `json:"reason"`
	Status     RequestStatus `json:"status"`
	ApproverID string        `json:"approver_id,omitempty"`
	CreatedAt  int64         `json:"created_at"`
	DecidedAt  int64         `json:"decided_at,omitempty"`
}
