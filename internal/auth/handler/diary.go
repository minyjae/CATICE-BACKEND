package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/service"
)

// DiaryHandler รวม HTTP endpoint ของ daily diary (บันทึก/ดูของตัวเอง/ดูของลูกทีม)
type DiaryHandler struct {
	store *service.DiaryStore
}

func NewDiaryHandler(store *service.DiaryStore) *DiaryHandler {
	return &DiaryHandler{store: store}
}

// Upsert : POST /diary  body {date, content} — บันทึกทับของวันนั้น (UserID มาจาก JWT)
func (h *DiaryHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r)
	var p domain.UpsertDiaryPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "รูปแบบข้อมูลไม่ถูกต้อง"})
		return
	}

	d, err := h.store.Upsert(caller.ID, p)
	if err != nil {
		writeJSON(w, statusForErr(err), map[string]string{"message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// Mine : GET /diary/mine?limit= — diary ของตัวเอง เรียงล่าสุดก่อน
func (h *DiaryHandler) Mine(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r)
	limit := 30
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = v
	}
	writeJSON(w, http.StatusOK, h.store.ListMine(caller.ID, limit))
}

// OfUser : GET /diary?user_id=&date= — ให้ HR/manager เข้าดู diary ของลูกทีมวันใดวันหนึ่ง
func (h *DiaryHandler) OfUser(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r)
	targetUserID := r.URL.Query().Get("user_id")
	date := r.URL.Query().Get("date")

	d, ok, err := h.store.OfUser(caller.ID, caller.Role, targetUserID, date)
	if err != nil {
		writeJSON(w, statusForErr(err), map[string]string{"message": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "ไม่พบ diary ของวันนี้"})
		return
	}
	writeJSON(w, http.StatusOK, d)
}
