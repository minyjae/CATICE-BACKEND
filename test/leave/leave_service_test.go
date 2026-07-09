package leave_test

import (
	"testing"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/service"
	"github/minyjae/catice/test/fakes"
)

func newLeaveStore() (*service.LeaveStore, *fakes.Users) {
	users := fakes.NewUsers()
	policy := service.NewPolicyStore(fakes.NewPolicy())
	return service.NewLeaveStore(fakes.NewLeaves(), users, policy), users
}

// เทส: LeaveStore.Create หา approver จาก ManagerID ของผู้ยื่นเมื่อมีอยู่แล้ว
// input: user emp1 ที่มี ManagerID="mgr1" + payload ลาแบบ vacation ที่ถูกต้อง
// aspect: ApproverID ของคำขอต้องตรงกับ mgr1, Status ต้องเป็น pending, ID/CreatedAt ต้องถูกตั้งค่า (ไม่ว่าง/ไม่เป็นศูนย์)
func TestLeaveStoreCreateWithManager(t *testing.T) {
	store, users := newLeaveStore()
	users.Seed(domain.User{ID: "mgr1", Email: "mgr@x.com", Role: domain.RoleDeveloper})
	users.Seed(domain.User{ID: "emp1", Email: "emp@x.com", Role: domain.RoleDeveloper, ManagerID: "mgr1"})

	l, err := store.Create("emp1", domain.CreateLeavePayload{
		Type: domain.LeaveVacation, StartDate: "2026-08-01", EndDate: "2026-08-03", Reason: "trip",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if l.ID == "" {
		t.Fatal("ID ควรถูกตั้งค่า")
	}
	if l.ApproverID != "mgr1" {
		t.Fatalf("ApproverID ควรเป็น mgr1 ได้ %q", l.ApproverID)
	}
	if l.Status != domain.StatusPending {
		t.Fatalf("Status ควรเป็น pending ได้ %q", l.Status)
	}
	if l.CreatedAt == 0 {
		t.Fatal("CreatedAt ควรถูกตั้งค่า")
	}
}

// เทส: LeaveStore.Create fallback ไปหา user role HR เมื่อผู้ยื่นไม่มี ManagerID
// input: user emp1 ที่ไม่มี ManagerID + มี user hr1 role HR อยู่ในระบบ
// aspect: ApproverID ของคำขอต้อง fallback ไปเป็น hr1 (ไม่ error แม้ไม่มี manager ตรง ๆ)
func TestLeaveStoreCreateFallbackToHR(t *testing.T) {
	store, users := newLeaveStore()
	users.Seed(domain.User{ID: "hr1", Role: domain.RoleHR})
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper}) // ไม่มี ManagerID

	l, err := store.Create("emp1", domain.CreateLeavePayload{Type: domain.LeaveSick, StartDate: "2026-08-01", EndDate: "2026-08-01"})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if l.ApproverID != "hr1" {
		t.Fatalf("ApproverID ควร fallback ไป hr1 ได้ %q", l.ApproverID)
	}
}

// เทส: LeaveStore.Create ต้อง error เมื่อหา approver ไม่ได้เลย
// input: user emp1 ที่ไม่มี ManagerID และไม่มี user role HR อยู่ในระบบเลย
// aspect: error ที่ได้ต้องเป็น ErrNoApprover เท่านั้น (กันคำขอค้างไม่มีคนอนุมัติ)
func TestLeaveStoreCreateNoApprover(t *testing.T) {
	store, users := newLeaveStore()
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper}) // ไม่มี manager ไม่มี HR ในระบบ

	_, err := store.Create("emp1", domain.CreateLeavePayload{Type: domain.LeaveSick, StartDate: "2026-08-01", EndDate: "2026-08-01"})
	if err != domain.ErrNoApprover {
		t.Fatalf("error ควรเป็น ErrNoApprover ได้ %v", err)
	}
}

// เทส: LeaveStore.Create ปฏิเสธ payload ที่ Type ไม่ valid
// input: payload ที่ Type เป็นค่าว่าง ""
// aspect: error ที่ได้ต้องเป็น ErrEmptyLeaveType
func TestLeaveStoreCreateInvalidType(t *testing.T) {
	store, users := newLeaveStore()
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})

	_, err := store.Create("emp1", domain.CreateLeavePayload{Type: "", StartDate: "2026-08-01", EndDate: "2026-08-01"})
	if err != domain.ErrEmptyLeaveType {
		t.Fatalf("error ควรเป็น ErrEmptyLeaveType ได้ %v", err)
	}
}

