package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/repository"
	"github/minyjae/catice/internal/auth/service"
)

func postJSON(fn http.HandlerFunc, body string) *http.Response {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	w := httptest.NewRecorder()
	fn(w, req)
	return w.Result()
}

func newHandler() *Handler {
	return NewHandler(service.NewStore(repository.NewMemUsers()), service.NewSessions())
}

// register → ต้องได้ 200 + cookie session
func TestRegisterEndpointSetsCookie(t *testing.T) {
	h := newHandler()
	res := postJSON(h.Register, `{"email":"a@x.com","role":"developer","password":"secret"}`)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("ควร 200 ได้ %d", res.StatusCode)
	}
	cookies := res.Cookies()
	if len(cookies) == 0 || cookies[0].Name != CookieName || cookies[0].Value == "" {
		t.Fatalf("ควร set cookie %q ได้ %+v", CookieName, cookies)
	}
}

// role มั่ว → 400
func TestRegisterEndpointBadRole(t *testing.T) {
	h := newHandler()
	res := postJSON(h.Register, `{"email":"b@x.com","role":"ceo","password":"p"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("role มั่วควร 400 ได้ %d", res.StatusCode)
	}
}

// login แล้วเอา cookie ไปแปลงกลับเป็น userID ได้
func TestLoginThenResolveUser(t *testing.T) {
	store := service.NewStore(repository.NewMemUsers())
	sessions := service.NewSessions()
	h := NewHandler(store, sessions)

	store.Register(domain.RegisterPayload{Email: "c@x.com", Role: domain.RolePM, Password: "pw"})
	res := postJSON(h.Login, `{"email":"c@x.com","password":"pw"}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login ควร 200 ได้ %d", res.StatusCode)
	}

	// จำลอง ws อ่าน cookie เดียวกัน → ต้องได้ userID
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	for _, c := range res.Cookies() {
		req.AddCookie(c)
	}
	if _, ok := h.UserIDFromRequest(req); !ok {
		t.Fatal("cookie จาก login ควรแปลงกลับเป็น userID ได้")
	}
}

// Users → คืนรายชื่อ user ที่สมัคร (id+name+role) สำหรับ selector
func TestUsersEndpointListsRegistered(t *testing.T) {
	store := service.NewStore(repository.NewMemUsers())
	h := NewHandler(store, service.NewSessions())
	store.Register(domain.RegisterPayload{Email: "becket@x.com", Role: domain.RoleDeveloper, Password: "secret"})
	store.Register(domain.RegisterPayload{Email: "minyjae@y.com", Role: domain.RolePM, Password: "secret"})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	h.Users(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ควร 200 ได้ %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"becket"`) || !strings.Contains(body, `"minyjae"`) {
		t.Fatalf("ควรมีชื่อย่อทั้งสองคน ได้ %s", body)
	}
	if strings.Contains(body, "@") || strings.Contains(body, "PassHash") {
		t.Fatalf("ต้องไม่หลุด email เต็ม/hash ได้ %s", body)
	}
}
