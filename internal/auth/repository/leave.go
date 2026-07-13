package repository

import (
	"fmt"

	"gorm.io/gorm"

	"github/minyjae/catice/internal/auth/domain"
)

// LeaveRequestModel = persistence model ของคำขอลา (รูปร่างตาราง leave_requests ใน Postgres)
type LeaveRequestModel struct {
	ID         string `gorm:"primaryKey"`
	UserID     string `gorm:"index;not null"`
	Type       string `gorm:"not null"`
	StartDate  string `gorm:"not null"`
	EndDate    string `gorm:"not null"`
	Reason     string
	Status     string `gorm:"index;not null"`
	ApproverID string `gorm:"index"`
	CreatedAt  int64  `gorm:"index"`
	DecidedAt  int64
}

func (LeaveRequestModel) TableName() string { return "leave_requests" }

func leaveToDomain(m LeaveRequestModel) domain.LeaveRequest {
	return domain.LeaveRequest{
		ID:         m.ID,
		UserID:     m.UserID,
		Type:       domain.LeaveType(m.Type),
		StartDate:  m.StartDate,
		EndDate:    m.EndDate,
		Reason:     m.Reason,
		Status:     domain.RequestStatus(m.Status),
		ApproverID: m.ApproverID,
		CreatedAt:  m.CreatedAt,
		DecidedAt:  m.DecidedAt,
	}
}

func leaveFromDomain(l domain.LeaveRequest) LeaveRequestModel {
	return LeaveRequestModel{
		ID:         l.ID,
		UserID:     l.UserID,
		Type:       string(l.Type),
		StartDate:  l.StartDate,
		EndDate:    l.EndDate,
		Reason:     l.Reason,
		Status:     string(l.Status),
		ApproverID: l.ApproverID,
		CreatedAt:  l.CreatedAt,
		DecidedAt:  l.DecidedAt,
	}
}

// gormLeaves = impl ของ LeaveRepository
type gormLeaves struct {
	db *gorm.DB
}

// NewGormLeaves สร้าง repository + run AutoMigrate สร้าง/อัปเดตตาราง leave_requests
func NewGormLeaves(db *gorm.DB) (*gormLeaves, error) {
	if err := db.AutoMigrate(&LeaveRequestModel{}); err != nil {
		return nil, err
	}
	return &gormLeaves{db: db}, nil
}

func (g *gormLeaves) Create(l domain.LeaveRequest) error {
	m := leaveFromDomain(l)
	return g.db.Create(&m).Error
}

// Update เซฟทับทั้งใบ (service อ่านของเดิมมาก่อนแล้วแก้ status/decidedAt)
func (g *gormLeaves) Update(l domain.LeaveRequest) error {
	m := leaveFromDomain(l)
	return g.db.Save(&m).Error
}

func (g *gormLeaves) ByID(id string) (domain.LeaveRequest, bool) {
	var m LeaveRequestModel
	if err := g.db.First(&m, "id = ?", id).Error; err != nil {
		return domain.LeaveRequest{}, false
	}
	return leaveToDomain(m), true
}

// ByUser คืนคำขอลาทั้งหมดของ user คนหนึ่ง เรียงตามวันสร้างล่าสุดก่อน
func (g *gormLeaves) ByUser(userID string) []domain.LeaveRequest {
	var ms []LeaveRequestModel
	if err := g.db.Where("user_id = ?", userID).Order("created_at desc").Find(&ms).Error; err != nil {
		return nil
	}
	out := make([]domain.LeaveRequest, 0, len(ms))
	for _, m := range ms {
		out = append(out, leaveToDomain(m))
	}
	return out
}

// PendingForApprover คืนคำขอลาที่รอ approver คนนี้ตัดสินใจ
func (g *gormLeaves) PendingForApprover(approverID string) []domain.LeaveRequest {
	var ms []LeaveRequestModel
	if err := g.db.Where("approver_id = ? AND status = ?", approverID, string(domain.StatusPending)).
		Order("created_at").Find(&ms).Error; err != nil {
		return nil
	}
	out := make([]domain.LeaveRequest, 0, len(ms))
	for _, m := range ms {
		out = append(out, leaveToDomain(m))
	}
	return out
}

// ApprovedByUserTypeYear คืนคำขอลาที่อนุมัติแล้วของ user ในปีที่กำหนด (ใช้นับวันเทียบ quota)
func (g *gormLeaves) ApprovedByUserTypeYear(userID string, leaveType domain.LeaveType, year int) []domain.LeaveRequest {
	yearStart := fmt.Sprintf("%d-01-01", year)
	yearEnd := fmt.Sprintf("%d-12-31", year)
	var ms []LeaveRequestModel
	if err := g.db.Where(
		"user_id = ? AND type = ? AND status = ? AND start_date >= ? AND start_date <= ?",
		userID, string(leaveType), string(domain.StatusApproved), yearStart, yearEnd,
	).Find(&ms).Error; err != nil {
		return nil
	}
	out := make([]domain.LeaveRequest, 0, len(ms))
	for _, m := range ms {
		out = append(out, leaveToDomain(m))
	}
	return out
}
