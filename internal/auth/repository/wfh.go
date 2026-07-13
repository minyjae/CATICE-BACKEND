package repository

import (
	"gorm.io/gorm"

	"github/minyjae/catice/internal/auth/domain"
)

// WFHRequestModel = persistence model ของคำขอ work-from-home (รูปร่างตาราง wfh_requests ใน Postgres)
type WFHRequestModel struct {
	ID         string `gorm:"primaryKey"`
	UserID     string `gorm:"index;not null"`
	Date       string `gorm:"index;not null"`
	Reason     string
	Status     string `gorm:"index;not null"`
	ApproverID string `gorm:"index"`
	CreatedAt  int64  `gorm:"index"`
	DecidedAt  int64
}

func (WFHRequestModel) TableName() string { return "wfh_requests" }

func wfhToDomain(m WFHRequestModel) domain.WFHRequest {
	return domain.WFHRequest{
		ID:         m.ID,
		UserID:     m.UserID,
		Date:       m.Date,
		Reason:     m.Reason,
		Status:     domain.RequestStatus(m.Status),
		ApproverID: m.ApproverID,
		CreatedAt:  m.CreatedAt,
		DecidedAt:  m.DecidedAt,
	}
}

func wfhFromDomain(w domain.WFHRequest) WFHRequestModel {
	return WFHRequestModel{
		ID:         w.ID,
		UserID:     w.UserID,
		Date:       w.Date,
		Reason:     w.Reason,
		Status:     string(w.Status),
		ApproverID: w.ApproverID,
		CreatedAt:  w.CreatedAt,
		DecidedAt:  w.DecidedAt,
	}
}

// gormWFH = impl ของ WFHRepository
type gormWFH struct {
	db *gorm.DB
}

// NewGormWFH สร้าง repository + run AutoMigrate สร้าง/อัปเดตตาราง wfh_requests
func NewGormWFH(db *gorm.DB) (*gormWFH, error) {
	if err := db.AutoMigrate(&WFHRequestModel{}); err != nil {
		return nil, err
	}
	return &gormWFH{db: db}, nil
}

func (g *gormWFH) Create(w domain.WFHRequest) error {
	m := wfhFromDomain(w)
	return g.db.Create(&m).Error
}

func (g *gormWFH) Update(w domain.WFHRequest) error {
	m := wfhFromDomain(w)
	return g.db.Save(&m).Error
}

func (g *gormWFH) ByID(id string) (domain.WFHRequest, bool) {
	var m WFHRequestModel
	if err := g.db.First(&m, "id = ?", id).Error; err != nil {
		return domain.WFHRequest{}, false
	}
	return wfhToDomain(m), true
}

func (g *gormWFH) ByUser(userID string) []domain.WFHRequest {
	var ms []WFHRequestModel
	if err := g.db.Where("user_id = ?", userID).Order("created_at desc").Find(&ms).Error; err != nil {
		return nil
	}
	out := make([]domain.WFHRequest, 0, len(ms))
	for _, m := range ms {
		out = append(out, wfhToDomain(m))
	}
	return out
}

func (g *gormWFH) PendingForApprover(approverID string) []domain.WFHRequest {
	var ms []WFHRequestModel
	if err := g.db.Where("approver_id = ? AND status = ?", approverID, string(domain.StatusPending)).
		Order("created_at").Find(&ms).Error; err != nil {
		return nil
	}
	out := make([]domain.WFHRequest, 0, len(ms))
	for _, m := range ms {
		out = append(out, wfhToDomain(m))
	}
	return out
}

// CountApprovedByUserInRange นับ WFH ที่อนุมัติแล้วของ user ในช่วงวันที่กำหนด (ใช้เช็ค quota รายสัปดาห์/เดือน)
func (g *gormWFH) CountApprovedByUserInRange(userID, startDate, endDate string) int {
	var count int64
	g.db.Model(&WFHRequestModel{}).Where(
		"user_id = ? AND status = ? AND date >= ? AND date <= ?",
		userID, string(domain.StatusApproved), startDate, endDate,
	).Count(&count)
	return int(count)
}
