package handler

import (
	"encoding/json"
	"net/http"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/service"
)

// WFHHandler รวม HTTP endpoint ของคำขอ work-from-home (โครงเดียวกับ LeaveHandler)
type WFHHandler struct {
	store *service.WFHStore
}

func NewWFHHandler(store *service.WFHStore) *WFHHandler {
	return &WFHHandler{store: store}
}

// Create : POST /wfh  body {date, reason} — UserID มาจาก JWT
func (h *WFHHandler) Create(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r)
	var p domain.CreateWFHPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "รูปแบบข้อมูลไม่ถูกต้อง"})
		return
	}

	req, err := h.store.Create(caller.ID, p)
	if err != nil {
		writeJSON(w, statusForErr(err), map[string]string{"message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, req)
}

// Mine : GET /wfh/mine — คำขอ WFH ของตัวเองทั้งหมด
func (h *WFHHandler) Mine(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r)
	writeJSON(w, http.StatusOK, h.store.ListMine(caller.ID))
}

// Pending : GET /wfh/pending — คำขอ WFH ที่ caller เป็นผู้อนุมัติและยังไม่ตัดสินใจ
func (h *WFHHandler) Pending(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r)
	writeJSON(w, http.StatusOK, h.store.ListPending(caller.ID))
}

// Approve : POST /wfh/{id}/approve
func (h *WFHHandler) Approve(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, true)
}

// Reject : POST /wfh/{id}/reject
func (h *WFHHandler) Reject(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, false)
}

func (h *WFHHandler) decide(w http.ResponseWriter, r *http.Request, approve bool) {
	caller, _ := UserOf(r)
	req, err := h.store.Decide(caller.ID, r.PathValue("id"), approve)
	if err != nil {
		writeJSON(w, statusForErr(err), map[string]string{"message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, req)
}
