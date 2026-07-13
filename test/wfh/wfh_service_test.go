package wfh_test

import (
	"testing"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/service"
	"github/minyjae/catice/test/fakes"
)

func newWFHStore() (*service.WFHStore, *fakes.Users) {
	users := fakes.NewUsers()
	policy := service.NewPolicyStore(fakes.NewPolicy())
	return service.NewWFHStore(fakes.NewWFH(), users, policy), users
}

// เทส: WFHStore.Create หา approver จาก ManagerID ของผู้ยื่นเมื่อมีอยู่แล้ว
// input: user emp1 ที่มี ManagerID="mgr1" + payload ขอ WFH วันเดียวที่ถูกต้อง
// aspect: ApproverID ของคำขอต้องตรงกับ mgr1, Status ต้องเป็น pending, ID ต้องถูกตั้งค่า
func TestWFHStoreCreateWithManager(t *testing.T) {
	store, users := newWFHStore()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})

	w, err := store.Create("emp1", domain.CreateWFHPayload{Date: "2026-08-10", Reason: "internet install"})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if w.ID == "" {
		t.Fatal("ID ควรถูกตั้งค่า")
	}
	if w.ApproverID != "mgr1" {
		t.Fatalf("ApproverID ควรเป็น mgr1 ได้ %q", w.ApproverID)
	}
	if w.Status != domain.StatusPending {
		t.Fatalf("Status ควรเป็น pending ได้ %q", w.Status)
	}
}

// เทส: WFHStore.Create fallback ไปหา user role HR เมื่อผู้ยื่นไม่มี ManagerID
// input: user emp1 ที่ไม่มี ManagerID + มี user hr1 role HR อยู่ในระบบ
// aspect: ApproverID ของคำขอต้อง fallback ไปเป็น hr1
func TestWFHStoreCreateFallbackToHR(t *testing.T) {
	store, users := newWFHStore()
	users.Seed(domain.User{ID: "hr1", Role: domain.RoleHR})
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper})

	w, err := store.Create("emp1", domain.CreateWFHPayload{Date: "2026-08-10"})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if w.ApproverID != "hr1" {
		t.Fatalf("ApproverID ควร fallback ไป hr1 ได้ %q", w.ApproverID)
	}
}

// เทส: WFHStore.Create ต้อง error เมื่อหา approver ไม่ได้เลย
// input: user emp1 ที่ไม่มี ManagerID และไม่มี user role HR อยู่ในระบบเลย
// aspect: error ที่ได้ต้องเป็น ErrNoApprover
func TestWFHStoreCreateNoApprover(t *testing.T) {
	store, users := newWFHStore()
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper})

	_, err := store.Create("emp1", domain.CreateWFHPayload{Date: "2026-08-10"})
	if err != domain.ErrNoApprover {
		t.Fatalf("error ควรเป็น ErrNoApprover ได้ %v", err)
	}
}

// เทส: WFHStore.Create ปฏิเสธ payload ที่ Date ว่าง
// input: payload ที่ Date เป็นค่าว่าง ""
// aspect: error ที่ได้ต้องเป็น ErrInvalidDateRange
func TestWFHStoreCreateEmptyDate(t *testing.T) {
	store, users := newWFHStore()
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})

	_, err := store.Create("emp1", domain.CreateWFHPayload{Date: ""})
	if err != domain.ErrInvalidDateRange {
		t.Fatalf("error ควรเป็น ErrInvalidDateRange ได้ %v", err)
	}
}

// เทส: WFHStore.Decide(approve=true) โดย approver ตัวจริง กับคำขอที่ยัง pending
// input: คำขอ WFH ที่เพิ่งสร้าง (status=pending) + เรียกโดย mgr1 ซึ่งเป็น ApproverID
// aspect: Status เปลี่ยนเป็น approved และ DecidedAt ต้องถูกตั้งค่า
func TestWFHStoreDecideApprove(t *testing.T) {
	store, users := newWFHStore()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	w, _ := store.Create("emp1", domain.CreateWFHPayload{Date: "2026-08-10"})

	decided, err := store.Decide("mgr1", w.ID, true)
	if err != nil {
		t.Fatalf("Decide error: %v", err)
	}
	if decided.Status != domain.StatusApproved {
		t.Fatalf("Status ควรเป็น approved ได้ %q", decided.Status)
	}
	if decided.DecidedAt == 0 {
		t.Fatal("DecidedAt ควรถูกตั้งค่า")
	}
}

