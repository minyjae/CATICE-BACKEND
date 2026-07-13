package user_test

import (
	"testing"

	"github/minyjae/catice/internal/auth/domain"
)

// เทส: Role.Valid() รับเฉพาะ 6 role ที่กำหนด และไม่รับค่านอกเหนือจากนั้น (regression ตรงบั๊กที่เพิ่งแก้: RoleHR เคยหายจาก switch ทำให้ register ด้วย role "hr" ไม่ผ่าน)
// input: role ที่ถูกต้องทั้ง 6 ตัว + ค่ามั่ว/ค่าว่าง/ค่าที่สะกดถูกแต่ตัวพิมพ์ผิด ("HR")
// aspect: role ที่ถูกต้องต้องได้ true ทั้งหมด (โดยเฉพาะ RoleHR), ค่านอกเหนือจากนั้นต้องได้ false และต้อง case-sensitive
func TestRoleValid(t *testing.T) {
	cases := []struct {
		role domain.Role
		want bool
	}{
		{domain.RoleDeveloper, true},
		{domain.RoleHR, true},
		{domain.RolePM, true},
		{domain.RolePO, true},
		{domain.RoleCTO, true},
		{domain.RoleUXUI, true},
		{domain.Role("manager"), false},
		{domain.Role(""), false},
		{domain.Role("HR"), false}, // case-sensitive
	}
	for _, c := range cases {
		if got := c.role.Valid(); got != c.want {
			t.Errorf("Role(%q).Valid() = %v, want %v", c.role, got, c.want)
		}
	}
}