// เทส: LeaveStore.Create ปฏิเสธช่วงวันที่ที่ไม่สมเหตุสมผล
// input: 2 เคส — (1) StartDate มาหลัง EndDate (2) StartDate/EndDate ว่างทั้งคู่
// aspect: ทั้งสองเคส error ที่ได้ต้องเป็น ErrInvalidDateRange
func TestLeaveStoreCreateInvalidDateRange(t *testing.T) {
	store, users := newLeaveStore()
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})

	_, err := store.Create("emp1", domain.CreateLeavePayload{Type: domain.LeaveVacation, StartDate: "2026-08-05", EndDate: "2026-08-01"})
	if err != domain.ErrInvalidDateRange {
		t.Fatalf("error ควรเป็น ErrInvalidDateRange (start>end) ได้ %v", err)
	}

	_, err = store.Create("emp1", domain.CreateLeavePayload{Type: domain.LeaveVacation, StartDate: "", EndDate: ""})
	if err != domain.ErrInvalidDateRange {
		t.Fatalf("error ควรเป็น ErrInvalidDateRange (ว่าง) ได้ %v", err)
	}
}

// เทส: LeaveStore.Decide(approve=true) โดย approver ตัวจริง กับคำขอที่ยัง pending
// input: คำขอลาที่เพิ่งสร้าง (status=pending) + เรียกโดย mgr1 ซึ่งเป็น ApproverID ของคำขอนั้น
// aspect: Status เปลี่ยนเป็น approved และ DecidedAt ต้องถูกตั้งค่า (ไม่เป็นศูนย์)
func TestLeaveStoreDecideApprove(t *testing.T) {
	store, users := newLeaveStore()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	l, err := store.Create("emp1", domain.CreateLeavePayload{Type: domain.LeaveVacation, StartDate: "2026-08-01", EndDate: "2026-08-01"})
	if err != nil {
		t.Fatalf("setup Create error: %v", err)
	}

	decided, err := store.Decide("mgr1", l.ID, true)
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

// เทส: LeaveStore.Decide(approve=false) โดย approver ตัวจริง กับคำขอที่ยัง pending
// input: คำขอลาที่เพิ่งสร้าง (status=pending) + เรียกโดย mgr1 พร้อม approve=false
// aspect: Status ต้องเปลี่ยนเป็น rejected
func TestLeaveStoreDecideReject(t *testing.T) {
	store, users := newLeaveStore()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	l, _ := store.Create("emp1", domain.CreateLeavePayload{Type: domain.LeaveVacation, StartDate: "2026-08-01", EndDate: "2026-08-01"})

	decided, err := store.Decide("mgr1", l.ID, false)
	if err != nil {
		t.Fatalf("Decide error: %v", err)
	}
	if decided.Status != domain.StatusRejected {
		t.Fatalf("Status ควรเป็น rejected ได้ %q", decided.Status)
	}
}

// เทส: LeaveStore.Decide ต้องปฏิเสธคนที่ไม่ใช่ approver ของคำขอนั้น
// input: คำขอลาที่ ApproverID=mgr1 แต่เรียก Decide ด้วย callerID="someone-else"
// aspect: error ต้องเป็น ErrNotApprover และ status ของคำขอต้อง "ไม่เปลี่ยน" (ยังคง pending)
func TestLeaveStoreDecideNotApprover(t *testing.T) {
	store, users := newLeaveStore()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	l, _ := store.Create("emp1", domain.CreateLeavePayload{Type: domain.LeaveVacation, StartDate: "2026-08-01", EndDate: "2026-08-01"})

	_, err := store.Decide("someone-else", l.ID, true)
	if err != domain.ErrNotApprover {
		t.Fatalf("error ควรเป็น ErrNotApprover ได้ %v", err)
	}

	mine := store.ListMine("emp1")
	if len(mine) != 1 || mine[0].Status != domain.StatusPending {
		t.Fatalf("status ไม่ควรเปลี่ยนหลังถูกปฏิเสธสิทธิ์: %+v", mine)
	}
}

// เทส: LeaveStore.Decide ต้องปฏิเสธการตัดสินใจซ้ำกับคำขอที่ตัดสินใจไปแล้ว
// input: คำขอลาที่ถูก approve ไปแล้วครั้งหนึ่ง แล้วเรียก Decide ซ้ำอีกครั้ง
// aspect: error ของการเรียกครั้งที่สองต้องเป็น ErrNotPending
func TestLeaveStoreDecideAlreadyDecided(t *testing.T) {
	store, users := newLeaveStore()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	l, _ := store.Create("emp1", domain.CreateLeavePayload{Type: domain.LeaveVacation, StartDate: "2026-08-01", EndDate: "2026-08-01"})

	if _, err := store.Decide("mgr1", l.ID, true); err != nil {
		t.Fatalf("setup Decide error: %v", err)
	}
	if _, err := store.Decide("mgr1", l.ID, true); err != domain.ErrNotPending {
		t.Fatalf("error ควรเป็น ErrNotPending ได้ %v", err)
	}
}

// เทส: LeaveStore.Decide กับ request id ที่ไม่มีอยู่จริงในระบบ
// input: requestID="nonexistent" ที่ไม่เคยถูกสร้างไว้เลย
// aspect: error ต้องเป็น ErrRequestNotFound
func TestLeaveStoreDecideNotFound(t *testing.T) {
	store, _ := newLeaveStore()
	if _, err := store.Decide("mgr1", "nonexistent", true); err != domain.ErrRequestNotFound {
		t.Fatalf("error ควรเป็น ErrRequestNotFound ได้ %v", err)
	}
}

// เทส: LeaveStore.ListMine กรองเฉพาะคำขอของ user คนที่ถามเท่านั้น
// input: สร้างคำขอลาให้ emp1 1 ใบ และ emp2 1 ใบ แล้วเรียก ListMine("emp1")
// aspect: ผลลัพธ์ต้องมีแค่ 1 รายการ และเป็นของ emp1 เท่านั้น ไม่ปนของ emp2
func TestLeaveStoreListMine(t *testing.T) {
	store, users := newLeaveStore()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	users.Seed(domain.User{ID: "emp2", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	store.Create("emp1", domain.CreateLeavePayload{Type: domain.LeaveVacation, StartDate: "2026-08-01", EndDate: "2026-08-01"})
	store.Create("emp2", domain.CreateLeavePayload{Type: domain.LeaveVacation, StartDate: "2026-08-02", EndDate: "2026-08-02"})

	mine := store.ListMine("emp1")
	if len(mine) != 1 || mine[0].UserID != "emp1" {
		t.Fatalf("ListMine ผิด: %+v", mine)
	}
}

// เทส: LeaveStore.ListPending กรองเฉพาะคำขอที่ approver คนนี้ต้องตัดสินใจ "และ" ยัง pending อยู่
// input: emp1 ยื่นคำขอ 2 ใบ (l1, l2) ให้ mgr1 อนุมัติ แล้ว mgr1 approve l2 ไปก่อน
// aspect: ListPending("mgr1") ต้องเหลือแค่ l1 (ที่ยัง pending) — l2 ที่ตัดสินใจแล้วต้องไม่โผล่มา
func TestLeaveStoreListPending(t *testing.T) {
	store, users := newLeaveStore()
	users.Seed(domain.User{ID: "mgr1", Role: domain.RoleDeveloper})
	users.Seed(domain.User{ID: "emp1", Role: domain.RoleDeveloper, ManagerID: "mgr1"})
	l1, _ := store.Create("emp1", domain.CreateLeavePayload{Type: domain.LeaveVacation, StartDate: "2026-08-01", EndDate: "2026-08-01"})
	l2, _ := store.Create("emp1", domain.CreateLeavePayload{Type: domain.LeaveSick, StartDate: "2026-08-05", EndDate: "2026-08-05"})
	if _, err := store.Decide("mgr1", l2.ID, true); err != nil {
		t.Fatalf("setup Decide error: %v", err)
	}

	pending := store.ListPending("mgr1")
	if len(pending) != 1 || pending[0].ID != l1.ID {
		t.Fatalf("ListPending ควรเหลือแค่ l1 (ยัง pending) ได้: %+v", pending)
	}
}
