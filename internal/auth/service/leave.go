package service

import (
	"strings"
	"time"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/repository"
	"github/minyjae/catice/internal/id"
)

// LeaveStore = business logic ของคำขอลา (validate + หา approver + อนุมัติ/ปฏิเสธ + เช็ค quota)
type LeaveStore struct {
	repo   repository.LeaveRepository
	users  repository.UsersRepository
	policy *PolicyStore
}

func NewLeaveStore(repo repository.LeaveRepository, users repository.UsersRepository, policy *PolicyStore) *LeaveStore {
	return &LeaveStore{repo: repo, users: users, policy: policy}
}

// resolveApprover หาผู้อนุมัติของ userID: ใช้ ManagerID ถ้ามี ไม่งั้น fallback ไปหา user แรกที่ role เป็น HR
func resolveApprover(users repository.UsersRepository, userID string) (string, error) {
	u, ok := users.ByID(userID)
	if !ok {
		return "", domain.ErrUserNotFound
	}
	if u.ManagerID != "" {
		return u.ManagerID, nil
	}
	for _, other := range users.All() {
		if other.Role == domain.RoleHR {
			return other.ID, nil
		}
	}
	return "", domain.ErrNoApprover
}

// Create ยื่นคำขอลาใหม่: server แจก id เอง + หา approver + เช็ค quota ก่อน create
//   - userID มาจาก JWT (handler ยื่นให้) ไม่เชื่อ client
func (s *LeaveStore) Create(userID string, p domain.CreateLeavePayload) (domain.LeaveRequest, error) {
	if !p.Type.Valid() {
		return domain.LeaveRequest{}, domain.ErrEmptyLeaveType
	}
	start, end := strings.TrimSpace(p.StartDate), strings.TrimSpace(p.EndDate)
	if start == "" || end == "" || start > end {
		return domain.LeaveRequest{}, domain.ErrInvalidDateRange
	}

	if err := s.checkLeaveQuota(userID, p.Type, start, end); err != nil {
		return domain.LeaveRequest{}, err
	}

	approverID, err := resolveApprover(s.users, userID)
	if err != nil {
		return domain.LeaveRequest{}, err
	}

	l := domain.LeaveRequest{
		ID:         id.New(),
		UserID:     userID,
		Type:       p.Type,
		StartDate:  start,
		EndDate:    end,
		Reason:     p.Reason,
		Status:     domain.StatusPending,
		ApproverID: approverID,
		CreatedAt:  time.Now().Unix(),
	}
	if err := s.repo.Create(l); err != nil {
		return domain.LeaveRequest{}, err
	}
	return l, nil
}

// Decide อนุมัติ/ปฏิเสธคำขอ — ต้องเป็น approver ของคำขอนั้น และคำขอต้องยัง pending อยู่
func (s *LeaveStore) Decide(callerID, requestID string, approve bool) (domain.LeaveRequest, error) {
	l, ok := s.repo.ByID(requestID)
	if !ok {
		return domain.LeaveRequest{}, domain.ErrRequestNotFound
	}
	if l.ApproverID != callerID {
		return domain.LeaveRequest{}, domain.ErrNotApprover
	}
	if l.Status != domain.StatusPending {
		return domain.LeaveRequest{}, domain.ErrNotPending
	}

	if approve {
		l.Status = domain.StatusApproved
	} else {
		l.Status = domain.StatusRejected
	}
	l.DecidedAt = time.Now().Unix()
	if err := s.repo.Update(l); err != nil {
		return domain.LeaveRequest{}, err
	}
	return l, nil
}

// ListMine คืนคำขอลาทั้งหมดของ user คนหนึ่ง
func (s *LeaveStore) ListMine(userID string) []domain.LeaveRequest {
	return s.repo.ByUser(userID)
}

// ListPending คืนคำขอลาที่รอ approver คนนี้ตัดสินใจ
func (s *LeaveStore) ListPending(approverID string) []domain.LeaveRequest {
	return s.repo.PendingForApprover(approverID)
}

// checkLeaveQuota ตรวจสอบว่าจำนวนวันลาของ user ในปีนั้นไม่เกิน policy
func (s *LeaveStore) checkLeaveQuota(userID string, leaveType domain.LeaveType, start, end string) error {
	policy := s.policy.Get()
	limit := leaveLimitForType(policy, leaveType)

	t1, _ := time.Parse("2006-01-02", start)
	approved := s.repo.ApprovedByUserTypeYear(userID, leaveType, t1.Year())

	used := 0
	for _, l := range approved {
		used += countDays(l.StartDate, l.EndDate)
	}
	if used+countDays(start, end) > limit {
		return domain.ErrLeaveQuotaExceeded
	}
	return nil
}

func leaveLimitForType(p domain.LeavePolicy, t domain.LeaveType) int {
	switch t {
	case domain.LeaveVacation:
		return p.VacationDaysPerYear
	case domain.LeaveSick:
		return p.SickDaysPerYear
	case domain.LeavePersonal:
		return p.PersonalDaysPerYear
	}
	return 0
}

// countDays คืนจำนวนวันตามปฏิทิน (รวมวันเริ่มและวันสิ้นสุด)
func countDays(start, end string) int {
	t1, _ := time.Parse("2006-01-02", start)
	t2, _ := time.Parse("2006-01-02", end)
	return int(t2.Sub(t1).Hours()/24) + 1
}
