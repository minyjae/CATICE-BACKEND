package repository

import (
	"errors"

	"gorm.io/gorm"

	"github/minyjae/catice/internal/auth/domain"
)

// UserModel = "persistence model" ของ GORM (รูปร่างตาราง users ใน Postgres)
// แยกจาก domain.User โดยตั้งใจ — ORM tag/รายละเอียด DB ไม่รั่วเข้า domain
type UserModel struct {
	ID        string `gorm:"primaryKey"`
	Email     string `gorm:"uniqueIndex;not null"` // unique → email ซ้ำไม่ได้ (DB การันตี)
	Role      string `gorm:"not null"`
	PassHash  string `gorm:"not null"`
	ManagerID string `gorm:"index"` // id ของหัวหน้า (self-reference) — ว่างได้
}

// TableName บังคับชื่อตารางเป็น "users" (ไม่ให้ GORM เดา pluralize เอง)
func (UserModel) TableName() string { return "users" }

// toDomain : persistence model → domain User
func toDomain(m UserModel) domain.User {
	return domain.User{ID: m.ID, Email: m.Email, Role: domain.Role(m.Role), PassHash: m.PassHash, ManagerID: m.ManagerID}
}

// fromDomain : domain User → persistence model
func fromDomain(u domain.User) UserModel {
	return UserModel{ID: u.ID, Email: u.Email, Role: string(u.Role), PassHash: u.PassHash, ManagerID: u.ManagerID}
}

// gormUsers = impl ของ UsersRepository ที่เก็บลง Postgres ผ่าน GORM (ถาวร — รอด restart ผ่าน volume)
// service/handler/ws พึ่งแค่ interface UsersRepository → สลับที่เก็บได้โดยไม่ต้องแก้
type gormUsers struct {
	db *gorm.DB
}

// NewGormUsers สร้าง repository + run AutoMigrate สร้าง/อัปเดตตาราง users ให้ตรงกับ UserModel
func NewGormUsers(db *gorm.DB) (*gormUsers, error) {
	if err := db.AutoMigrate(&UserModel{}); err != nil {
		return nil, err
	}
	return &gormUsers{db: db}, nil
}

// Create insert user ใหม่ — แปลง duplicate key (email ซ้ำ) → domain.ErrEmailTaken ให้ตรง contract
// (อาศัย gorm.Config{TranslateError:true} ใน config.NewGormDB)
func (g *gormUsers) Create(u domain.User) error {
	m := fromDomain(u)
	err := g.db.Create(&m).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return domain.ErrEmailTaken
	}
	return err
}

func (g *gormUsers) ByEmail(email string) (domain.User, bool) {
	var m UserModel
	if err := g.db.Where("email = ?", email).First(&m).Error; err != nil {
		return domain.User{}, false // รวม gorm.ErrRecordNotFound → ไม่เจอ
	}
	return toDomain(m), true
}

// Update เซฟทับทั้งใบ (service อ่านของเดิมมาก่อนแล้วแก้) — ตอนนี้ใช้ตั้ง ManagerID เป็นหลัก
func (g *gormUsers) Update(u domain.User) error {
	m := fromDomain(u)
	return g.db.Save(&m).Error
}

func (g *gormUsers) ByID(id string) (domain.User, bool) {
	var m UserModel
	if err := g.db.First(&m, "id = ?", id).Error; err != nil {
		return domain.User{}, false
	}
	return toDomain(m), true
}

// All คืน user ทั้งหมด (selector มอบหมาย task) — เรียงตาม email ให้ลำดับคงที่
func (g *gormUsers) All() []domain.User {
	var ms []UserModel
	if err := g.db.Order("email").Find(&ms).Error; err != nil {
		return nil
	}
	out := make([]domain.User, 0, len(ms))
	for _, m := range ms {
		out = append(out, toDomain(m))
	}
	return out
}
