package proximity

// PlayerState คือข้อมูลตำแหน่งขั้นต่ำที่ใช้คำนวณความใกล้
// (แยกออกจาก room.Player เพื่อให้ package นี้ "บริสุทธิ์" ไม่พึ่งใคร)
type PlayerState struct {
	ID string
	X  int
	Y  int
}
