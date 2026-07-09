// Package handler = ชั้น HTTP ของ auth (รับ request → เรียก service → เขียน response)
// ไม่มี business logic เอง — มอบให้ service ทำ
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/service"
)

// AuthHandler รวม HTTP endpoint ของ user/auth (register/login/logout/me/users)
type AuthHandler struct {
	store  *service.Store
	tokens *service.Tokens
}

func NewAuthHandler(store *service.Store, tokens *service.Tokens) *AuthHandler {
	return &AuthHandler{store: store, tokens: tokens}
}

// Register : POST /register  body {email, role, password}
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
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

	// สมัครเสร็จ login ให้เลย → ออก JWT คืนไปใน body
	writeJSON(w, http.StatusOK, domain.RegisterResponse{Message: "สมัครสำเร็จ", Token: h.tokens.Create(u.ID)})
}

// Login : POST /login  body {email, password}
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
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

	writeJSON(w, http.StatusOK, domain.LoginResponse{Message: "เข้าสู่ระบบสำเร็จ", Role: u.Role, Token: h.tokens.Create(u.ID)})
}

// Logout : POST /logout — JWT เป็น stateless → server ไม่มี state ให้ลบ
// การออกจากระบบจริง = ฝั่ง client ทิ้ง token ทิ้งเอง (ลบจาก localStorage ฯลฯ)
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, domain.LoginResponse{Message: "ออกจากระบบแล้ว"})
}

// UserIDFromRequest แกะ JWT จาก request → userID (เอาไว้ให้ ServeWs ฝั่ง ws เรียกใช้)
func (h *AuthHandler) UserIDFromRequest(r *http.Request) (string, bool) {
	token := tokenFromRequest(r)
	if token == "" {
		return "", false
	}
	return h.tokens.UserID(token)
}

// tokenFromRequest ดึง JWT ออกจาก request
//   - ปกติ: header "Authorization: Bearer <token>"
//   - fallback: query "?token=<token>" — เผื่อ WebSocket ที่ตั้ง custom header ตอน handshake ไม่ได้
func tokenFromRequest(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		if t, ok := strings.CutPrefix(h, "Bearer "); ok {
			return strings.TrimSpace(t)
		}
	}
	return r.URL.Query().Get("token")
}

// UserFromRequest แกะ JWT → User เต็ม ๆ (token → userID → user)
func (h *AuthHandler) UserFromRequest(r *http.Request) (domain.User, bool) {
	uid, ok := h.UserIDFromRequest(r)
	if !ok {
		return domain.User{}, false
	}
	return h.store.GetByID(uid)
}

// Me : GET /me — คืนข้อมูล user ที่ login อยู่ (frontend ใช้เช็คว่า login ไหม + เอา email/role)
// ต้องอยู่หลัง RequireAuth → อ่าน User จาก context ได้เลย (PassHash มี json:"-" → ไม่หลุด)
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	u, _ := UserOf(r) // RequireAuth การันตีว่ามีแล้ว
	writeJSON(w, http.StatusOK, u)
}

// Users : GET /users — รายชื่อ user ที่สมัครทั้งหมด (id + ชื่อย่อ + role)
// frontend เอาไปทำ selector "มอบหมายให้ใคร" บน Kanban board
// อยู่หลัง RequireAuth (ต้อง login ก่อน) — คืนเฉพาะข้อมูลที่ปลอดภัยจะให้คนอื่นเห็น
func (h *AuthHandler) Users(w http.ResponseWriter, r *http.Request) {
	users := h.store.ListUsers()
	out := make([]domain.PublicUser, 0, len(users))
	for _, u := range users {
		out = append(out, domain.PublicUser{ID: u.ID, Name: nameFromEmail(u.Email), Role: u.Role, ManagerID: u.ManagerID})
	}
	writeJSON(w, http.StatusOK, out)
}

// SetManager : PATCH /users/{id}/manager  body {manager_id} — ตั้ง/เคลียร์หัวหน้าของ user (เฉพาะ HR)
func (h *AuthHandler) SetManager(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r) // RequireAuth การันตีว่ามีแล้ว
	var p domain.SetManagerPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "รูปแบบข้อมูลไม่ถูกต้อง"})
		return
	}

	u, err := h.store.SetManager(caller.Role, r.PathValue("id"), p.ManagerID)
	if err != nil {
		writeJSON(w, statusForErr(err), map[string]string{"message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, domain.PublicUser{ID: u.ID, Name: nameFromEmail(u.Email), Role: u.Role, ManagerID: u.ManagerID})
}

// ---------- helpers ----------

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
	case errors.Is(err, domain.ErrMissingFields),
		errors.Is(err, domain.ErrInvalidRole),
		errors.Is(err, domain.ErrEmptyHolidayName),
		errors.Is(err, domain.ErrEmptyLeaveType),
		errors.Is(err, domain.ErrInvalidDateRange),
		errors.Is(err, domain.ErrEmptyDiaryContent):
		return http.StatusBadRequest // 400
	case errors.Is(err, domain.ErrForbidden), errors.Is(err, domain.ErrNotApprover):
		return http.StatusForbidden // 403
	case errors.Is(err, domain.ErrUserNotFound), errors.Is(err, domain.ErrRequestNotFound):
		return http.StatusNotFound // 404
	case errors.Is(err, domain.ErrLeaveQuotaExceeded),
		errors.Is(err, domain.ErrWFHWeeklyExceeded),
		errors.Is(err, domain.ErrWFHMonthlyExceeded):
		return http.StatusUnprocessableEntity // 422
	case errors.Is(err, domain.ErrInvalidPolicy):
		return http.StatusBadRequest // 400
	case errors.Is(err, domain.ErrNoApprover), errors.Is(err, domain.ErrNotPending):
		return http.StatusConflict // 409
	default:
		return http.StatusInternalServerError // 500
	}
}
