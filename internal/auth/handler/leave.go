package handler

import (
	"encoding/json"
	"net/http"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/service"
)

// LeaveHandler รวม HTTP endpoint ของคำขอลา (สร้าง/ดูของตัวเอง/ดูที่รออนุมัติ/อนุมัติ/ปฏิเสธ)
type LeaveHandler struct {
	store *service.LeaveStore
}

func NewLeaveHandler(store *service.LeaveStore) *LeaveHandler {
	return &LeaveHandler{store: store}
}

// Create : POST /leaves  body {type, start_date, end_date, reason} — UserID มาจาก JWT
func (h *LeaveHandler) Create(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r)
	var p domain.CreateLeavePayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "รูปแบบข้อมูลไม่ถูกต้อง"})
		return
	}

	l, err := h.store.Create(caller.ID, p)
	if err != nil {
		writeJSON(w, statusForErr(err), map[string]string{"message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, l)
}

// Mine : GET /leaves/mine — คำขอลาของตัวเองทั้งหมด
func (h *LeaveHandler) Mine(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r)
	writeJSON(w, http.StatusOK, h.store.ListMine(caller.ID))
}

// Pending : GET /leaves/pending — คำขอลาที่ caller เป็นผู้อนุมัติและยังไม่ตัดสินใจ
func (h *LeaveHandler) Pending(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r)
	writeJSON(w, http.StatusOK, h.store.ListPending(caller.ID))
}

// Approve : POST /leaves/{id}/approve
func (h *LeaveHandler) Approve(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, true)
}

// Reject : POST /leaves/{id}/reject
func (h *LeaveHandler) Reject(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, false)
}

func (h *LeaveHandler) decide(w http.ResponseWriter, r *http.Request, approve bool) {
	caller, _ := UserOf(r)
	l, err := h.store.Decide(caller.ID, r.PathValue("id"), approve)
	if err != nil {
		writeJSON(w, statusForErr(err), map[string]string{"message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, l)
}
