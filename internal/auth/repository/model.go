package repository

import "github/minyjae/catice/internal/auth/domain"

// UserModel = "persistence model" ของ GORM (รูปร่างตาราง users ใน Postgres)
// แยกจาก domain.User โดยตั้งใจ — ORM tag/รายละเอียด DB ไม่รั่วเข้า domain
// repository เป็นคนแปลงไป-กลับ (toDomain/fromDomain) → เปลี่ยน ORM/DB ได้โดย domain ไม่รู้เรื่อง
type UserModel struct {
	ID       string `gorm:"primaryKey"`
	Email    string `gorm:"uniqueIndex;not null"` // unique → email ซ้ำไม่ได้ (DB การันตี)
	Role     string `gorm:"not null"`
	PassHash string `gorm:"not null"`
}

// TableName บังคับชื่อตารางเป็น "users" (ไม่ให้ GORM เดา pluralize เอง)
func (UserModel) TableName() string { return "users" }

// toDomain : persistence model → domain User
func toDomain(m UserModel) domain.User {
	return domain.User{ID: m.ID, Email: m.Email, Role: domain.Role(m.Role), PassHash: m.PassHash}
}

// fromDomain : domain User → persistence model
func fromDomain(u domain.User) UserModel {
	return UserModel{ID: u.ID, Email: u.Email, Role: string(u.Role), PassHash: u.PassHash}
}
