package handler

import (
	"encoding/json"
	"net/http"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/service"
)

// UserManagementHandler รวม HTTP endpoint สำหรับ HR จัดการข้อมูลพนักงาน
type UserManagementHandler struct {
	store *service.Store
}

func NewUserManagementHandler(store *service.Store) *UserManagementHandler {
	return &UserManagementHandler{store: store}
}

// toHRView แปลง domain.User → domain.HRUserView (รวม sensitive fields)
func toHRView(u domain.User) domain.HRUserView {
	return domain.HRUserView{
		ID:        u.ID,
		Email:     u.Email,
		Name:      nameFromEmail(u.Email),
		Role:      u.Role,
		ManagerID: u.ManagerID,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Phone:     u.Phone,
		BirthDate: u.BirthDate,
		Address:   u.Address,
		Salary:    u.Salary,
		StartDate: u.StartDate,
	}
}

// ListAll : GET /hr/users — รายชื่อพนักงานทั้งหมดพร้อมข้อมูลเต็ม (HR เท่านั้น)
func (h *UserManagementHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r)
	if caller.Role != domain.RoleHR {
		writeJSON(w, http.StatusForbidden, map[string]string{"message": domain.ErrForbidden.Error()})
		return
	}
	users := h.store.ListUsers()
	views := make([]domain.HRUserView, 0, len(users))
	for _, u := range users {
		views = append(views, toHRView(u))
	}
	writeJSON(w, http.StatusOK, views)
}

// GetOne : GET /hr/users/{id} — ข้อมูลพนักงานคนเดียวเต็ม ๆ (HR เท่านั้น)
func (h *UserManagementHandler) GetOne(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r)
	if caller.Role != domain.RoleHR {
		writeJSON(w, http.StatusForbidden, map[string]string{"message": domain.ErrForbidden.Error()})
		return
	}
	u, ok := h.store.GetByID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": domain.ErrUserNotFound.Error()})
		return
	}
	writeJSON(w, http.StatusOK, toHRView(u))
}

// UpdateProfile : PATCH /hr/users/{id} — แก้ไขข้อมูลส่วนตัวพนักงาน (HR เท่านั้น)
func (h *UserManagementHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r)
	var p domain.UpdateProfilePayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "รูปแบบข้อมูลไม่ถูกต้อง"})
		return
	}
	u, err := h.store.UpdateProfile(caller.Role, r.PathValue("id"), p)
	if err != nil {
		writeJSON(w, statusForErr(err), map[string]string{"message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, toHRView(u))
}

// ChangeRole : PATCH /hr/users/{id}/role — เปลี่ยนตำแหน่งพนักงาน (HR เท่านั้น)
func (h *UserManagementHandler) ChangeRole(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r)
	var p domain.ChangeRolePayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "รูปแบบข้อมูลไม่ถูกต้อง"})
		return
	}
	u, err := h.store.ChangeRole(caller.Role, r.PathValue("id"), p.Role)
	if err != nil {
		writeJSON(w, statusForErr(err), map[string]string{"message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, toHRView(u))
}

// DeleteUser : DELETE /hr/users/{id} — soft-delete พนักงาน (ลาออก/เลิกจ้าง) — HR เท่านั้น
func (h *UserManagementHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserOf(r)
	if err := h.store.DeleteUser(caller.Role, r.PathValue("id")); err != nil {
		writeJSON(w, statusForErr(err), map[string]string{"message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ลบพนักงานออกจากระบบแล้ว"})
}
