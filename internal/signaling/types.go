package signaling

import "encoding/json"

// Signal คือข้อความ WebRTC signaling ที่ relay ระหว่าง peer 2 คน
//   - To   : id ปลายทาง (client ใส่มา)
//   - From : id ผู้ส่ง (server เติมให้ ไม่เชื่อค่าจาก client)
//   - Data : ก้อน WebRTC ดิบ (offer/answer/ICE) — server ไม่ต้องเข้าใจ แค่ส่งต่อ
type Signal struct {
	To   string          `json:"to"`
	From string          `json:"from"`
	Data json.RawMessage `json:"data"`
}
