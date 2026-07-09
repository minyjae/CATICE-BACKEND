package service

import (
	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/repository"
)

// PolicyStore = business logic ของ leave/WFH policy (get/update)
// เฉพาะ HR ถึง update ได้ — permission check ทำที่นี่ ไม่ใช่ handler
type PolicyStore struct {
	repo repository.PolicyRepository
}

func NewPolicyStore(repo repository.PolicyRepository) *PolicyStore {
	return &PolicyStore{repo: repo}
}

func (s *PolicyStore) Get() domain.LeavePolicy {
	return s.repo.Get()
}

func (s *PolicyStore) Update(callerRole domain.Role, p domain.LeavePolicy) (domain.LeavePolicy, error) {
	if callerRole != domain.RoleHR {
		return domain.LeavePolicy{}, domain.ErrForbidden
	}
	if p.VacationDaysPerYear < 0 || p.SickDaysPerYear < 0 || p.PersonalDaysPerYear < 0 ||
		p.WFHDaysPerWeek < 0 || p.WFHDaysPerMonth < 0 {
		return domain.LeavePolicy{}, domain.ErrInvalidPolicy
	}
	if err := s.repo.Save(p); err != nil {
		return domain.LeavePolicy{}, err
	}
	return p, nil
}
