package diary_test

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

func newDiaryHandler() (*handler.DiaryHandler, *fakes.Users) {
	users := fakes.NewUsers()
	store := service.NewDiaryStore(fakes.NewDiaries(), users)
	return handler.NewDiaryHandler(store), users
}

// เทส: DiaryHandler.Upsert ทาง happy path
// input: HTTP POST body {"date":"2026-07-09","content":"fixed bug"} โดยผู้เรียก id="emp1"
// aspect: status 200, response body content ตรงกับที่ส่งไป
func TestDiaryHandlerUpsert(t *testing.T) {
	h, _ := newDiaryHandler()
	caller := domain.User{ID: "emp1"}

	req := httptest.NewRequest(http.MethodPost, "/diary", strings.NewReader(`{"date":"2026-07-09","content":"fixed bug"}`))
	req = handler.WithUser(req, caller)
	rec := httptest.NewRecorder()
	h.Upsert(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var got domain.DailyDiary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.Content != "fixed bug" {
		t.Fatalf("content ผิด: %+v", got)
	}
}

// เทส: error จาก service (ErrEmptyDiaryContent) ไหลผ่าน statusForErr มาเป็น HTTP status ที่ถูกต้อง
// input: body ที่ content เป็นค่าว่าง ""
// aspect: status ต้องเป็น 400
func TestDiaryHandlerUpsertEmptyContent(t *testing.T) {
	h, _ := newDiaryHandler()
	caller := domain.User{ID: "emp1"}

	req := httptest.NewRequest(http.MethodPost, "/diary", strings.NewReader(`{"date":"2026-07-09","content":""}`))
	req = handler.WithUser(req, caller)
	rec := httptest.NewRecorder()
	h.Upsert(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// เทส: DiaryHandler.Mine อ่านค่า query param ?limit= และส่งต่อไปยัง service ถูกต้อง
// input: บันทึก diary ให้ emp1 3 วัน แล้วเรียก GET /diary/mine?limit=2
// aspect: response ต้องมีจำนวนเท่ากับ limit ที่ขอ คือ 2 รายการ
func TestDiaryHandlerMineLimit(t *testing.T) {
	h, _ := newDiaryHandler()
	caller := domain.User{ID: "emp1"}

	dates := []string{"2026-07-07", "2026-07-08", "2026-07-09"}
	for _, d := range dates {
		req := httptest.NewRequest(http.MethodPost, "/diary", strings.NewReader(`{"date":"`+d+`","content":"work"}`))
		req = handler.WithUser(req, caller)
		h.Upsert(httptest.NewRecorder(), req)
	}

	req := httptest.NewRequest(http.MethodGet, "/diary/mine?limit=2", nil)
	req = handler.WithUser(req, caller)
	rec := httptest.NewRecorder()
	h.Mine(rec, req)

	var got []domain.DailyDiary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("limit=2 ควรได้ 2 รายการ ได้ %d", len(got))
	}
}

// เทส: DiaryHandler.OfUser อนุญาตให้ HR ดู diary ของ user คนอื่นผ่าน query param user_id/date
// input: emp1 บันทึก diary วันที่ "2026-07-09" ไว้ + เรียก GET /diary?user_id=emp1&date=2026-07-09 ในฐานะ hr1 role HR
// aspect: status ต้องเป็น 200
func TestDiaryHandlerOfUserByHR(t *testing.T) {
	h, users := newDiaryHandler()
	target := domain.User{ID: "emp1", ManagerID: "mgr1"}
	users.Seed(target)

	upsertReq := httptest.NewRequest(http.MethodPost, "/diary", strings.NewReader(`{"date":"2026-07-09","content":"fixed bug"}`))
	upsertReq = handler.WithUser(upsertReq, target)
	h.Upsert(httptest.NewRecorder(), upsertReq)

	req := httptest.NewRequest(http.MethodGet, "/diary?user_id=emp1&date=2026-07-09", nil)
	req = handler.WithUser(req, domain.User{ID: "hr1", Role: domain.RoleHR})
	rec := httptest.NewRecorder()
	h.OfUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
}

// เทส: DiaryHandler.OfUser ปฏิเสธผู้เรียกที่ไม่ใช่ทั้ง HR และไม่ใช่ manager ของ target
// input: emp1 มี ManagerID="mgr1" แต่เรียกดูในฐานะ "coworker1" (role developer ธรรมดา)
// aspect: status ต้องเป็น 403
func TestDiaryHandlerOfUserForbidden(t *testing.T) {
	h, users := newDiaryHandler()
	target := domain.User{ID: "emp1", ManagerID: "mgr1"}
	users.Seed(target)

	upsertReq := httptest.NewRequest(http.MethodPost, "/diary", strings.NewReader(`{"date":"2026-07-09","content":"fixed bug"}`))
	upsertReq = handler.WithUser(upsertReq, target)
	h.Upsert(httptest.NewRecorder(), upsertReq)

	req := httptest.NewRequest(http.MethodGet, "/diary?user_id=emp1&date=2026-07-09", nil)
	req = handler.WithUser(req, domain.User{ID: "coworker1", Role: domain.RoleDeveloper})
	rec := httptest.NewRecorder()
	h.OfUser(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

// เทส: DiaryHandler.OfUser คืน 404 เมื่อ target ไม่มี entry ของวันที่ถามจริง (แม้ caller มีสิทธิ์ดู)
// input: emp1 มี ManagerID="mgr1" แต่ไม่เคยเขียน diary เลย + เรียกในฐานะ mgr1 ซึ่งมีสิทธิ์
// aspect: status ต้องเป็น 404
func TestDiaryHandlerOfUserNotFound(t *testing.T) {
	h, users := newDiaryHandler()
	users.Seed(domain.User{ID: "emp1", ManagerID: "mgr1"})

	req := httptest.NewRequest(http.MethodGet, "/diary?user_id=emp1&date=2026-07-09", nil)
	req = handler.WithUser(req, domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	rec := httptest.NewRecorder()
	h.OfUser(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
