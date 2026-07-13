package service

import (
	"strings"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/repository"
	"github/minyjae/catice/internal/id"
)

// HolidayStore = business logic ของวันหยุดบริษัท (เฉพาะ HR สร้าง/ลบได้ ใครก็ดูได้)
type HolidayStore struct {
	repo repository.HolidayRepository
}

func NewHolidayStore(repo repository.HolidayRepository) *HolidayStore {
	return &HolidayStore{repo: repo}
}

// Create เพิ่มวันหยุดใหม่ — เฉพาะ callerRole เป็น HR เท่านั้น
func (s *HolidayStore) Create(callerRole domain.Role, createdBy string, p domain.CreateHolidayPayload) (domain.Holiday, error) {
	if callerRole != domain.RoleHR {
		return domain.Holiday{}, domain.ErrForbidden
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return domain.Holiday{}, domain.ErrEmptyHolidayName
	}

	h := domain.Holiday{ID: id.New(), Name: name, Date: p.Date, CreatedBy: createdBy}
	if err := s.repo.Create(h); err != nil {
		return domain.Holiday{}, err
	}
	return h, nil
}

// Delete ลบวันหยุด — เฉพาะ callerRole เป็น HR เท่านั้น
func (s *HolidayStore) Delete(callerRole domain.Role, id string) error {
	if callerRole != domain.RoleHR {
		return domain.ErrForbidden
	}
	return s.repo.Delete(id)
}

// List คืนวันหยุดทั้งหมด
func (s *HolidayStore) List() []domain.Holiday {
	return s.repo.All()
}
