package controller

import (
	"context"
	"net/http"

	"github/minyjae/catice/internal/auth/domain"
)

// ctxKey เป็น type ส่วนตัว → กันชนกับ key ของ package อื่นใน context
type ctxKey int

const userCtxKey ctxKey = 0

// RequireAuth = middleware: เช็ค cookie → ถ้า login แล้ว ฝัง User ลง context แล้วเรียก handler ถัดไป
// ถ้ายังไม่ login → ตอบ 401 เลย (handler ปลายทางไม่ต้องเช็คซ้ำ)
//
// ใช้: http.HandleFunc("/me", authH.RequireAuth(authH.Me))
func (h *Handler) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := h.UserFromRequest(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "ยังไม่ได้เข้าสู่ระบบ"})
			return
		}
		ctx := context.WithValue(r.Context(), userCtxKey, u)
		next(w, r.WithContext(ctx)) // ส่ง request ที่ "ติด User" ไว้ใน context
	}
}

// UserOf อ่าน User ที่ RequireAuth ฝังไว้ใน context (ใช้ใน handler ที่อยู่หลัง RequireAuth)
func UserOf(r *http.Request) (domain.User, bool) {
	u, ok := r.Context().Value(userCtxKey).(domain.User)
	return u, ok
}