// เทส: WFHStore.Decide(approve=false) โดย approver ตัวจริง กับคำขอที่ยัง pending
// input: คำขอ WFH ที่เพิ่งสร้าง + เรียกโดย mgr1 พร้อม approve=false
// aspect: Status ต้องเปลี่ยนเป็น rejected
func TestWFHStoreDecideReject(t *testing.T) {
	store, users := newWFHStore()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	w, _ := store.Create("emp1", domain.CreateWFHPayload{Date: "2026-08-10"})

	decided, err := store.Decide("mgr1", w.ID, false)
	if err != nil {
		t.Fatalf("Decide error: %v", err)
	}
	if decided.Status != domain.StatusRejected {
		t.Fatalf("Status ควรเป็น rejected ได้ %q", decided.Status)
	}
}

// เทส: WFHStore.Decide ต้องปฏิเสธคนที่ไม่ใช่ approver ของคำขอนั้น
// input: คำขอ WFH ที่ ApproverID=mgr1 แต่เรียก Decide ด้วย callerID="someone-else"
// aspect: error ต้องเป็น ErrNotApprover
func TestWFHStoreDecideNotApprover(t *testing.T) {
	store, users := newWFHStore()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	w, _ := store.Create("emp1", domain.CreateWFHPayload{Date: "2026-08-10"})

	if _, err := store.Decide("someone-else", w.ID, true); err != domain.ErrNotApprover {
		t.Fatalf("error ควรเป็น ErrNotApprover ได้ %v", err)
	}
}

// เทส: WFHStore.Decide ต้องปฏิเสธการตัดสินใจซ้ำกับคำขอที่ตัดสินใจไปแล้ว
// input: คำขอ WFH ที่ถูก approve ไปแล้วครั้งหนึ่ง แล้วเรียก Decide ซ้ำอีกครั้ง
// aspect: error ของการเรียกครั้งที่สองต้องเป็น ErrNotPending
func TestWFHStoreDecideAlreadyDecided(t *testing.T) {
	store, users := newWFHStore()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	w, _ := store.Create("emp1", domain.CreateWFHPayload{Date: "2026-08-10"})

	if _, err := store.Decide("mgr1", w.ID, true); err != nil {
		t.Fatalf("setup Decide error: %v", err)
	}
	if _, err := store.Decide("mgr1", w.ID, true); err != domain.ErrNotPending {
		t.Fatalf("error ควรเป็น ErrNotPending ได้ %v", err)
	}
}

// เทส: WFHStore.Decide กับ request id ที่ไม่มีอยู่จริงในระบบ
// input: requestID="nonexistent" ที่ไม่เคยถูกสร้างไว้เลย
// aspect: error ต้องเป็น ErrRequestNotFound
func TestWFHStoreDecideNotFound(t *testing.T) {
	store, _ := newWFHStore()
	if _, err := store.Decide("mgr1", "nonexistent", true); err != domain.ErrRequestNotFound {
		t.Fatalf("error ควรเป็น ErrRequestNotFound ได้ %v", err)
	}
}

// เทส: WFHStore.ListMine กรองเฉพาะคำขอของ user คนที่ถามเท่านั้น
// input: สร้างคำขอ WFH ให้ emp1 1 ใบ และ emp2 1 ใบ แล้วเรียก ListMine("emp1")
// aspect: ผลลัพธ์ต้องมีแค่ 1 รายการ และเป็นของ emp1 เท่านั้น
func TestWFHStoreListMine(t *testing.T) {
	store, users := newWFHStore()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	users.Seed(domain.User{ID: "emp2", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	store.Create("emp1", domain.CreateWFHPayload{Date: "2026-08-10"})
	store.Create("emp2", domain.CreateWFHPayload{Date: "2026-08-11"})

	mine := store.ListMine("emp1")
	if len(mine) != 1 || mine[0].UserID != "emp1" {
		t.Fatalf("ListMine ผิด: %+v", mine)
	}
}

// เทส: WFHStore.ListPending กรองเฉพาะคำขอที่ approver คนนี้ต้องตัดสินใจ "และ" ยัง pending อยู่
// input: emp1 ยื่นคำขอ WFH 2 ใบ (w1, w2) ให้ mgr1 อนุมัติ แล้ว mgr1 approve w2 ไปก่อน
// aspect: ListPending("mgr1") ต้องเหลือแค่ w1 (ที่ยัง pending)
func TestWFHStoreListPending(t *testing.T) {
	store, users := newWFHStore()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	w1, _ := store.Create("emp1", domain.CreateWFHPayload{Date: "2026-08-10"})
	w2, _ := store.Create("emp1", domain.CreateWFHPayload{Date: "2026-08-11"})
	if _, err := store.Decide("mgr1", w2.ID, true); err != nil {
		t.Fatalf("setup Decide error: %v", err)
	}

	pending := store.ListPending("mgr1")
	if len(pending) != 1 || pending[0].ID != w1.ID {
		t.Fatalf("ListPending ควรเหลือแค่ w1 (ยัง pending) ได้: %+v", pending)
	}
}
