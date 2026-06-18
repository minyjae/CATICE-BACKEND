// Package repository = "ที่เก็บ user" (data access ล้วน ๆ ไม่มี business logic)
// แยกเป็น interface → สลับ in-memory ↔ Postgres(GORM) ได้ โดย service/controller ไม่ต้องรู้เรื่อง
package repository

import "github/minyjae/catice/internal/auth/domain"

// UsersRepository = สัญญาของที่เก็บ user (service พึ่งแค่ interface นี้ ไม่ผูกกับ DB ตัวจริง)
type UsersRepository interface {
	Create(u domain.User) error               // คืน domain.ErrEmailTaken ถ้าซ้ำ
	ByEmail(email string) (domain.User, bool) // หาด้วย email (ใช้ตอน login)
	ByID(id string) (domain.User, bool)       // หาด้วย id (cookie → user)
	All() []domain.User                       // user ทั้งหมด (selector มอบหมาย task)
}
