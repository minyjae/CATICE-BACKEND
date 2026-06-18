package repository

import (
	"sync"

	"github/minyjae/catice/internal/auth/domain"
)

// memUsers = impl แบบ in-memory (restart แล้วหาย — เริ่มต้นง่าย/ใช้ตอน dev/test)
// มี mutex เพราะ HTTP handler รันหลาย goroutine พร้อมกัน
type memUsers struct {
	mu      sync.RWMutex
	byEmail map[string]domain.User
	byID    map[string]domain.User
}

func NewMemUsers() *memUsers {
	return &memUsers{
		byEmail: make(map[string]domain.User),
		byID:    make(map[string]domain.User),
	}
}

func (m *memUsers) Create(u domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.byEmail[u.Email]; exists {
		return domain.ErrEmailTaken
	}
	m.byEmail[u.Email] = u
	m.byID[u.ID] = u
	return nil
}

func (m *memUsers) ByEmail(email string) (domain.User, bool) {
	m.mu.RLock()
	u, ok := m.byEmail[email]
	m.mu.RUnlock()
	return u, ok
}

func (m *memUsers) ByID(id string) (domain.User, bool) {
	m.mu.RLock()
	u, ok := m.byID[id]
	m.mu.RUnlock()
	return u, ok
}

func (m *memUsers) All() []domain.User {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.User, 0, len(m.byID))
	for _, u := range m.byID {
		out = append(out, u)
	}
	return out
}
