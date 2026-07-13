package holiday_test

import (
	"testing"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/service"
	"github/minyjae/catice/test/fakes"
)

func newHolidayStore() (*service.HolidayStore, *fakes.Holidays) {
	repo := fakes.NewHolidays()
	return service.NewHolidayStore(repo), repo
}

// เทส: HolidayStore.Create สำเร็จเมื่อ caller role เป็น HR และชื่อไม่ว่าง
// input: callerRole=RoleHR, createdBy="hr1", payload {Name:"Songkran", Date:"2026-04-13"}
// aspect: ไม่ error, ID ถูกตั้งค่า (ไม่ว่าง), CreatedBy ตรงกับ hr1, และ List() ต้องมี holiday นี้อยู่จริง
func TestHolidayStoreCreateByHR(t *testing.T) {
	store, _ := newHolidayStore()

	h, err := store.Create(domain.RoleHR, "hr1", domain.CreateHolidayPayload{Name: "Songkran", Date: "2026-04-13"})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if h.ID == "" {
		t.Fatal("ID ควรถูกตั้งค่า")
	}
	if h.CreatedBy != "hr1" {
		t.Fatalf("CreatedBy ควรเป็น hr1 ได้ %q", h.CreatedBy)
	}

	all := store.List()
	if len(all) != 1 || all[0].ID != h.ID {
		t.Fatalf("List ควรมี holiday ที่สร้างไว้: %+v", all)
	}
}

// เทส: HolidayStore.Create ต้องปฏิเสธ caller ที่ role ไม่ใช่ HR
// input: callerRole=RoleDeveloper
// aspect: error ต้องเป็น ErrForbidden
func TestHolidayStoreCreateForbiddenForNonHR(t *testing.T) {
	store, _ := newHolidayStore()

	_, err := store.Create(domain.RoleDeveloper, "emp1", domain.CreateHolidayPayload{Name: "Fake", Date: "2026-01-01"})
	if err != domain.ErrForbidden {
		t.Fatalf("error ควรเป็น ErrForbidden ได้ %v", err)
	}
}

// เทส: HolidayStore.Create ปฏิเสธชื่อที่ว่างหรือมีแต่ whitespace
// input: callerRole=RoleHR, payload Name="   " (มีแต่ space)
// aspect: error ต้องเป็น ErrEmptyHolidayName (พิสูจน์ว่ามีการ trim ก่อนเช็คว่าง)
func TestHolidayStoreCreateEmptyName(t *testing.T) {
	store, _ := newHolidayStore()

	_, err := store.Create(domain.RoleHR, "hr1", domain.CreateHolidayPayload{Name: "   ", Date: "2026-01-01"})
	if err != domain.ErrEmptyHolidayName {
		t.Fatalf("error ควรเป็น ErrEmptyHolidayName ได้ %v", err)
	}
}

// เทส: HolidayStore.Delete โดย HR ลบวันหยุดออกจริง
// input: holiday ที่สร้างไว้ก่อนแล้ว + เรียก Delete(RoleHR, holidayID)
// aspect: ไม่ error และ List() หลังลบต้องว่างเปล่า
func TestHolidayStoreDelete(t *testing.T) {
	store, _ := newHolidayStore()
	h, _ := store.Create(domain.RoleHR, "hr1", domain.CreateHolidayPayload{Name: "Songkran", Date: "2026-04-13"})

	if err := store.Delete(domain.RoleHR, h.ID); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if all := store.List(); len(all) != 0 {
		t.Fatalf("List ควรว่างหลังลบ: %+v", all)
	}
}

// เทส: HolidayStore.Delete ต้องปฏิเสธ caller ที่ role ไม่ใช่ HR และห้ามลบจริง
// input: holiday ที่สร้างไว้ + เรียก Delete(RoleDeveloper, holidayID)
// aspect: error ต้องเป็น ErrForbidden และ List() ต้องยังมี holiday นั้นอยู่ (ไม่ถูกลบ)
func TestHolidayStoreDeleteForbiddenForNonHR(t *testing.T) {
	store, _ := newHolidayStore()
	h, _ := store.Create(domain.RoleHR, "hr1", domain.CreateHolidayPayload{Name: "Songkran", Date: "2026-04-13"})

	if err := store.Delete(domain.RoleDeveloper, h.ID); err != domain.ErrForbidden {
		t.Fatalf("error ควรเป็น ErrForbidden ได้ %v", err)
	}
	if all := store.List(); len(all) != 1 {
		t.Fatalf("ไม่ควรถูกลบ: %+v", all)
	}
}
