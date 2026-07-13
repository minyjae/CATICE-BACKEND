package diary_test

import (
	"testing"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/service"
	"github/minyjae/catice/test/fakes"
)

func newDiaryStore() (*service.DiaryStore, *fakes.Users) {
	users := fakes.NewUsers()
	return service.NewDiaryStore(fakes.NewDiaries(), users), users
}

// เทส: DiaryStore.Upsert สร้าง entry ใหม่เมื่อยังไม่มี entry ของ (user,date) นี้มาก่อน
// input: userID="emp1", payload {Date:"2026-07-09", Content:"fixed bug"}
// aspect: ไม่ error, ID ถูกตั้งค่า, และ CreatedAt ต้องเท่ากับ UpdatedAt (เพราะเพิ่งสร้างครั้งแรก)
func TestDiaryStoreUpsertCreatesNew(t *testing.T) {
	store, _ := newDiaryStore()

	d, err := store.Upsert("emp1", domain.UpsertDiaryPayload{Date: "2026-07-09", Content: "fixed bug"})
	if err != nil {
		t.Fatalf("Upsert error: %v", err)
	}
	if d.ID == "" {
		t.Fatal("ID ควรถูกตั้งค่า")
	}
	if d.CreatedAt != d.UpdatedAt {
		t.Fatalf("entry แรก CreatedAt ควรเท่ากับ UpdatedAt ได้ %d != %d", d.CreatedAt, d.UpdatedAt)
	}
}

// เทส: DiaryStore.Upsert เรียกซ้ำวันเดียวกันต้องทับ ไม่สร้างแถวใหม่ (mimic ON CONFLICT ของ GORM จริง)
// input: Upsert("emp1", date="2026-07-09") สองครั้งติดกันด้วย content คนละค่า
// aspect: ครั้งที่สอง ID/CreatedAt ต้องเท่าเดิม, Content/UpdatedAt ต้องเปลี่ยนเป็นค่าล่าสุด, และ ListMine ต้องมีแค่ 1 entry ไม่ซ้ำ
func TestDiaryStoreUpsertOverwritesSameDay(t *testing.T) {
	store, _ := newDiaryStore()

	first, err := store.Upsert("emp1", domain.UpsertDiaryPayload{Date: "2026-07-09", Content: "fixed bug"})
	if err != nil {
		t.Fatalf("Upsert 1 error: %v", err)
	}

	second, err := store.Upsert("emp1", domain.UpsertDiaryPayload{Date: "2026-07-09", Content: "fixed bug + wrote tests"})
	if err != nil {
		t.Fatalf("Upsert 2 error: %v", err)
	}

	if second.ID != first.ID {
		t.Fatalf("ID ไม่ควรเปลี่ยนตอนทับวันเดิม: %q != %q", second.ID, first.ID)
	}
	if second.CreatedAt != first.CreatedAt {
		t.Fatalf("CreatedAt ไม่ควรเปลี่ยนตอนทับวันเดิม: %d != %d", second.CreatedAt, first.CreatedAt)
	}
	if second.Content != "fixed bug + wrote tests" {
		t.Fatalf("Content ควรถูกทับด้วยค่าใหม่ ได้ %q", second.Content)
	}

	mine := store.ListMine("emp1", 30)
	if len(mine) != 1 {
		t.Fatalf("ไม่ควรมี entry ซ้ำวันเดียวกัน: %+v", mine)
	}
}

// เทส: DiaryStore.Upsert ปฏิเสธ content ที่ว่างหรือมีแต่ whitespace
// input: payload Content="   " (มีแต่ space)
// aspect: error ต้องเป็น ErrEmptyDiaryContent
func TestDiaryStoreUpsertEmptyContent(t *testing.T) {
	store, _ := newDiaryStore()

	_, err := store.Upsert("emp1", domain.UpsertDiaryPayload{Date: "2026-07-09", Content: "   "})
	if err != domain.ErrEmptyDiaryContent {
		t.Fatalf("error ควรเป็น ErrEmptyDiaryContent ได้ %v", err)
	}
}

// เทส: DiaryStore.Upsert ปฏิเสธ date ที่ว่าง
// input: payload Date=""
// aspect: error ต้องเป็น ErrInvalidDateRange
func TestDiaryStoreUpsertEmptyDate(t *testing.T) {
	store, _ := newDiaryStore()

	_, err := store.Upsert("emp1", domain.UpsertDiaryPayload{Date: "", Content: "fixed bug"})
	if err != domain.ErrInvalidDateRange {
		t.Fatalf("error ควรเป็น ErrInvalidDateRange ได้ %v", err)
	}
}

