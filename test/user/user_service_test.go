package user_test

import (
	"testing"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/service"
	"github/minyjae/catice/test/fakes"
)

func newUserStore() (*service.Store, *fakes.Users) {
	users := fakes.NewUsers()
	return service.NewStore(users), users
}

// เทส: Store.SetManager โดย caller role HR ตั้ง manager ให้ user อื่นสำเร็จ
// input: callerRole=RoleHR, userID="emp1", managerID="mgr1"
// aspect: ไม่ error, ManagerID ของ user ที่คืนมาต้องเป็น mgr1 และค่าที่บันทึกจริงใน repo ก็ต้องตรงกัน
func TestStoreSetManagerByHR(t *testing.T) {
	store, users := newUserStore()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper})

	updated, err := store.SetManager(domain.RoleHR, "emp1", "mgr1")
	if err != nil {
		t.Fatalf("SetManager error: %v", err)
	}
	if updated.ManagerID != "mgr1" {
		t.Fatalf("ManagerID ควรถูกอัปเดตเป็น mgr1 ได้ %q", updated.ManagerID)
	}

	stored, _ := users.ByID("emp1")
	if stored.ManagerID != "mgr1" {
		t.Fatalf("ค่าใน repo ควรถูกอัปเดตด้วย ได้ %q", stored.ManagerID)
	}
}

// เทส: Store.SetManager ต้องปฏิเสธ caller ที่ role ไม่ใช่ HR
// input: callerRole=RoleDeveloper
// aspect: error ต้องเป็น ErrForbidden
func TestStoreSetManagerForbiddenForNonHR(t *testing.T) {
	store, users := newUserStore()
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper})

	_, err := store.SetManager(domain.RoleDeveloper, "emp1", "mgr1")
	if err != domain.ErrForbidden {
		t.Fatalf("error ควรเป็น ErrForbidden ได้ %v", err)
	}
}

// เทส: Store.SetManager กับ user id ที่ไม่มีอยู่จริงในระบบ
// input: userID="nonexistent" ที่ไม่เคย seed ไว้เลย
// aspect: error ต้องเป็น ErrUserNotFound
func TestStoreSetManagerUserNotFound(t *testing.T) {
	store, _ := newUserStore()

	_, err := store.SetManager(domain.RoleHR, "nonexistent", "mgr1")
	if err != domain.ErrUserNotFound {
		t.Fatalf("error ควรเป็น ErrUserNotFound ได้ %v", err)
	}
}
