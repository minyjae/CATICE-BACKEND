// Package id สร้างรหัสสุ่มไม่ซ้ำ ใช้ร่วมกันได้หลาย package (player id, object id, ...)
package id

import (
	"crypto/rand"
	"encoding/hex"
)

// New สุ่มรหัส hex 16 ตัว (จาก 8 byte) โอกาสซ้ำต่ำมาก
func New() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
