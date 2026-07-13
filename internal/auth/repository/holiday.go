package repository

import (
	"gorm.io/gorm"

	"github/minyjae/catice/internal/auth/domain"
)

// HolidayModel = persistence model ของวันหยุดบริษัท (รูปร่างตาราง holidays ใน Postgres)
type HolidayModel struct {
	ID        string `gorm:"primaryKey"`
	Name      string `gorm:"not null"`
	Date      string `gorm:"index;not null"`
	CreatedBy string `gorm:"index"`
}

func (HolidayModel) TableName() string { return "holidays" }

func holidayToDomain(m HolidayModel) domain.Holiday {
	return domain.Holiday{ID: m.ID, Name: m.Name, Date: m.Date, CreatedBy: m.CreatedBy}
}

func holidayFromDomain(h domain.Holiday) HolidayModel {
	return HolidayModel{ID: h.ID, Name: h.Name, Date: h.Date, CreatedBy: h.CreatedBy}
}

// gormHolidays = impl ของ HolidayRepository
type gormHolidays struct {
	db *gorm.DB
}

// NewGormHolidays สร้าง repository + run AutoMigrate สร้าง/อัปเดตตาราง holidays
func NewGormHolidays(db *gorm.DB) (*gormHolidays, error) {
	if err := db.AutoMigrate(&HolidayModel{}); err != nil {
		return nil, err
	}
	return &gormHolidays{db: db}, nil
}

func (g *gormHolidays) Create(h domain.Holiday) error {
	m := holidayFromDomain(h)
	return g.db.Create(&m).Error
}

func (g *gormHolidays) Delete(id string) error {
	return g.db.Delete(&HolidayModel{}, "id = ?", id).Error
}

// All คืนวันหยุดทั้งหมด เรียงตามวันที่
func (g *gormHolidays) All() []domain.Holiday {
	var ms []HolidayModel
	if err := g.db.Order("date").Find(&ms).Error; err != nil {
		return nil
	}
	out := make([]domain.Holiday, 0, len(ms))
	for _, m := range ms {
		out = append(out, holidayToDomain(m))
	}
	return out
}
