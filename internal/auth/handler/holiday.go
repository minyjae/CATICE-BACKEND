package handler

import (
	"encoding/json"
	"net/http"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/service"
)

// HolidayHandler รวม HTTP endpoint ของวันหยุดบริษัท (list/create/delete)
type HolidayHandler struct {
	store *service.HolidayStore
}

func NewHolidayHandler(store *service.HolidayStore) *HolidayHandler {
	return &HolidayHandler{store: store}
}

// List : GET /holidays — วันหยุดทั้งหมด (login แล้วดูได้ทุกคน)
func (h *HolidayHandler) List(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.List())
}

// Create : POST /holidays  body {name, date} — เฉพาะ HR
func (h *HolidayHandler) Create(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r)
	var p domain.CreateHolidayPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "รูปแบบข้อมูลไม่ถูกต้อง"})
		return
	}

	holiday, err := h.store.Create(caller.Role, caller.ID, p)
	if err != nil {
		writeJSON(w, statusForErr(err), map[string]string{"message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, holiday)
}

// Delete : DELETE /holidays/{id} — เฉพาะ HR
func (h *HolidayHandler) Delete(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r)
	if err := h.store.Delete(caller.Role, r.PathValue("id")); err != nil {
		writeJSON(w, statusForErr(err), map[string]string{"message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ลบวันหยุดแล้ว"})
}
