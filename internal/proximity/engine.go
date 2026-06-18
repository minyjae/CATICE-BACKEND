// Package proximity คำนวณ "ความใกล้ → ระดับเสียง" สำหรับ proximity audio
// เป็น logic บริสุทธิ์ (pure) ไม่พึ่ง network/state → ทดสอบง่าย ใช้ซ้ำได้
//
// หมายเหตุ: ตอนนี้ frontend คำนวณ proximity เองฝั่ง client (เปิด/ปิดวิดีโอ)
// package นี้เตรียมไว้สำหรับย้ายมาคิดฝั่ง server (เช่น ปรับ volume ตามระยะ) ในอนาคต
package proximity

// Distance วัดระยะแบบ Chebyshev (เดินทแยงนับเป็น 1 ช่อง)
func Distance(a, b PlayerState) int {
	dx := abs(a.X - b.X)
	dy := abs(a.Y - b.Y)
	if dx > dy {
		return dx
	}
	return dy
}

// Volume คืนระดับเสียง 0.0–1.0 จากระยะห่าง
//   - ทับกัน (d=0)      → 1.0 (ดังสุด)
//   - ไกลขึ้น           → ค่อย ๆ ลด
//   - เกิน maxDist      → 0.0 (เงียบ)
func Volume(a, b PlayerState, maxDist int) float64 {
	if maxDist <= 0 {
		return 0
	}
	d := Distance(a, b)
	if d > maxDist {
		return 0
	}
	return 1 - float64(d)/float64(maxDist+1)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
