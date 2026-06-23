package domain

// Message = ข้อความแชต 1 ชิ้นที่เก็บถาวร (room / all / private)
//   - Scope    : "room" | "all" | "private"
//   - Room     : ห้อง (เฉพาะ scope=room) — ใช้ดึงประวัติเฉพาะห้อง
//   - FromID   : id ผู้ส่ง (จาก JWT) / FromName : ชื่อแสดงผลตอนส่ง (denormalize ไว้แสดงประวัติ)
//   - To       : id ปลายทาง (เฉพาะ scope=private)
//   - CreatedAt: unix seconds — ใช้เรียงลำดับ
type Message struct {
	ID        string `json:"id"`
	Scope     string `json:"scope"`
	Room      string `json:"room"`
	FromID    string `json:"from_id"`
	FromName  string `json:"from_name"`
	To        string `json:"to"`
	Text      string `json:"text"`
	CreatedAt int64  `json:"created_at"`
}
