package service

import (
	"strings"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/repository"
	"github/minyjae/catice/internal/id"
)

// TaskStore = business logic ของ task (validate + แจก id + ตั้งค่าดีฟอลต์)
// ส่วน "เก็บจริง" มอบให้ repository (interface เดียว → สลับที่เก็บได้)
type TaskStore struct {
	repo repository.TaskRepository
}

func NewTaskStore(repo repository.TaskRepository) *TaskStore {
	return &TaskStore{repo: repo}
}

// Create สร้าง task ใหม่: server แจก id เอง + ตั้ง status ดีฟอลต์ "todo"
//   - createdBy มาจาก JWT (handler ยื่นให้) ไม่เชื่อ client
//   - คืน task ที่สร้างเสร็จ (พร้อม id) ให้ตอบกลับ frontend
func (s *TaskStore) Create(createdBy string, p domain.CreateTaskPayload) (domain.Task, error) {
	title := strings.TrimSpace(p.Title)
	if title == "" {
		return domain.Task{}, domain.ErrEmptyTitle
	}

	status := p.Status
	if status == "" {
		status = domain.ToDoStatus // ว่าง → ดีฟอลต์ todo
	}
	if !status.Valid() {
		return domain.Task{}, domain.ErrInvalidStatus
	}

	t := domain.Task{
		ID:        id.New(),
		Title:     title,
		Detail:    p.Detail,
		TStatus:   status,
		CreatedBy: createdBy,
		AssignTo:  p.AssignTo,
	}
	if err := s.repo.Create(t); err != nil {
		return domain.Task{}, err
	}
	return t, nil
}

// Move เปลี่ยน status ของ task (ok=false ถ้าไม่มี task นั้น/status ไม่ถูกต้อง)
func (s *TaskStore) Move(taskID string, status domain.Status) (domain.Task, bool) {
	if !status.Valid() {
		return domain.Task{}, false
	}
	t, ok := s.repo.ByID(taskID)
	if !ok {
		return domain.Task{}, false
	}
	t.TStatus = status
	if err := s.repo.Update(t); err != nil {
		return domain.Task{}, false
	}
	return t, true
}

// Update แก้ title/detail/assignTo (ok=false ถ้าไม่มี task นั้น)
func (s *TaskStore) Update(taskID, title, detail string, assignTo []string) (domain.Task, bool) {
	t, ok := s.repo.ByID(taskID)
	if !ok {
		return domain.Task{}, false
	}
	t.Title, t.Detail, t.AssignTo = title, detail, assignTo
	if err := s.repo.Update(t); err != nil {
		return domain.Task{}, false
	}
	return t, true
}

// Delete ลบ task ตาม id
func (s *TaskStore) Delete(taskID string) {
	_ = s.repo.Delete(taskID)
}

// List คืน task ทั้งหมด
func (s *TaskStore) List() []domain.Task {
	return s.repo.All()
}