// เทส: DiaryStore.ListMine จำกัดจำนวนผลลัพธ์ตาม limit ที่ส่งเข้าไป
// input: สร้าง diary ให้ emp1 3 วันติดกัน แล้วเรียก ListMine("emp1", 2)
// aspect: ผลลัพธ์ต้องมีจำนวนไม่เกิน (เท่ากับ) limit ที่ขอ คือ 2 รายการ
func TestDiaryStoreListMineLimit(t *testing.T) {
	store, _ := newDiaryStore()
	store.Upsert("emp1", domain.UpsertDiaryPayload{Date: "2026-07-07", Content: "a"})
	store.Upsert("emp1", domain.UpsertDiaryPayload{Date: "2026-07-08", Content: "b"})
	store.Upsert("emp1", domain.UpsertDiaryPayload{Date: "2026-07-09", Content: "c"})

	got := store.ListMine("emp1", 2)
	if len(got) != 2 {
		t.Fatalf("limit ควรจำกัดเหลือ 2 ได้ %d รายการ", len(got))
	}
}

// เทส: DiaryStore.OfUser อนุญาตให้ caller role HR ดู diary ของใครก็ได้ แม้ไม่ใช่ manager ของเขา
// input: emp1 มี diary วันที่ "2026-07-09" + เรียก OfUser("hr1", RoleHR, "emp1", "2026-07-09")
// aspect: ไม่ error, ok=true, และ content ตรงกับที่บันทึกไว้
func TestDiaryStoreOfUserByHR(t *testing.T) {
	store, users := newDiaryStore()
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	store.Upsert("emp1", domain.UpsertDiaryPayload{Date: "2026-07-09", Content: "fixed bug"})

	d, ok, err := store.OfUser("hr1", domain.RoleHR, "emp1", "2026-07-09")
	if err != nil {
		t.Fatalf("OfUser error: %v", err)
	}
	if !ok || d.Content != "fixed bug" {
		t.Fatalf("HR ควรเห็น diary ได้: ok=%v d=%+v", ok, d)
	}
}

// เทส: DiaryStore.OfUser อนุญาตให้ caller ที่เป็น manager ของ target ดู diary ได้ แม้ role ไม่ใช่ HR
// input: emp1 มี ManagerID="mgr1" + emp1 มี diary วันที่ "2026-07-09" + เรียก OfUser("mgr1", RoleDeveloper, "emp1", "2026-07-09")
// aspect: ไม่ error, ok=true, content ตรงกับที่บันทึกไว้
func TestDiaryStoreOfUserByManager(t *testing.T) {
	store, users := newDiaryStore()
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	store.Upsert("emp1", domain.UpsertDiaryPayload{Date: "2026-07-09", Content: "fixed bug"})

	d, ok, err := store.OfUser("mgr1", domain.RoleDeveloper, "emp1", "2026-07-09")
	if err != nil {
		t.Fatalf("OfUser error: %v", err)
	}
	if !ok || d.Content != "fixed bug" {
		t.Fatalf("manager ควรเห็น diary ของลูกทีมได้: ok=%v d=%+v", ok, d)
	}
}

// เทส: DiaryStore.OfUser ปฏิเสธ caller ที่ไม่ใช่ทั้ง HR และไม่ใช่ manager ของ target
// input: emp1 มี ManagerID="mgr1" แต่เรียก OfUser ในฐานะ "coworker1" (role developer ธรรมดา ไม่ใช่ mgr1)
// aspect: error ต้องเป็น ErrForbidden
func TestDiaryStoreOfUserForbidden(t *testing.T) {
	store, users := newDiaryStore()
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	store.Upsert("emp1", domain.UpsertDiaryPayload{Date: "2026-07-09", Content: "fixed bug"})

	_, _, err := store.OfUser("coworker1", domain.RoleDeveloper, "emp1", "2026-07-09")
	if err != domain.ErrForbidden {
		t.Fatalf("error ควรเป็น ErrForbidden ได้ %v", err)
	}
}

// เทส: DiaryStore.OfUser คืน ok=false (ไม่ error) เมื่อ target ไม่มี entry ของวันที่ถามจริง
// input: emp1 มี ManagerID="mgr1" แต่ไม่เคยเขียน diary เลย + เรียก OfUser("mgr1", ..., "emp1", "2026-07-09")
// aspect: err ต้องเป็น nil และ ok ต้องเป็น false (แยกกรณี "ไม่มีสิทธิ์" กับ "ไม่มีข้อมูล" ออกจากกันชัดเจน)
func TestDiaryStoreOfUserNotFound(t *testing.T) {
	store, users := newDiaryStore()
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	// ไม่มี entry วันนี้เลย

	_, ok, err := store.OfUser("mgr1", domain.RoleDeveloper, "emp1", "2026-07-09")
	if err != nil {
		t.Fatalf("ไม่ควร error: %v", err)
	}
	if ok {
		t.Fatal("ok ควรเป็น false เพราะไม่มี entry วันนี้")
	}
}
