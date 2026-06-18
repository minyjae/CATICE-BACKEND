package repository

import (
	"errors"

	"gorm.io/gorm"

	"github/minyjae/catice/internal/auth/domain"
)

// gormUsers = impl ของ UsersRepository ที่เก็บลง Postgres ผ่าน GORM (ถาวร — รอด restart ผ่าน volume)
// สลับมาจาก memUsers ได้โดยไม่แตะ service/controller/ws — interface เดียวกัน
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
