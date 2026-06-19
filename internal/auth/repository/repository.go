// Package repository = "ที่เก็บข้อมูล" (data access ล้วน ๆ ไม่มี business logic)
// แยก interface (สัญญา) ออกจาก implementation (GORM) → service/handler พึ่งแค่ interface
// ไม่ผูกกับ DB ตัวจริง → สลับที่เก็บ/ทดสอบด้วย fake ได้
package repository

import "github/minyjae/catice/internal/auth/domain"

// UsersRepository = สัญญาของที่เก็บ user (impl อยู่ที่ auth.go)
type UsersRepository interface {
	Create(u domain.User) error               // คืน domain.ErrEmailTaken ถ้าซ้ำ
	ByEmail(email string) (domain.User, bool) // หาด้วย email (ใช้ตอน login)
	ByID(id string) (domain.User, bool)       // หาด้วย id (JWT → user)
	All() []domain.User                       // user ทั้งหมด (selector มอบหมาย task)
}

// TaskRepository = สัญญาของที่เก็บ task (impl อยู่ที่ task.go)
type TaskRepository interface {
	Create(t domain.Task) error         // insert ใหม่
	Update(t domain.Task) error         // เซฟทับทั้งใบ (move/update อ่านของเดิมมาก่อนแล้วแก้)
	Delete(id string) error             //
	ByID(id string) (domain.Task, bool) // อ่าน task เดิมมาก่อนแก้ (move/update)
	All() []domain.Task
}
