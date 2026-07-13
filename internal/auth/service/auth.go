// Package service = business logic ของ auth (hash, validate, normalize) + JWT + task
// พึ่ง repository ผ่าน interface → สลับที่เก็บได้โดย service ไม่ต้องแก้
package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/repository"
	"github/minyjae/catice/internal/id"
)

// ===================== user store =====================

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

// GetByID หา user จาก id (เช่น id ที่ได้จาก JWT)
func (s *Store) GetByID(userID string) (domain.User, bool) {
	return s.repo.ByID(userID)
}

// ListUsers คืน user ทั้งหมดที่สมัครไว้ — ใช้ทำ selector ผู้รับมอบหมาย task บน Kanban board
func (s *Store) ListUsers() []domain.User {
	return s.repo.All()
}

// SetManager ตั้ง ManagerID ให้ user (ใครเป็นผู้อนุมัติ leave/WFH ของเขา) — เฉพาะ HR ทำได้
// managerID ว่างได้ (เคลียร์หัวหน้า → fallback ไปหา HR ตอนยื่นคำขอ)
func (s *Store) SetManager(callerRole domain.Role, userID, managerID string) (domain.User, error) {
	if callerRole != domain.RoleHR {
		return domain.User{}, domain.ErrForbidden
	}
	u, ok := s.repo.ByID(userID)
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	u.ManagerID = managerID
	if err := s.repo.Update(u); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

// UpdateProfile แก้ไขโปรไฟล์พนักงาน (partial update) — เฉพาะ HR ทำได้
// field ที่เป็น nil ใน payload = ไม่แก้ field นั้น
func (s *Store) UpdateProfile(callerRole domain.Role, userID string, p domain.UpdateProfilePayload) (domain.User, error) {
	if callerRole != domain.RoleHR {
		return domain.User{}, domain.ErrForbidden
	}
	u, ok := s.repo.ByID(userID)
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	if p.FirstName != nil {
		u.FirstName = *p.FirstName
	}
	if p.LastName != nil {
		u.LastName = *p.LastName
	}
	if p.Phone != nil {
		u.Phone = *p.Phone
	}
	if p.BirthDate != nil {
		u.BirthDate = *p.BirthDate
	}
	if p.Address != nil {
		u.Address = *p.Address
	}
	if p.Salary != nil {
		u.Salary = p.Salary
	}
	if p.StartDate != nil {
		u.StartDate = *p.StartDate
	}
	if err := s.repo.Update(u); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

// ChangeRole เปลี่ยนตำแหน่งพนักงาน (เลื่อนขั้น/ย้ายแผนก) — เฉพาะ HR ทำได้
func (s *Store) ChangeRole(callerRole domain.Role, userID string, newRole domain.Role) (domain.User, error) {
	if callerRole != domain.RoleHR {
		return domain.User{}, domain.ErrForbidden
	}
	if !newRole.Valid() {
		return domain.User{}, domain.ErrInvalidRole
	}
	u, ok := s.repo.ByID(userID)
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	u.Role = newRole
	if err := s.repo.Update(u); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

// DeleteUser soft-delete พนักงานออกจากระบบ (ลาออก/เลิกจ้าง) — เฉพาะ HR ทำได้
// user login ไม่ได้อีก, ไม่ปรากฏใน All() แต่ข้อมูล leave/WFH/diary ยังอยู่เป็น audit trail
func (s *Store) DeleteUser(callerRole domain.Role, userID string) error {
	if callerRole != domain.RoleHR {
		return domain.ErrForbidden
	}
	if _, ok := s.repo.ByID(userID); !ok {
		return domain.ErrUserNotFound
	}
	return s.repo.Delete(userID)
}

// ===================== JWT =====================

// Tokens ออก/ตรวจ JWT (HS256) แบบ stateless — แทน session store แบบ in-memory เดิม
// flow: login สำเร็จ → Create(userID) ได้ JWT → client เก็บไว้ (เช่น localStorage)
//
//	ทุก request แนบ "Authorization: Bearer <jwt>" → UserID(jwt) → รู้ว่าใคร
//
// stateless = server ไม่เก็บ state เลย → รอด restart, scale หลาย instance ได้
// แลกกับ: revoke token ทันทีไม่ได้ (ใช้ได้จนหมดอายุ) → logout = ฝั่ง client ทิ้ง token เอง
type Tokens struct {
	secret []byte
	ttl    time.Duration
}

func NewTokens(secret string) *Tokens {
	return &Tokens{secret: []byte(secret), ttl: 7 * 24 * time.Hour} // อยู่ได้ 7 วัน
}

// header ของ JWT คงที่ → คำนวณครั้งเดียว
var jwtHeader = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

// jwtClaims = payload ของ token (sub=userID, iat/exp=วินาที unix)
type jwtClaims struct {
	Sub string `json:"sub"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
}

// Create ออก JWT ผูกกับ userID (อยู่ใน claim "sub") อายุ ttl
func (t *Tokens) Create(userID string) string {
	now := time.Now()
	body, _ := json.Marshal(jwtClaims{Sub: userID, Iat: now.Unix(), Exp: now.Add(t.ttl).Unix()})
	payload := base64.RawURLEncoding.EncodeToString(body)
	signing := jwtHeader + "." + payload
	return signing + "." + t.sign(signing)
}

// UserID ตรวจลายเซ็น + วันหมดอายุ → คืน userID (ok=false ถ้า token เสีย/ถูกแก้/หมดอายุ)
func (t *Tokens) UserID(token string) (string, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", false
	}
	signing := parts[0] + "." + parts[1]
	// hmac.Equal เทียบแบบ constant-time → กัน timing attack
	if !hmac.Equal([]byte(parts[2]), []byte(t.sign(signing))) {
		return "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}
	var c jwtClaims
	if err := json.Unmarshal(raw, &c); err != nil {
		return "", false
	}
	if c.Sub == "" || time.Now().Unix() >= c.Exp {
		return "", false
	}
	return c.Sub, true
}

// sign = HMAC-SHA256 ของ "<header>.<payload>" แล้ว encode แบบ base64url
func (t *Tokens) sign(signing string) string {
	mac := hmac.New(sha256.New, t.secret)
	mac.Write([]byte(signing))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
