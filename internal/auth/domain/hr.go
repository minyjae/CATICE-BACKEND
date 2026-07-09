package domain

import "errors"

// RequestStatus = สถานะของคำขอที่ต้องอนุมัติ (leave/WFH ใช้ร่วมกัน)
type RequestStatus string

const (
	StatusPending  RequestStatus = "pending"
	StatusApproved RequestStatus = "approved"
	StatusRejected RequestStatus = "rejected"
)

// Valid เช็คว่า status ที่ส่งมาเป็นค่าที่รองรับไหม
func (s RequestStatus) Valid() bool {
	switch s {
	case StatusPending, StatusApproved, StatusRejected:
		return true
	}
	return false
}

// errors ของ leave/WFH request — ให้ทุก layer อ้างถึงตัวเดียวกันผ่าน errors.Is
var (
	ErrInvalidDateRange = errors.New("ช่วงวันที่ไม่ถูกต้อง")
	ErrNoApprover       = errors.New("ไม่พบผู้อนุมัติ (ไม่มี manager และไม่มี HR ในระบบ)")
	ErrNotPending       = errors.New("คำขอนี้ถูกตัดสินใจไปแล้ว")
	ErrNotApprover      = errors.New("คุณไม่มีสิทธิ์อนุมัติคำขอนี้")
	ErrRequestNotFound  = errors.New("ไม่พบคำขอนี้")
)
