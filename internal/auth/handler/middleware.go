package handler

import (
	"context"
	"net/http"

	"github/minyjae/catice/internal/auth/domain"
)

// ctxKey เป็น type ส่วนตัว → กันชนกับ key ของ package อื่นใน context
type ctxKey int

const userCtxKey ctxKey = 0

// RequireAuth = middleware: เช็ค JWT → ถ้า login แล้ว ฝัง User ลง context แล้วเรียก handler ถัดไป
// ถ้ายังไม่ login → ตอบ 401 เลย (handler ปลายทางไม่ต้องเช็คซ้ำ)
//
// ใช้: http.HandleFunc("/me", authH.RequireAuth(authH.Me))
func (h *AuthHandler) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := h.UserFromRequest(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "ยังไม่ได้เข้าสู่ระบบ"})
			return
		}
		next(w, WithUser(r, u)) // ส่ง request ที่ "ติด User" ไว้ใน context
	}
}

// UserOf อ่าน User ที่ RequireAuth ฝังไว้ใน context (ใช้ใน handler ที่อยู่หลัง RequireAuth)
func UserOf(r *http.Request) (domain.User, bool) {
	u, ok := r.Context().Value(userCtxKey).(domain.User)
	return u, ok
}

// WithUser ฝัง User ลง context แบบเดียวกับที่ RequireAuth ทำ — export ไว้ให้ test ยิง handler
// ตรง ๆ ด้วย request ที่จำลองผู้ใช้ login แล้ว โดยไม่ต้องพึ่ง JWT จริง
func WithUser(r *http.Request, u domain.User) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userCtxKey, u))
}
