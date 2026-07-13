package leave_test

import (
	"testing"

	"github/minyjae/catice/internal/auth/domain"
)

// เทส: LeaveType.Valid() รับเฉพาะ 3 ค่าที่กำหนด (vacation/sick/personal) เท่านั้น
// input: ค่า LeaveType ต่าง ๆ ทั้งที่ถูกต้องและไม่ถูกต้อง (รวมค่าว่างกับค่ามั่ว)
// aspect: ค่า valid ต้องได้ true, ค่านอกเหนือจากนั้นต้องได้ false — ป้องกันไม่ให้ยื่นคำขอลาด้วย type ที่ไม่รองรับ
func TestLeaveTypeValid(t *testing.T) {
	cases := []struct {
		typ  domain.LeaveType
		want bool
	}{
		{domain.LeaveVacation, true},
		{domain.LeaveSick, true},
		{domain.LeavePersonal, true},
		{domain.LeaveType(""), false},
		{domain.LeaveType("maternity"), false},
	}
	for _, c := range cases {
		if got := c.typ.Valid(); got != c.want {
			t.Errorf("LeaveType(%q).Valid() = %v, want %v", c.typ, got, c.want)
		}
	}
}

// เทส: RequestStatus.Valid() รับเฉพาะ pending/approved/rejected (ใช้ร่วมกันทั้ง LeaveRequest และ WFHRequest — เทสไว้ที่นี่ที่เดียว)
// input: ค่า RequestStatus ที่ถูกต้อง 3 ค่า และค่าว่าง/ค่ามั่ว
// aspect: ค่า valid ต้องได้ true, ค่านอกเหนือจากนั้นต้องได้ false
func TestRequestStatusValid(t *testing.T) {
	cases := []struct {
		status domain.RequestStatus
		want   bool
	}{
		{domain.StatusPending, true},
		{domain.StatusApproved, true},
		{domain.StatusRejected, true},
		{domain.RequestStatus(""), false},
		{domain.RequestStatus("cancelled"), false},
	}
	for _, c := range cases {
		if got := c.status.Valid(); got != c.want {
			t.Errorf("RequestStatus(%q).Valid() = %v, want %v", c.status, got, c.want)
		}
	}
}
