package leave_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/handler"
	"github/minyjae/catice/internal/auth/service"
	"github/minyjae/catice/test/fakes"
)

func newLeaveHandler() (*handler.LeaveHandler, *fakes.Users) {
	users := fakes.NewUsers()
	policy := service.NewPolicyStore(fakes.NewPolicy())
	store := service.NewLeaveStore(fakes.NewLeaves(), users, policy)
	return handler.NewLeaveHandler(store), users
}

func createLeave(t *testing.T, h *handler.LeaveHandler, caller domain.User, body string) domain.LeaveRequest {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/leaves", strings.NewReader(body))
	req = handler.WithUser(req, caller)
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("setup Create ล้มเหลว: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got domain.LeaveRequest
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	return got
}

// เทส: LeaveHandler.Create ทาง happy path
// input: HTTP POST body ที่ถูกต้องครบ (type/start_date/end_date/reason) โดยผู้เรียกมี ManagerID=mgr1
// aspect: status 200, response body มี status="pending" และ approver_id="mgr1"
func TestLeaveHandlerCreate(t *testing.T) {
	h, users := newLeaveHandler()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	caller := domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"}
	users.Seed(caller)

	got := createLeave(t, h, caller, `{"type":"vacation","start_date":"2026-08-01","end_date":"2026-08-03","reason":"trip"}`)
	if got.Status != domain.StatusPending {
		t.Fatalf("status ควรเป็น pending ได้ %q", got.Status)
	}
	if got.ApproverID != "mgr1" {
		t.Fatalf("approver_id ควรเป็น mgr1 ได้ %q", got.ApproverID)
	}
}

// เทส: LeaveHandler.Create ปฏิเสธ request body ที่ไม่ใช่ JSON ที่ถูกต้อง
// input: body เป็น string "{bad json" (JSON เสีย)
// aspect: status ต้องเป็น 400
func TestLeaveHandlerCreateBadBody(t *testing.T) {
	h, _ := newLeaveHandler()
	req := httptest.NewRequest(http.MethodPost, "/leaves", strings.NewReader("{bad json"))
	req = handler.WithUser(req, domain.User{ID: "emp1"})
	rec := httptest.NewRecorder()

	h.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// เทส: error จาก service (ErrEmptyLeaveType) ไหลผ่าน statusForErr มาเป็น HTTP status ที่ถูกต้อง
// input: body ที่ type เป็นค่าว่าง ""
// aspect: status ต้องเป็น 400
func TestLeaveHandlerCreateInvalidType(t *testing.T) {
	h, users := newLeaveHandler()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	caller := domain.User{ID: "emp1", ManagerID: "mgr1"}
	users.Seed(caller)

	req := httptest.NewRequest(http.MethodPost, "/leaves", strings.NewReader(`{"type":"","start_date":"2026-08-01","end_date":"2026-08-01"}`))
	req = handler.WithUser(req, caller)
	rec := httptest.NewRecorder()

	h.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
}

// เทส: LeaveHandler.Mine คืนเฉพาะคำขอของผู้เรียก (caller) เท่านั้น
// input: สร้างคำขอลาให้ emp1 และ emp2 คนละ 1 ใบ แล้วเรียก Mine ในฐานะ emp1
// aspect: response ต้องมี 1 รายการ และเป็นของ emp1 เท่านั้น
func TestLeaveHandlerMine(t *testing.T) {
	h, users := newLeaveHandler()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	emp1 := domain.User{ID: "emp1", ManagerID: "mgr1"}
	emp2 := domain.User{ID: "emp2", ManagerID: "mgr1"}
	users.Seed(emp1)
	users.Seed(emp2)

	createLeave(t, h, emp1, `{"type":"vacation","start_date":"2026-08-01","end_date":"2026-08-01"}`)
	createLeave(t, h, emp2, `{"type":"vacation","start_date":"2026-08-02","end_date":"2026-08-02"}`)

	req := httptest.NewRequest(http.MethodGet, "/leaves/mine", nil)
	req = handler.WithUser(req, emp1)
	rec := httptest.NewRecorder()
	h.Mine(rec, req)

	var got []domain.LeaveRequest
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(got) != 1 || got[0].UserID != "emp1" {
		t.Fatalf("Mine ผิด: %+v", got)
	}
}

// เทส: LeaveHandler.Pending คืนเฉพาะคำขอที่ caller เป็น approver และยัง pending
// input: emp1 ยื่นคำขอ 2 ใบให้ mgr1, mgr1 approve ไปแล้ว 1 ใบ (l2) แล้วเรียก Pending ในฐานะ mgr1
// aspect: response ต้องเหลือแค่ l1 (ที่ยัง pending)
func TestLeaveHandlerPending(t *testing.T) {
	h, users := newLeaveHandler()
	mgr := domain.User{ID: "mgr1", Role: domain.RoleDeveloper}
	emp := domain.User{ID: "emp1", ManagerID: "mgr1"}
	users.Seed(mgr)
	users.Seed(emp)

	l1 := createLeave(t, h, emp, `{"type":"vacation","start_date":"2026-08-01","end_date":"2026-08-01"}`)
	l2 := createLeave(t, h, emp, `{"type":"sick","start_date":"2026-08-05","end_date":"2026-08-05"}`)

	approveReq := httptest.NewRequest(http.MethodPost, "/leaves/"+l2.ID+"/approve", nil)
	approveReq.SetPathValue("id", l2.ID)
	approveReq = handler.WithUser(approveReq, mgr)
	h.Approve(httptest.NewRecorder(), approveReq)

	req := httptest.NewRequest(http.MethodGet, "/leaves/pending", nil)
	req = handler.WithUser(req, mgr)
	rec := httptest.NewRecorder()
	h.Pending(rec, req)

	var got []domain.LeaveRequest
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(got) != 1 || got[0].ID != l1.ID {
		t.Fatalf("Pending ควรเหลือแค่ l1 ได้: %+v", got)
	}
}

// เทส: LeaveHandler.Approve โดย approver ตัวจริง
// input: คำขอลาที่ ApproverID=mgr1 + เรียก Approve พร้อม path value id=คำขอนั้น ในฐานะ mgr1
// aspect: status 200, response body มี status="approved"
func TestLeaveHandlerApprove(t *testing.T) {
	h, users := newLeaveHandler()
	mgr := domain.User{ID: "mgr1", Role: domain.RoleDeveloper}
	emp := domain.User{ID: "emp1", ManagerID: "mgr1"}
	users.Seed(mgr)
	users.Seed(emp)

	created := createLeave(t, h, emp, `{"type":"vacation","start_date":"2026-08-01","end_date":"2026-08-01"}`)

	req := httptest.NewRequest(http.MethodPost, "/leaves/"+created.ID+"/approve", nil)
	req.SetPathValue("id", created.ID)
	req = handler.WithUser(req, mgr)
	rec := httptest.NewRecorder()
	h.Approve(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var got domain.LeaveRequest
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.Status != domain.StatusApproved {
		t.Fatalf("status ควรเป็น approved ได้ %q", got.Status)
	}
}

// เทส: LeaveHandler.Approve ต้องปฏิเสธคนที่ไม่ใช่ approver ของคำขอนั้น
// input: คำขอลาที่ ApproverID=mgr1 แต่เรียก Approve ในฐานะ "someone-else"
// aspect: status ต้องเป็น 403
func TestLeaveHandlerApproveWrongApprover(t *testing.T) {
	h, users := newLeaveHandler()
	mgr := domain.User{ID: "mgr1", Role: domain.RoleDeveloper}
	emp := domain.User{ID: "emp1", ManagerID: "mgr1"}
	users.Seed(mgr)
	users.Seed(emp)

	created := createLeave(t, h, emp, `{"type":"vacation","start_date":"2026-08-01","end_date":"2026-08-01"}`)

	req := httptest.NewRequest(http.MethodPost, "/leaves/"+created.ID+"/approve", nil)
	req.SetPathValue("id", created.ID)
	req = handler.WithUser(req, domain.User{ID: "someone-else"})
	rec := httptest.NewRecorder()
	h.Approve(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

// เทส: LeaveHandler.Reject โดย approver ตัวจริง
// input: คำขอลาที่ ApproverID=mgr1 + เรียก Reject ในฐานะ mgr1
// aspect: response body ต้องมี status="rejected"
func TestLeaveHandlerReject(t *testing.T) {
	h, users := newLeaveHandler()
	mgr := domain.User{ID: "mgr1", Role: domain.RoleDeveloper}
	emp := domain.User{ID: "emp1", ManagerID: "mgr1"}
	users.Seed(mgr)
	users.Seed(emp)

	created := createLeave(t, h, emp, `{"type":"vacation","start_date":"2026-08-01","end_date":"2026-08-01"}`)

	req := httptest.NewRequest(http.MethodPost, "/leaves/"+created.ID+"/reject", nil)
	req.SetPathValue("id", created.ID)
	req = handler.WithUser(req, mgr)
	rec := httptest.NewRecorder()
	h.Reject(rec, req)

	var got domain.LeaveRequest
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.Status != domain.StatusRejected {
		t.Fatalf("status ควรเป็น rejected ได้ %q", got.Status)
	}
}

// เทส: LeaveHandler.Approve กับ request id ที่ไม่มีอยู่จริง
// input: path value id="nope" ที่ไม่เคยถูกสร้างไว้เลย
// aspect: status ต้องเป็น 404
func TestLeaveHandlerApproveNotFound(t *testing.T) {
	h, users := newLeaveHandler()
	mgr := domain.User{ID: "mgr1", Role: domain.RoleDeveloper}
	users.Seed(mgr)

	req := httptest.NewRequest(http.MethodPost, "/leaves/nope/approve", nil)
	req.SetPathValue("id", "nope")
	req = handler.WithUser(req, mgr)
	rec := httptest.NewRecorder()
	h.Approve(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
