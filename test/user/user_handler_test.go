package user_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/handler"
	"github/minyjae/catice/internal/auth/service"
	"github/minyjae/catice/test/fakes"
)

func newAuthHandler(users *fakes.Users) *handler.AuthHandler {
	return handler.NewAuthHandler(service.NewStore(users), nil) // tokens ไม่ถูกใช้โดย Users/SetManager
}

// เทส: AuthHandler.Users ใส่ manager_id ใน response เฉพาะ user ที่มี manager จริง (omitempty ทำงานถูกต้อง)
// input: emp1 มี ManagerID="mgr1", emp2 ไม่มี ManagerID เลย
// aspect: JSON ของ emp1 ต้องมี key manager_id="mgr1", ส่วน emp2 ต้อง "ไม่มี key manager_id เลย" (ไม่ใช่แค่ค่าว่าง)
func TestUsersIncludesManagerID(t *testing.T) {
	users := fakes.NewUsers()
	users.Seed(domain.User{ID: "emp1", Email: "emp1@x.com", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	users.Seed(domain.User{ID: "emp2", Email: "emp2@x.com", Role: domain.RoleDeveloper})
	h := newAuthHandler(users)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	h.Users(rec, req)

	var raw []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	var withManager, withoutManager map[string]any
	for _, u := range raw {
		switch u["id"] {
		case "emp1":
			withManager = u
		case "emp2":
			withoutManager = u
		}
	}
	if withManager == nil || withManager["manager_id"] != "mgr1" {
		t.Fatalf("emp1 ควรมี manager_id=mgr1 ได้: %+v", withManager)
	}
	if withoutManager == nil {
		t.Fatal("emp2 ควรอยู่ใน response")
	}
	if _, exists := withoutManager["manager_id"]; exists {
		t.Fatalf("emp2 ไม่มี manager ควรไม่มี key manager_id เลย (omitempty) ได้: %+v", withoutManager)
	}
}

// เทส: AuthHandler.SetManager ทาง happy path โดยผู้เรียก role HR
// input: HTTP PATCH /users/emp1/manager body {"manager_id":"mgr1"} โดยผู้เรียก role=HR
// aspect: status 200 และค่า ManagerID ของ emp1 ใน repo ต้องถูกอัปเดตเป็น mgr1 จริง
func TestSetManagerHandlerByHR(t *testing.T) {
	users := fakes.NewUsers()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper})
	h := newAuthHandler(users)

	req := httptest.NewRequest(http.MethodPatch, "/users/emp1/manager", strings.NewReader(`{"manager_id":"mgr1"}`))
	req.SetPathValue("id", "emp1")
	req = handler.WithUser(req, domain.User{ID: "hr1", Role: domain.RoleHR})
	rec := httptest.NewRecorder()
	h.SetManager(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	updated, _ := users.ByID("emp1")
	if updated.ManagerID != "mgr1" {
		t.Fatalf("ManagerID ควรถูกอัปเดต ได้ %q", updated.ManagerID)
	}
}

// เทส: AuthHandler.SetManager ปฏิเสธผู้เรียกที่ role ไม่ใช่ HR
// input: body ที่ถูกต้อง แต่ผู้เรียก role=developer
// aspect: status ต้องเป็น 403
func TestSetManagerHandlerForbiddenForNonHR(t *testing.T) {
	users := fakes.NewUsers()
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper})
	h := newAuthHandler(users)

	req := httptest.NewRequest(http.MethodPatch, "/users/emp1/manager", strings.NewReader(`{"manager_id":"mgr1"}`))
	req.SetPathValue("id", "emp1")
	req = handler.WithUser(req, domain.User{ID: "emp2", Role: domain.RoleDeveloper})
	rec := httptest.NewRecorder()
	h.SetManager(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

// เทส: AuthHandler.SetManager กับ user id ที่ไม่มีอยู่จริง
// input: path value id="nonexistent" ที่ไม่เคย seed ไว้เลย โดยผู้เรียก role=HR
// aspect: status ต้องเป็น 404
func TestSetManagerHandlerUserNotFound(t *testing.T) {
	users := fakes.NewUsers()
	h := newAuthHandler(users)

	req := httptest.NewRequest(http.MethodPatch, "/users/nonexistent/manager", strings.NewReader(`{"manager_id":"mgr1"}`))
	req.SetPathValue("id", "nonexistent")
	req = handler.WithUser(req, domain.User{ID: "hr1", Role: domain.RoleHR})
	rec := httptest.NewRecorder()
	h.SetManager(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
