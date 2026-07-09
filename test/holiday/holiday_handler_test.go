package holiday_test

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

func newHolidayHandler() *handler.HolidayHandler {
	store := service.NewHolidayStore(fakes.NewHolidays())
	return handler.NewHolidayHandler(store)
}

// เทส: HolidayHandler.Create ทาง happy path โดย caller role HR
// input: HTTP POST body {"name":"Songkran","date":"2026-04-13"} โดยผู้เรียก role=HR
// aspect: status 200, response JSON มี name/created_by ตรงตามที่ส่งและตัวผู้เรียก
func TestHolidayHandlerCreateByHR(t *testing.T) {
	h := newHolidayHandler()
	caller := domain.User{ID: "hr1", Role: domain.RoleHR}

	req := httptest.NewRequest(http.MethodPost, "/holidays", strings.NewReader(`{"name":"Songkran","date":"2026-04-13"}`))
	req = handler.WithUser(req, caller)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var got domain.Holiday
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.Name != "Songkran" || got.CreatedBy != "hr1" {
		t.Fatalf("holiday ผิด: %+v", got)
	}
}

// เทส: HolidayHandler.Create ปฏิเสธผู้เรียกที่ role ไม่ใช่ HR
// input: body ที่ถูกต้อง แต่ผู้เรียก role=developer
// aspect: status ต้องเป็น 403
func TestHolidayHandlerCreateForbiddenForNonHR(t *testing.T) {
	h := newHolidayHandler()
	caller := domain.User{ID: "emp1", Role: domain.RoleDeveloper}

	req := httptest.NewRequest(http.MethodPost, "/holidays", strings.NewReader(`{"name":"Fake","date":"2026-01-01"}`))
	req = handler.WithUser(req, caller)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

// เทส: HolidayHandler.Create ปฏิเสธ request body ที่ไม่ใช่ JSON ที่ถูกต้อง
// input: body เป็น string "{bad json" โดยผู้เรียก role=HR
// aspect: status ต้องเป็น 400
func TestHolidayHandlerCreateBadBody(t *testing.T) {
	h := newHolidayHandler()
	caller := domain.User{ID: "hr1", Role: domain.RoleHR}

	req := httptest.NewRequest(http.MethodPost, "/holidays", strings.NewReader("{bad json"))
	req = handler.WithUser(req, caller)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// เทส: error จาก service (ErrEmptyHolidayName) ไหลผ่าน statusForErr มาเป็น HTTP status ที่ถูกต้อง
// input: body ที่ name เป็นค่าว่าง ""
// aspect: status ต้องเป็น 400
func TestHolidayHandlerCreateEmptyName(t *testing.T) {
	h := newHolidayHandler()
	caller := domain.User{ID: "hr1", Role: domain.RoleHR}

	req := httptest.NewRequest(http.MethodPost, "/holidays", strings.NewReader(`{"name":"","date":"2026-01-01"}`))
	req = handler.WithUser(req, caller)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// เทส: HolidayHandler.List คืนวันหยุดทั้งหมด และดูได้แม้ผู้เรียกไม่ใช่ HR
// input: สร้าง holiday ไว้ 1 รายการโดย HR แล้วเรียก List ในฐานะ developer ธรรมดา
// aspect: response ต้องมี 1 รายการ (ไม่ถูกบล็อกด้วย permission เพราะ List ไม่จำกัด role)
func TestHolidayHandlerList(t *testing.T) {
	h := newHolidayHandler()
	hr := domain.User{ID: "hr1", Role: domain.RoleHR}

	createReq := httptest.NewRequest(http.MethodPost, "/holidays", strings.NewReader(`{"name":"Songkran","date":"2026-04-13"}`))
	createReq = handler.WithUser(createReq, hr)
	h.Create(httptest.NewRecorder(), createReq)

	req := httptest.NewRequest(http.MethodGet, "/holidays", nil)
	req = handler.WithUser(req, domain.User{ID: "emp1", Role: domain.RoleDeveloper})
	rec := httptest.NewRecorder()
	h.List(rec, req)

	var got []domain.Holiday
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("List ควรมี 1 รายการ: %+v", got)
	}
}

// เทส: HolidayHandler.Delete โดย HR ลบสำเร็จ
// input: holiday ที่สร้างไว้ + path value id=holidayID + ผู้เรียก role=HR
// aspect: status ต้องเป็น 200
func TestHolidayHandlerDeleteByHR(t *testing.T) {
	h := newHolidayHandler()
	hr := domain.User{ID: "hr1", Role: domain.RoleHR}

	createReq := httptest.NewRequest(http.MethodPost, "/holidays", strings.NewReader(`{"name":"Songkran","date":"2026-04-13"}`))
	createReq = handler.WithUser(createReq, hr)
	createRec := httptest.NewRecorder()
	h.Create(createRec, createReq)
	var created domain.Holiday
	json.Unmarshal(createRec.Body.Bytes(), &created)

	req := httptest.NewRequest(http.MethodDelete, "/holidays/"+created.ID, nil)
	req.SetPathValue("id", created.ID)
	req = handler.WithUser(req, hr)
	rec := httptest.NewRecorder()
	h.Delete(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
}

// เทส: HolidayHandler.Delete ปฏิเสธผู้เรียกที่ role ไม่ใช่ HR
// input: holiday ที่สร้างไว้ + ผู้เรียก role=developer
// aspect: status ต้องเป็น 403
func TestHolidayHandlerDeleteForbiddenForNonHR(t *testing.T) {
	h := newHolidayHandler()
	hr := domain.User{ID: "hr1", Role: domain.RoleHR}

	createReq := httptest.NewRequest(http.MethodPost, "/holidays", strings.NewReader(`{"name":"Songkran","date":"2026-04-13"}`))
	createReq = handler.WithUser(createReq, hr)
	createRec := httptest.NewRecorder()
	h.Create(createRec, createReq)
	var created domain.Holiday
	json.Unmarshal(createRec.Body.Bytes(), &created)

	req := httptest.NewRequest(http.MethodDelete, "/holidays/"+created.ID, nil)
	req.SetPathValue("id", created.ID)
	req = handler.WithUser(req, domain.User{ID: "emp1", Role: domain.RoleDeveloper})
	rec := httptest.NewRecorder()
	h.Delete(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}
