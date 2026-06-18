// Package controller = ชั้น HTTP ของ auth (รับ request → เรียก service → เขียน response/cookie)
// ไม่มี business logic เอง — มอบให้ service ทำ
package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/service"
)

// CookieName คือชื่อ cookie ที่ใช้เก็บ session token (ServeWs ฝั่ง ws จะอ่านชื่อนี้)
const CookieName = "session"

// Handler รวม HTTP endpoint ของ auth (register/login/logout/me/users)
type Handler struct {
	store    *service.Store
	sessions *service.Sessions
}

func NewHandler(store *service.Store, sessions *service.Sessions) *Handler {
	return &Handler{store: store, sessions: sessions}
}

// Register : POST /register  body {email, role, password}
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var p domain.RegisterPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, domain.RegisterResponse{Message: "รูปแบบข้อมูลไม่ถูกต้อง"})
		return
	}

	u, err := h.store.Register(p)
	if err != nil {
		writeJSON(w, statusForErr(err), domain.RegisterResponse{Message: err.Error()})
		return
	}

	// สมัครเสร็จ login ให้เลย → ออก session + set cookie
	h.setSession(w, u.ID)
	writeJSON(w, http.StatusOK, domain.RegisterResponse{Message: "สมัครสำเร็จ"})
}

// Login : POST /login  body {email, password}
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var p domain.LoginPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, domain.LoginResponse{Message: "รูปแบบข้อมูลไม่ถูกต้อง"})
		return
	}

	u, err := h.store.Login(p)
	if err != nil {
		writeJSON(w, statusForErr(err), domain.LoginResponse{Message: err.Error()})
		return
	}

	h.setSession(w, u.ID)
	writeJSON(w, http.StatusOK, domain.LoginResponse{Message: "เข้าสู่ระบบสำเร็จ", Role: u.Role})
}

// Logout : POST /logout — ลบ session + เคลียร์ cookie
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(CookieName); err == nil {
		h.sessions.Destroy(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	writeJSON(w, http.StatusOK, domain.LoginResponse{Message: "ออกจากระบบแล้ว"})
}

// UserIDFromRequest อ่าน cookie → userID (เอาไว้ให้ ServeWs ฝั่ง ws เรียกใช้)
func (h *Handler) UserIDFromRequest(r *http.Request) (string, bool) {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return "", false
	}
	return h.sessions.UserID(c.Value)
}

// UserFromRequest อ่าน cookie → User เต็ม ๆ (cookie → token → userID → user)
func (h *Handler) UserFromRequest(r *http.Request) (domain.User, bool) {
	uid, ok := h.UserIDFromRequest(r)
	if !ok {
		return domain.User{}, false
	}
	return h.store.GetByID(uid)
}

// Me : GET /me — คืนข้อมูล user ที่ login อยู่ (frontend ใช้เช็คว่า login ไหม + เอา email/role)
// ต้องอยู่หลัง RequireAuth → อ่าน User จาก context ได้เลย (PassHash มี json:"-" → ไม่หลุด)
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	u, _ := UserOf(r) // RequireAuth การันตีว่ามีแล้ว
	writeJSON(w, http.StatusOK, u)
}

// Users : GET /users — รายชื่อ user ที่สมัครทั้งหมด (id + ชื่อย่อ + role)
// frontend เอาไปทำ selector "มอบหมายให้ใคร" บน Kanban board
// อยู่หลัง RequireAuth (ต้อง login ก่อน) — คืนเฉพาะข้อมูลที่ปลอดภัยจะให้คนอื่นเห็น
func (h *Handler) Users(w http.ResponseWriter, r *http.Request) {
	users := h.store.ListUsers()
	out := make([]domain.PublicUser, 0, len(users))
	for _, u := range users {
		out = append(out, domain.PublicUser{ID: u.ID, Name: nameFromEmail(u.Email), Role: u.Role})
	}
	writeJSON(w, http.StatusOK, out)
}

// ---------- helpers ----------

// setSession ออก token ใหม่ → ฝังเป็น cookie
func (h *Handler) setSession(w http.ResponseWriter, userID string) {
	token := h.sessions.Create(userID)
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true, // JS อ่านไม่ได้ → กัน XSS ขโมย token
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60, // อยู่ได้ 7 วัน
		// Secure: true,            // ⚠️ เปิดตอน deploy ผ่าน https
	})
}

// nameFromEmail ตัดส่วนหน้า "@" มาเป็นชื่อแสดงผล (เช่น becket@x.com → becket)
func nameFromEmail(email string) string {
	if i := strings.IndexByte(email, '@'); i >= 0 {
		return email[:i]
	}
	return email
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// statusForErr แปลง error ของ service → HTTP status code
func statusForErr(err error) int {
	switch {
	case errors.Is(err, domain.ErrEmailTaken):
		return http.StatusConflict // 409
	case errors.Is(err, domain.ErrBadCredentials):
		return http.StatusUnauthorized // 401
	case errors.Is(err, domain.ErrMissingFields), errors.Is(err, domain.ErrInvalidRole):
		return http.StatusBadRequest // 400
	default:
		return http.StatusInternalServerError // 500
	}
}
