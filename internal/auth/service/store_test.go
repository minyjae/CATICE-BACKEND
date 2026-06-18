package service

import (
	"errors"
	"testing"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/repository"
)

func TestRegisterAndLogin(t *testing.T) {
	s := NewStore(repository.NewMemUsers())

	u, err := s.Register(domain.RegisterPayload{Email: "a@x.com", Role: domain.RoleDeveloper, Password: "secret"})
	if err != nil {
		t.Fatalf("register error: %v", err)
	}
	if u.ID == "" {
		t.Fatal("ควรได้ id")
	}
	if u.PassHash == "secret" {
		t.Fatal("รหัสต้องถูก hash ไม่ใช่เก็บ plain")
	}

	// login ถูก
	if _, err := s.Login(domain.LoginPayload{Email: "a@x.com", Password: "secret"}); err != nil {
		t.Fatalf("login ที่ถูกควรผ่าน: %v", err)
	}
	// login รหัสผิด
	if _, err := s.Login(domain.LoginPayload{Email: "a@x.com", Password: "wrong"}); !errors.Is(err, domain.ErrBadCredentials) {
		t.Fatalf("รหัสผิดควรได้ ErrBadCredentials ได้ %v", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	s := NewStore(repository.NewMemUsers())

	// role มั่ว
	if _, err := s.Register(domain.RegisterPayload{Email: "b@x.com", Role: "ceo", Password: "p"}); !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf("role มั่วควรถูกปฏิเสธ ได้ %v", err)
	}
	// email ซ้ำ
	s.Register(domain.RegisterPayload{Email: "c@x.com", Role: domain.RolePM, Password: "p"})
	if _, err := s.Register(domain.RegisterPayload{Email: "c@x.com", Role: domain.RolePO, Password: "p"}); !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("email ซ้ำควรถูกปฏิเสธ ได้ %v", err)
	}
	// email ตัวพิมพ์ใหญ่/ช่องว่าง → ต้อง normalize แล้วเจอว่าซ้ำ
	if _, err := s.Register(domain.RegisterPayload{Email: "  C@X.com ", Role: domain.RolePO, Password: "p"}); !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("email ควร normalize แล้วเจอซ้ำ ได้ %v", err)
	}
}

func TestSessions(t *testing.T) {
	sess := NewSessions()
	token := sess.Create("user-1")

	if uid, ok := sess.UserID(token); !ok || uid != "user-1" {
		t.Fatalf("token ควร map กลับเป็น user-1 ได้ %q ok=%v", uid, ok)
	}
	sess.Destroy(token)
	if _, ok := sess.UserID(token); ok {
		t.Fatal("หลัง Destroy token ต้องใช้ไม่ได้")
	}
}
