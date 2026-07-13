package wfh_test

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

func newWFHHandler() (*handler.WFHHandler, *fakes.Users) {
	users := fakes.NewUsers()
	policy := service.NewPolicyStore(fakes.NewPolicy())
	store := service.NewWFHStore(fakes.NewWFH(), users, policy)
	return handler.NewWFHHandler(store), users
}

func createWFH(t *testing.T, h *handler.WFHHandler, caller domain.User, body string) domain.WFHRequest {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/wfh", strings.NewReader(body))
	req = handler.WithUser(req, caller)
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("setup Create ล้มเหลว: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got domain.WFHRequest
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	return got
}

// เทส: WFHHandler.Create ทาง happy path
// input: HTTP POST body ที่ถูกต้อง (date/reason) โดยผู้เรียกมี ManagerID=mgr1
// aspect: response body มี status="pending" และ approver_id="mgr1"
func TestWFHHandlerCreate(t *testing.T) {
	h, users := newWFHHandler()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	caller := domain.User{ID: "emp1", ManagerID: "mgr1"}
	users.Seed(caller)

	got := createWFH(t, h, caller, `{"date":"2026-08-10","reason":"internet install"}`)
	if got.Status != domain.StatusPending {
		t.Fatalf("status ควรเป็น pending ได้ %q", got.Status)
	}
	if got.ApproverID != "mgr1" {
		t.Fatalf("approver_id ควรเป็น mgr1 ได้ %q", got.ApproverID)
	}
}

// เทส: WFHHandler.Create ปฏิเสธ request body ที่ไม่ใช่ JSON ที่ถูกต้อง
// input: body เป็น string "{bad json" (JSON เสีย)
// aspect: status ต้องเป็น 400
func TestWFHHandlerCreateBadBody(t *testing.T) {
	h, _ := newWFHHandler()
	req := httptest.NewRequest(http.MethodPost, "/wfh", strings.NewReader("{bad json"))
	req = handler.WithUser(req, domain.User{ID: "emp1"})
	rec := httptest.NewRecorder()

	h.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// เทส: error จาก service (ErrInvalidDateRange) ไหลผ่าน statusForErr มาเป็น HTTP status ที่ถูกต้อง
// input: body ที่ date เป็นค่าว่าง ""
// aspect: status ต้องเป็น 400
func TestWFHHandlerCreateEmptyDate(t *testing.T) {
	h, users := newWFHHandler()
	caller := domain.User{ID: "emp1", ManagerID: "mgr1"}
	users.Seed(caller)

	req := httptest.NewRequest(http.MethodPost, "/wfh", strings.NewReader(`{"date":""}`))
	req = handler.WithUser(req, caller)
	rec := httptest.NewRecorder()

	h.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
}

// เทส: WFHHandler.Mine คืนเฉพาะคำขอของผู้เรียก (caller) เท่านั้น
// input: สร้างคำขอ WFH ให้ emp1 และ emp2 คนละ 1 ใบ แล้วเรียก Mine ในฐานะ emp1
// aspect: response ต้องมี 1 รายการ และเป็นของ emp1 เท่านั้น
func TestWFHHandlerMine(t *testing.T) {
	h, users := newWFHHandler()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	emp1 := domain.User{ID: "emp1", ManagerID: "mgr1"}
	emp2 := domain.User{ID: "emp2", ManagerID: "mgr1"}
	users.Seed(emp1)
	users.Seed(emp2)

	createWFH(t, h, emp1, `{"date":"2026-08-10"}`)
	createWFH(t, h, emp2, `{"date":"2026-08-11"}`)

	req := httptest.NewRequest(http.MethodGet, "/wfh/mine", nil)
	req = handler.WithUser(req, emp1)
	rec := httptest.NewRecorder()
	h.Mine(rec, req)

	var got []domain.WFHRequest
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(got) != 1 || got[0].UserID != "emp1" {
		t.Fatalf("Mine ผิด: %+v", got)
	}
}

// เทส: WFHHandler.Pending คืนเฉพาะคำขอที่ caller เป็น approver และยัง pending
// input: emp1 ยื่นคำขอ WFH 2 ใบให้ mgr1, mgr1 approve ไปแล้ว 1 ใบ (w2) แล้วเรียก Pending ในฐานะ mgr1
// aspect: response ต้องเหลือแค่ w1 (ที่ยัง pending)
func TestWFHHandlerPending(t *testing.T) {
	h, users := newWFHHandler()
	mgr := domain.User{ID: "mgr1", Role: domain.RoleDeveloper}
	emp := domain.User{ID: "emp1", ManagerID: "mgr1"}
	users.Seed(mgr)
	users.Seed(emp)

	w1 := createWFH(t, h, emp, `{"date":"2026-08-10"}`)
	w2 := createWFH(t, h, emp, `{"date":"2026-08-11"}`)

	approveReq := httptest.NewRequest(http.MethodPost, "/wfh/"+w2.ID+"/approve", nil)
	approveReq.SetPathValue("id", w2.ID)
	approveReq = handler.WithUser(approveReq, mgr)
	h.Approve(httptest.NewRecorder(), approveReq)

	req := httptest.NewRequest(http.MethodGet, "/wfh/pending", nil)
	req = handler.WithUser(req, mgr)
	rec := httptest.NewRecorder()
	h.Pending(rec, req)

	var got []domain.WFHRequest
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(got) != 1 || got[0].ID != w1.ID {
		t.Fatalf("Pending ควรเหลือแค่ w1 ได้: %+v", got)
	}
}

// เทส: WFHHandler.Approve โดย approver ตัวจริง
// input: คำขอ WFH ที่ ApproverID=mgr1 + เรียก Approve พร้อม path value id=คำขอนั้น ในฐานะ mgr1
// aspect: status 200, response body มี status="approved"
func TestWFHHandlerApprove(t *testing.T) {
	h, users := newWFHHandler()
	mgr := domain.User{ID: "mgr1", Role: domain.RoleDeveloper}
	emp := domain.User{ID: "emp1", ManagerID: "mgr1"}
	users.Seed(mgr)
	users.Seed(emp)

	created := createWFH(t, h, emp, `{"date":"2026-08-10"}`)

	req := httptest.NewRequest(http.MethodPost, "/wfh/"+created.ID+"/approve", nil)
	req.SetPathValue("id", created.ID)
	req = handler.WithUser(req, mgr)
	rec := httptest.NewRecorder()
	h.Approve(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var got domain.WFHRequest
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.Status != domain.StatusApproved {
		t.Fatalf("status ควรเป็น approved ได้ %q", got.Status)
	}
}

// เทส: WFHHandler.Approve ต้องปฏิเสธคนที่ไม่ใช่ approver ของคำขอนั้น
// input: คำขอ WFH ที่ ApproverID=mgr1 แต่เรียก Approve ในฐานะ "someone-else"
// aspect: status ต้องเป็น 403
func TestWFHHandlerApproveWrongApprover(t *testing.T) {
	h, users := newWFHHandler()
	mgr := domain.User{ID: "mgr1", Role: domain.RoleDeveloper}
	emp := domain.User{ID: "emp1", ManagerID: "mgr1"}
	users.Seed(mgr)
	users.Seed(emp)

	created := createWFH(t, h, emp, `{"date":"2026-08-10"}`)

	req := httptest.NewRequest(http.MethodPost, "/wfh/"+created.ID+"/approve", nil)
	req.SetPathValue("id", created.ID)
	req = handler.WithUser(req, domain.User{ID: "someone-else"})
	rec := httptest.NewRecorder()
	h.Approve(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

// เทส: WFHHandler.Reject โดย approver ตัวจริง
// input: คำขอ WFH ที่ ApproverID=mgr1 + เรียก Reject ในฐานะ mgr1
// aspect: response body ต้องมี status="rejected"
func TestWFHHandlerReject(t *testing.T) {
	h, users := newWFHHandler()
	mgr := domain.User{ID: "mgr1", Role: domain.RoleDeveloper}
	emp := domain.User{ID: "emp1", ManagerID: "mgr1"}
	users.Seed(mgr)
	users.Seed(emp)

	created := createWFH(t, h, emp, `{"date":"2026-08-10"}`)

	req := httptest.NewRequest(http.MethodPost, "/wfh/"+created.ID+"/reject", nil)
	req.SetPathValue("id", created.ID)
	req = handler.WithUser(req, mgr)
	rec := httptest.NewRecorder()
	h.Reject(rec, req)

	var got domain.WFHRequest
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.Status != domain.StatusRejected {
		t.Fatalf("status ควรเป็น rejected ได้ %q", got.Status)
	}
}

// เทส: WFHHandler.Approve กับ request id ที่ไม่มีอยู่จริง
// input: path value id="nope" ที่ไม่เคยถูกสร้างไว้เลย
// aspect: status ต้องเป็น 404
func TestWFHHandlerApproveNotFound(t *testing.T) {
	h, users := newWFHHandler()
	mgr := domain.User{ID: "mgr1", Role: domain.RoleDeveloper}
	users.Seed(mgr)

	req := httptest.NewRequest(http.MethodPost, "/wfh/nope/approve", nil)
	req.SetPathValue("id", "nope")
	req = handler.WithUser(req, mgr)
	rec := httptest.NewRecorder()
	h.Approve(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
