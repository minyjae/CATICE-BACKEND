// Package service = business logic ของ auth (hash, validate, normalize) + session
// พึ่ง repository ผ่าน interface → สลับที่เก็บได้โดย service ไม่ต้องแก้
package service

import (
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/repository"
	"github/minyjae/catice/internal/id"
)

// Store = service ของ user (สมัคร/ล็อกอิน/ดึงรายชื่อ) — ส่วน "เก็บจริง" มอบให้ repository
type Store struct {
	repo repository.UsersRepository
}

func NewStore(repo repository.UsersRepository) *Store {
	return &Store{repo: repo}
}

// hashPassword: plain → hash (bcrypt ฝัง salt + ทน brute-force) — เก็บแต่ hash
func hashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// checkPassword: เทียบ plain กับ hash (bcrypt ถอด salt จาก hash มาเทียบเอง)
func checkPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func normEmail(s string) string { return strings.TrimSpace(strings.ToLower(s)) }

// Register สร้างบัญชีใหม่ → คืน User (พร้อม id คงที่)
func (s *Store) Register(p domain.RegisterPayload) (domain.User, error) {
	email := normEmail(p.Email)
	if email == "" || p.Password == "" {
		return domain.User{}, domain.ErrMissingFields
	}
	if !p.Role.Valid() {
		return domain.User{}, domain.ErrInvalidRole
	}

	hash, err := hashPassword(p.Password)
	if err != nil {
		return domain.User{}, err
	}

	u := domain.User{ID: id.New(), Email: email, Role: p.Role, PassHash: hash}
	if err := s.repo.Create(u); err != nil { // repo เช็ค email ซ้ำให้ (อะตอมมิก)
		return domain.User{}, err
	}
	return u, nil
}

// Login ตรวจ email + password → คืน User ถ้าถูก
func (s *Store) Login(p domain.LoginPayload) (domain.User, error) {
	u, ok := s.repo.ByEmail(normEmail(p.Email))
	// ไม่บอกแยกว่า "ไม่มี email" หรือ "รหัสผิด" (กันคนเดา email ที่มีอยู่)
	if !ok || !checkPassword(u.PassHash, p.Password) {
		return domain.User{}, domain.ErrBadCredentials
	}
	return u, nil
}

// GetByID หา user จาก id (เช่น id ที่ได้จาก session/cookie)
func (s *Store) GetByID(userID string) (domain.User, bool) {
	return s.repo.ByID(userID)
}

// ListUsers คืน user ทั้งหมดที่สมัครไว้ — ใช้ทำ selector ผู้รับมอบหมาย task บน Kanban board
func (s *Store) ListUsers() []domain.User {
	return s.repo.All()
}
