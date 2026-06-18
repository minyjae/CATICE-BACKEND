// Package domain เก็บ "แก่น" ของ auth: ตัวตนผู้ใช้ (User), ตำแหน่ง (Role) และ error กลาง
// ไม่พึ่ง HTTP/DB/ORM ใด ๆ → เป็นชั้นในสุดที่ layer อื่นชี้เข้าหา (dependency ไหลเข้าหา domain)
package domain

import "errors"

// Role คือตำแหน่งของผู้ใช้ (เลือกตอน register)
type Role string

const (
	RoleDeveloper Role = "developer"
	RolePM        Role = "pm"
	RolePO        Role = "po"
	RoleCTO       Role = "cto"
	RoleUXUI      Role = "uxui"
)

// Valid เช็คว่า role ที่ส่งมาเป็นค่าที่รองรับไหม (กันส่งตำแหน่งมั่ว)
func (r Role) Valid() bool {
	switch r {
	case RoleDeveloper, RolePM, RolePO, RoleCTO, RoleUXUI:
		return true
	}
	return false
}

// User คือบัญชีผู้ใช้ (domain model)
//   - ID       : รหัสคงที่ (ใช้เป็น client.id ในเกม — ไม่เปลี่ยนแม้ refresh)
//   - Email    : ใช้ login
//   - Role     : ตำแหน่ง
//   - PassHash : รหัสผ่านที่ hash แล้ว (json:"-" → ไม่หลุดออก API เด็ดขาด)
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Role     Role   `json:"role"`
	PassHash string `json:"-"`
}

// errors ที่ register/login อาจคืน — อยู่ใน domain เพื่อให้ทุก layer (repository/service/controller)
// อ้างถึงตัวเดียวกันผ่าน errors.Is ได้ (เช่น gorm repo แปลง dup key → ErrEmailTaken)
var (
	ErrMissingFields  = errors.New("ต้องกรอก email และ password")
	ErrInvalidRole    = errors.New("role ไม่ถูกต้อง")
	ErrEmailTaken     = errors.New("email นี้ถูกใช้แล้ว")
	ErrBadCredentials = errors.New("email หรือ password ไม่ถูกต้อง")
)
