package service

import (
	"sync"

	"github/minyjae/catice/internal/id"
)

// Sessions แมป "token → userID"
// flow: login สำเร็จ → Create(userID) ได้ token → เอาไปใส่ cookie
//        ต่อ ws → อ่าน cookie → UserID(token) → รู้ว่าใคร → client.id = userID
//
// in-memory → ต้องมี lock (HTTP/WS หลาย goroutine). หมายเหตุ: restart แล้ว session หาย (ต้อง login ใหม่)
type Sessions struct {
	mu      sync.RWMutex
	byToken map[string]string // token -> userID
}

func NewSessions() *Sessions {
	return &Sessions{byToken: make(map[string]string)}
}

// Create ออก token ใหม่ผูกกับ userID (เรียกตอน login สำเร็จ)
func (s *Sessions) Create(userID string) string {
	token := id.New()
	s.mu.Lock()
	s.byToken[token] = userID
	s.mu.Unlock()
	return token
}

// UserID แปลง token (จาก cookie) → userID (ok=false ถ้า token ไม่ valid)
func (s *Sessions) UserID(token string) (string, bool) {
	s.mu.RLock()
	uid, ok := s.byToken[token]
	s.mu.RUnlock()
	return uid, ok
}

// Destroy ลบ session (เรียกตอน logout)
func (s *Sessions) Destroy(token string) {
	s.mu.Lock()
	delete(s.byToken, token)
	s.mu.Unlock()
}
