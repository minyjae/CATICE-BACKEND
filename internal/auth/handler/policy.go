package handler

import (
	"encoding/json"
	"net/http"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/service"
)

// PolicyHandler รวม HTTP endpoint ของ leave/WFH policy (ดู/อัปเดต)
type PolicyHandler struct {
	store *service.PolicyStore
}

func NewPolicyHandler(store *service.PolicyStore) *PolicyHandler {
	return &PolicyHandler{store: store}
}

// Get : GET /policy — ดู policy ปัจจุบัน (ทุกคนที่ login แล้วดูได้)
func (h *PolicyHandler) Get(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.Get())
}

// Update : PUT /policy  body {vacation_days_per_year, sick_days_per_year, ...} — เฉพาะ HR
func (h *PolicyHandler) Update(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r)
	var p domain.LeavePolicy
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "รูปแบบข้อมูลไม่ถูกต้อง"})
		return
	}
	updated, err := h.store.Update(caller.Role, p)
	if err != nil {
		writeJSON(w, statusForErr(err), map[string]string{"message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
