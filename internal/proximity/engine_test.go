package proximity

import "testing"

func TestDistance(t *testing.T) {
	a := PlayerState{X: 0, Y: 0}
	b := PlayerState{X: 3, Y: 1}
	if d := Distance(a, b); d != 3 { // Chebyshev = max(3,1) = 3
		t.Fatalf("Distance ผิด: อยากได้ 3 ได้ %d", d)
	}
}

func TestVolume(t *testing.T) {
	a := PlayerState{X: 0, Y: 0}

	// ทับกัน → 1.0
	if v := Volume(a, PlayerState{X: 0, Y: 0}, 3); v != 1.0 {
		t.Fatalf("ทับกันควรได้ 1.0 ได้ %v", v)
	}
	// เกินระยะ → 0
	if v := Volume(a, PlayerState{X: 9, Y: 0}, 3); v != 0 {
		t.Fatalf("ไกลเกินควรได้ 0 ได้ %v", v)
	}
	// ใกล้ ๆ → ระหว่าง 0 ถึง 1
	if v := Volume(a, PlayerState{X: 2, Y: 0}, 3); v <= 0 || v >= 1 {
		t.Fatalf("ระยะ 2 ควรอยู่ระหว่าง 0–1 ได้ %v", v)
	}
}
