package repository

import (
	"encoding/json"

	"gorm.io/gorm"

	"github/minyjae/catice/internal/auth/domain"
)

// TaskModel = persistence model ของ task (รูปร่างตาราง tasks ใน Postgres)
// แยกจาก domain.Task โดยตั้งใจ — รายละเอียด ORM/DB ไม่รั่วเข้า domain
//   - AssignTo เก็บเป็น JSON text (คอลัมน์เดียว) → ไม่ต้องมีตาราง join ให้ยุ่ง
type TaskModel struct {
	ID        string `gorm:"primaryKey"`
	BoardID   string `gorm:"index"` // board ที่ task สังกัด — index → query/ลบตามบอร์ดได้เร็ว
	Title     string `gorm:"not null"`
	Detail    string
	Status    string `gorm:"not null"`
	CreatedBy string `gorm:"index"` // index → query "task ของฉัน" ได้เร็ว
	AssignTo  string `gorm:"type:text"`
}

func (TaskModel) TableName() string { return "tasks" }

// taskToDomain : persistence model → domain Task (แตก AssignTo จาก JSON)
func taskToDomain(m TaskModel) domain.Task {
	var assign []string
	if m.AssignTo != "" {
		_ = json.Unmarshal([]byte(m.AssignTo), &assign)
	}
	return domain.Task{
		ID:        m.ID,
		BoardID:   m.BoardID,
		Title:     m.Title,
		Detail:    m.Detail,
		TStatus:   domain.Status(m.Status),
		CreatedBy: m.CreatedBy,
		AssignTo:  assign,
	}
}

// taskFromDomain : domain Task → persistence model (อัด AssignTo เป็น JSON)
func taskFromDomain(t domain.Task) (TaskModel, error) {
	assign := "" // nil/ว่าง → เก็บเป็น string ว่าง (toDomain คืน nil กลับ)
	if len(t.AssignTo) > 0 {
		b, err := json.Marshal(t.AssignTo)
		if err != nil {
			return TaskModel{}, err
		}
		assign = string(b)
	}
	return TaskModel{
		ID:        t.ID,
		BoardID:   t.BoardID,
		Title:     t.Title,
		Detail:    t.Detail,
		Status:    string(t.TStatus),
		CreatedBy: t.CreatedBy,
		AssignTo:  assign,
	}, nil
}

// gormTasks = impl ของ TaskRepository ที่เก็บลง Postgres ผ่าน GORM (ถาวร)
type gormTasks struct {
	db *gorm.DB
}

// NewGormTasks สร้าง repository + run AutoMigrate สร้าง/อัปเดตตาราง tasks
func NewGormTasks(db *gorm.DB) (*gormTasks, error) {
	if err := db.AutoMigrate(&TaskModel{}); err != nil {
		return nil, err
	}
	return &gormTasks{db: db}, nil
}

func (g *gormTasks) Create(t domain.Task) error {
	m, err := taskFromDomain(t)
	if err != nil {
		return err
	}
	return g.db.Create(&m).Error
}

// Update เซฟทับทั้งใบ (service อ่านของเดิมมาก่อนแล้วแก้ field ที่ต้องการ)
func (g *gormTasks) Update(t domain.Task) error {
	m, err := taskFromDomain(t)
	if err != nil {
		return err
	}
	return g.db.Save(&m).Error
}

func (g *gormTasks) Delete(id string) error {
	return g.db.Delete(&TaskModel{}, "id = ?", id).Error
}

// DeleteByBoard ลบ task ทั้งหมดของบอร์ด (cascade ตอนลบ board)
func (g *gormTasks) DeleteByBoard(boardID string) error {
	return g.db.Delete(&TaskModel{}, "board_id = ?", boardID).Error
}

func (g *gormTasks) ByID(id string) (domain.Task, bool) {
	var m TaskModel
	if err := g.db.First(&m, "id = ?", id).Error; err != nil {
		return domain.Task{}, false
	}
	return taskToDomain(m), true
}

// All คืน task ทั้งหมด — เรียงตาม id ให้ลำดับคงที่
func (g *gormTasks) All() []domain.Task {
	var ms []TaskModel
	if err := g.db.Order("id").Find(&ms).Error; err != nil {
		return nil
	}
	out := make([]domain.Task, 0, len(ms))
	for _, m := range ms {
		out = append(out, taskToDomain(m))
	}
	return out
}
