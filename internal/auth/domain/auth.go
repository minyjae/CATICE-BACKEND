// Package domain เก็บ "แก่น" ของ auth: ตัวตนผู้ใช้ (User), ตำแหน่ง (Role), task และ error กลาง
// ไม่พึ่ง HTTP/DB/ORM ใด ๆ → เป็นชั้นในสุดที่ layer อื่นชี้เข้าหา (dependency ไหลเข้าหา domain)
package domain

import "errors"

// Role คือตำแหน่งของผู้ใช้ (เลือกตอน register)
type Role string

const (
	RoleDeveloper Role = "developer"
	RoleHR        Role = "hr"
	RolePM        Role = "pm"
	RolePO        Role = "po"
	RoleCTO       Role = "cto"
	RoleUXUI      Role = "uxui"
)

// Valid เช็คว่า role ที่ส่งมาเป็นค่าที่รองรับไหม (กันส่งตำแหน่งมั่ว)
func (r Role) Valid() bool {
	switch r {
	case RoleDeveloper, RoleHR, RolePM, RolePO, RoleCTO, RoleUXUI:
		return true
	}
	return false
}

// User คือบัญชีผู้ใช้ (domain model)
//   - ID        : รหัสคงที่ (ใช้เป็น client.id ในเกม — ไม่เปลี่ยนแม้ refresh)
//   - Email     : ใช้ login
//   - Role      : ตำแหน่ง
//   - PassHash  : รหัสผ่านที่ hash แล้ว (json:"-" → ไม่หลุดออก API เด็ดขาด)
//   - ManagerID : id ของหัวหน้าที่อนุมัติ leave/WFH ของ user นี้ (ว่างได้ถ้าไม่มีหัวหน้า — fallback ไป HR)
//   - Salary    : *float64 (pointer แยก null กับ 0 ออกจากกัน — HR-sensitive)
type User struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Role      Role     `json:"role"`
	PassHash  string   `json:"-"`
	ManagerID string   `json:"manager_id,omitempty"`
	FirstName string   `json:"first_name,omitempty"`
	LastName  string   `json:"last_name,omitempty"`
	Phone     string   `json:"phone,omitempty"`
	BirthDate string   `json:"birth_date,omitempty"`
	Address   string   `json:"address,omitempty"`
	Salary    *float64 `json:"salary,omitempty"`
	StartDate string   `json:"start_date,omitempty"`
}

// errors ที่ register/login อาจคืน — อยู่ใน domain เพื่อให้ทุก layer (repository/service/handler)
// อ้างถึงตัวเดียวกันผ่าน errors.Is ได้ (เช่น gorm repo แปลง dup key → ErrEmailTaken)
var (
	ErrMissingFields  = errors.New("ต้องกรอก email และ password")
	ErrInvalidRole    = errors.New("role ไม่ถูกต้อง")
	ErrEmailTaken     = errors.New("email นี้ถูกใช้แล้ว")
	ErrBadCredentials = errors.New("email หรือ password ไม่ถูกต้อง")
	ErrForbidden      = errors.New("ไม่มีสิทธิ์ทำรายการนี้")
	ErrUserNotFound   = errors.New("ไม่พบ user")
)
