// Package repository = "ที่เก็บข้อมูล" (data access ล้วน ๆ ไม่มี business logic)
// แยก interface (สัญญา) ออกจาก implementation (GORM) → service/handler พึ่งแค่ interface
// ไม่ผูกกับ DB ตัวจริง → สลับที่เก็บ/ทดสอบด้วย fake ได้
package repository

import "github/minyjae/catice/internal/auth/domain"

// UsersRepository = สัญญาของที่เก็บ user (impl อยู่ที่ auth.go)
type UsersRepository interface {
	Create(u domain.User) error               // คืน domain.ErrEmailTaken ถ้าซ้ำ
	Update(u domain.User) error               // เซฟทับทั้งใบ (เช่น ตั้ง ManagerID)
	ByEmail(email string) (domain.User, bool) // หาด้วย email (ใช้ตอน login)
	ByID(id string) (domain.User, bool)       // หาด้วย id (JWT → user)
	All() []domain.User                       // user ทั้งหมด (selector มอบหมาย task)
}

// TaskRepository = สัญญาของที่เก็บ task (impl อยู่ที่ task.go)
type TaskRepository interface {
	Create(t domain.Task) error         // insert ใหม่
	Update(t domain.Task) error         // เซฟทับทั้งใบ (move/update อ่านของเดิมมาก่อนแล้วแก้)
	Delete(id string) error             //
	DeleteByBoard(boardID string) error // ลบ task ทั้งหมดของบอร์ด (cascade ตอนลบ board)
	ByID(id string) (domain.Task, bool) // อ่าน task เดิมมาก่อนแก้ (move/update)
	All() []domain.Task
}

// BoardRepository = สัญญาของที่เก็บ board (impl อยู่ที่ board.go)
type BoardRepository interface {
	Create(b domain.Board) error
	Update(b domain.Board) error
	Delete(id string) error
	ByID(id string) (domain.Board, bool)
	All() []domain.Board
}

// MessageRepository = สัญญาของที่เก็บข้อความแชต (impl อยู่ที่ message.go)
// ประวัติคืนแบบเรียงเวลาจากเก่า→ใหม่ (asc) เอา N ข้อความล่าสุด
type MessageRepository interface {
	Create(m domain.Message) error
	RoomHistory(room string, limit int) []domain.Message      // chat "ห้องนี้" ของห้องหนึ่ง
	AllHistory(limit int) []domain.Message                    // chat "ทั้งหมด"
	PrivateHistory(userID string, limit int) []domain.Message // DM ที่ user นี้เกี่ยวข้อง (ส่ง/รับ)
}

// HolidayRepository = สัญญาของที่เก็บวันหยุดบริษัท (impl อยู่ที่ holiday.go)
type HolidayRepository interface {
	Create(h domain.Holiday) error
	Delete(id string) error
	All() []domain.Holiday
}

// LeaveRepository = สัญญาของที่เก็บคำขอลา (impl อยู่ที่ leave.go)
type LeaveRepository interface {
	Create(l domain.LeaveRequest) error
	Update(l domain.LeaveRequest) error // เซฟทับทั้งใบ (ตอนอนุมัติ/ปฏิเสธ อ่านของเดิมมาก่อนแล้วแก้)
	ByID(id string) (domain.LeaveRequest, bool)
	ByUser(userID string) []domain.LeaveRequest
	PendingForApprover(approverID string) []domain.LeaveRequest
	// ApprovedByUserTypeYear ใช้นับวันลาที่อนุมัติแล้วของ user ในปีนั้น (เช็ค quota)
	ApprovedByUserTypeYear(userID string, leaveType domain.LeaveType, year int) []domain.LeaveRequest
}

// WFHRepository = สัญญาของที่เก็บคำขอ work-from-home (impl อยู่ที่ wfh.go)
type WFHRepository interface {
	Create(w domain.WFHRequest) error
	Update(w domain.WFHRequest) error
	ByID(id string) (domain.WFHRequest, bool)
	ByUser(userID string) []domain.WFHRequest
	PendingForApprover(approverID string) []domain.WFHRequest
	// CountApprovedByUserInRange ใช้นับ WFH ที่อนุมัติแล้วในช่วงวันที่กำหนด (เช็ค quota รายสัปดาห์/เดือน)
	CountApprovedByUserInRange(userID, startDate, endDate string) int
}

// PolicyRepository = singleton นโยบาย leave/WFH ของบริษัท (impl อยู่ที่ policy.go)
type PolicyRepository interface {
	Get() domain.LeavePolicy         // คืน DefaultPolicy ถ้ายังไม่มีใน DB
	Save(p domain.LeavePolicy) error // upsert (สร้างหรืออัปเดต row เดียว)
}

// DiaryRepository = สัญญาของที่เก็บ daily diary (impl อยู่ที่ diary.go)
type DiaryRepository interface {
	Upsert(d domain.DailyDiary) error // สร้างใหม่ถ้ายังไม่มี (user_id,date) นี้ ถ้ามีแล้วอัปเดต content/updated_at
	ByUserAndDate(userID, date string) (domain.DailyDiary, bool)
	ByUser(userID string, limit int) []domain.DailyDiary
}
