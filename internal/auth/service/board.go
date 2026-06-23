package service

import (
	"strings"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/repository"
	"github/minyjae/catice/internal/id"
)

// BoardStore = business logic ของ board (validate ชื่อ + แจก id) — เก็บจริงมอบให้ repository
type BoardStore struct {
	repo repository.BoardRepository
}

func NewBoardStore(repo repository.BoardRepository) *BoardStore {
	return &BoardStore{repo: repo}
}

// Create สร้างบอร์ดใหม่ (server แจก id) — ชื่อต้องไม่ว่าง
func (s *BoardStore) Create(name string) (domain.Board, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Board{}, domain.ErrEmptyBoardName
	}
	b := domain.Board{ID: id.New(), Name: name}
	if err := s.repo.Create(b); err != nil {
		return domain.Board{}, err
	}
	return b, nil
}

// Rename แก้ชื่อบอร์ด (ok=false ถ้าไม่มีบอร์ดนั้น/ชื่อว่าง)
func (s *BoardStore) Rename(boardID, name string) (domain.Board, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Board{}, false
	}
	b, ok := s.repo.ByID(boardID)
	if !ok {
		return domain.Board{}, false
	}
	b.Name = name
	if err := s.repo.Update(b); err != nil {
		return domain.Board{}, false
	}
	return b, true
}

// Delete ลบบอร์ดตาม id (task ของบอร์ด ลบที่ TaskStore แยก — ดู router)
func (s *BoardStore) Delete(boardID string) {
	_ = s.repo.Delete(boardID)
}

// Exists เช็คว่ามีบอร์ดนี้จริงไหม (router ใช้กันสร้าง task ใต้บอร์ดที่ไม่มี)
func (s *BoardStore) Exists(boardID string) bool {
	_, ok := s.repo.ByID(boardID)
	return ok
}

// List คืนบอร์ดทั้งหมด
func (s *BoardStore) List() []domain.Board {
	return s.repo.All()
}
