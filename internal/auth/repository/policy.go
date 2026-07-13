package repository

import (
	"errors"

	"gorm.io/gorm"

	"github/minyjae/catice/internal/auth/domain"
)

// LeavePolicyModel = persistence model ของนโยบาย leave/WFH (รูปร่างตาราง leave_policy ใน Postgres)
// มีแถวเดียวเสมอ — ID คงที่เป็น "company"
type LeavePolicyModel struct {
	ID                  string `gorm:"primaryKey"`
	VacationDaysPerYear int    `gorm:"not null"`
	SickDaysPerYear     int    `gorm:"not null"`
	PersonalDaysPerYear int    `gorm:"not null"`
	WFHDaysPerWeek      int    `gorm:"not null"`
	WFHDaysPerMonth     int    `gorm:"not null"`
}

func (LeavePolicyModel) TableName() string { return "leave_policy" }

type gormPolicy struct {
	db *gorm.DB
}

// NewGormPolicy สร้าง repository + run AutoMigrate สร้าง/อัปเดตตาราง leave_policy
func NewGormPolicy(db *gorm.DB) (*gormPolicy, error) {
	if err := db.AutoMigrate(&LeavePolicyModel{}); err != nil {
		return nil, err
	}
	return &gormPolicy{db: db}, nil
}

// Get คืน policy ปัจจุบัน — ถ้ายังไม่มีใน DB คืน DefaultPolicy
func (g *gormPolicy) Get() domain.LeavePolicy {
	var m LeavePolicyModel
	if err := g.db.First(&m, "id = ?", "company").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.DefaultPolicy
		}
		return domain.DefaultPolicy
	}
	return domain.LeavePolicy{
		VacationDaysPerYear: m.VacationDaysPerYear,
		SickDaysPerYear:     m.SickDaysPerYear,
		PersonalDaysPerYear: m.PersonalDaysPerYear,
		WFHDaysPerWeek:      m.WFHDaysPerWeek,
		WFHDaysPerMonth:     m.WFHDaysPerMonth,
	}
}

// Save upsert policy (สร้างหรืออัปเดตแถวเดียวด้วย id = "company")
func (g *gormPolicy) Save(p domain.LeavePolicy) error {
	m := LeavePolicyModel{
		ID:                  "company",
		VacationDaysPerYear: p.VacationDaysPerYear,
		SickDaysPerYear:     p.SickDaysPerYear,
		PersonalDaysPerYear: p.PersonalDaysPerYear,
		WFHDaysPerWeek:      p.WFHDaysPerWeek,
		WFHDaysPerMonth:     p.WFHDaysPerMonth,
	}
	return g.db.Save(&m).Error
}
