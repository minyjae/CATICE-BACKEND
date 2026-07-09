package domain

import "errors"

// LeaveType คือประเภทของการลา
type LeaveType string

const (
	LeaveVacation LeaveType = "vacation"
	LeaveSick     LeaveType = "sick"
	LeavePersonal LeaveType = "personal"
)

// Valid เช็คว่า type ที่ส่งมาเป็นค่าที่รองรับไหม
func (t LeaveType) Valid() bool {
	switch t {
	case LeaveVacation, LeaveSick, LeavePersonal:
		return true
	}
	return false
}

// LeaveRequest = คำขอลา 1 ใบ ของ user คนหนึ่ง
//   - UserID     : ผู้ยื่นคำขอ (จาก JWT — server เซ็ตเอง ไม่เชื่อ client)
//   - ApproverID : snapshot ค่า User.ManagerID ของผู้ยื่น ณ ตอนสร้างคำขอ (เปลี่ยน manager ทีหลังไม่กระทบคำขอเก่า)
//   - CreatedAt/DecidedAt : unix seconds — DecidedAt เป็น 0 จนกว่าจะถูกอนุมัติ/ปฏิเสธ
type LeaveRequest struct {
	ID         string        `json:"id"`
	UserID     string        `json:"user_id"`
	Type       LeaveType     `json:"type"`
	StartDate  string        `json:"start_date"`
	EndDate    string        `json:"end_date"`
	Reason     string        `json:"reason"`
	Status     RequestStatus `json:"status"`
	ApproverID string        `json:"approver_id,omitempty"`
	CreatedAt  int64         `json:"created_at"`
	DecidedAt  int64         `json:"decided_at,omitempty"`
}

// errors ของ leave request
var (
	ErrEmptyLeaveType = errors.New("ต้องระบุประเภทการลา")
)
