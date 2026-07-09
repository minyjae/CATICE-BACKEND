package service

import (
	"strings"
	"time"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/repository"
	"github/minyjae/catice/internal/id"
)

// WFHStore = business logic ของคำขอ work-from-home (validate + หา approver + อนุมัติ/ปฏิเสธ + เช็ค quota)
type WFHStore struct {
	repo   repository.WFHRepository
	users  repository.UsersRepository
	policy *PolicyStore
}

func NewWFHStore(repo repository.WFHRepository, users repository.UsersRepository, policy *PolicyStore) *WFHStore {
	return &WFHStore{repo: repo, users: users, policy: policy}
}

// Create ยื่นคำขอ WFH ใหม่ (1 วัน) — server แจก id เอง + หา approver + เช็ค quota ก่อน create
func (s *WFHStore) Create(userID string, p domain.CreateWFHPayload) (domain.WFHRequest, error) {
	date := strings.TrimSpace(p.Date)
	if date == "" {
		return domain.WFHRequest{}, domain.ErrInvalidDateRange
	}

	if err := s.checkWFHQuota(userID, date); err != nil {
		return domain.WFHRequest{}, err
	}

	approverID, err := resolveApprover(s.users, userID)
	if err != nil {
		return domain.WFHRequest{}, err
	}

	w := domain.WFHRequest{
		ID:         id.New(),
		UserID:     userID,
		Date:       date,
		Reason:     p.Reason,
		Status:     domain.StatusPending,
		ApproverID: approverID,
		CreatedAt:  time.Now().Unix(),
	}
	if err := s.repo.Create(w); err != nil {
		return domain.WFHRequest{}, err
	}
	return w, nil
}

// Decide อนุมัติ/ปฏิเสธคำขอ — ต้องเป็น approver ของคำขอนั้น และคำขอต้องยัง pending อยู่
func (s *WFHStore) Decide(callerID, requestID string, approve bool) (domain.WFHRequest, error) {
	w, ok := s.repo.ByID(requestID)
	if !ok {
		return domain.WFHRequest{}, domain.ErrRequestNotFound
	}
	if w.ApproverID != callerID {
		return domain.WFHRequest{}, domain.ErrNotApprover
	}
	if w.Status != domain.StatusPending {
		return domain.WFHRequest{}, domain.ErrNotPending
	}

	if approve {
		w.Status = domain.StatusApproved
	} else {
		w.Status = domain.StatusRejected
	}
	w.DecidedAt = time.Now().Unix()
	if err := s.repo.Update(w); err != nil {
		return domain.WFHRequest{}, err
	}
	return w, nil
}

// ListMine คืนคำขอ WFH ทั้งหมดของ user คนหนึ่ง
func (s *WFHStore) ListMine(userID string) []domain.WFHRequest {
	return s.repo.ByUser(userID)
}

// ListPending คืนคำขอ WFH ที่รอ approver คนนี้ตัดสินใจ
func (s *WFHStore) ListPending(approverID string) []domain.WFHRequest {
	return s.repo.PendingForApprover(approverID)
}

// checkWFHQuota ตรวจสอบ quota WFH รายสัปดาห์และรายเดือน
func (s *WFHStore) checkWFHQuota(userID, date string) error {
	policy := s.policy.Get()
	t, _ := time.Parse("2006-01-02", date)

	// รายสัปดาห์ (จันทร์–อาทิตย์)
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7 (ISO week)
	}
	monday := t.AddDate(0, 0, -(weekday - 1))
	sunday := monday.AddDate(0, 0, 6)
	weekCount := s.repo.CountApprovedByUserInRange(userID, monday.Format("2006-01-02"), sunday.Format("2006-01-02"))
	if weekCount >= policy.WFHDaysPerWeek {
		return domain.ErrWFHWeeklyExceeded
	}

	// รายเดือน
	monthStart := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)
	monthCount := s.repo.CountApprovedByUserInRange(userID, monthStart.Format("2006-01-02"), monthEnd.Format("2006-01-02"))
	if monthCount >= policy.WFHDaysPerMonth {
		return domain.ErrWFHMonthlyExceeded
	}

	return nil
}
