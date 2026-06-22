// Package presence = ที่เก็บ "ตำแหน่งล่าสุดของ client" แบบถาวร (เช่น Redis)
// เพื่อให้ reconnect / refresh / logout แล้วกลับมา ยังยืนอยู่ที่เดิม
//
// router เป็นคนเรียก: ตอน join → Load มาวางตำแหน่งเดิม, ตอนขยับ → Save ทับ
// แยกเป็น interface → ไม่มี Redis (dev) ก็ใช้ Noop ได้ โดย router ไม่ต้องรู้เรื่อง
package presence

import "github/minyjae/catice/internal/room"

// Store = สัญญาของที่เก็บตำแหน่ง (impl: Redis หรือ Noop)
type Store interface {
	// Load อ่านตำแหน่งล่าสุดของ client (ok=false ถ้าไม่เคยเก็บ/หมดอายุ)
	Load(roomName, id string) (room.Player, bool)
	// Save บันทึกตำแหน่งล่าสุด (write-through ตอน client ขยับ) — ไม่ควร block นาน
	Save(roomName string, p room.Player)
}

// Noop = ไม่เก็บอะไรเลย (ใช้ตอนไม่ได้ตั้ง REDIS_URL) → พฤติกรรมเหมือนเดิม (spawn ใหม่ทุกครั้ง)
type Noop struct{}

func (Noop) Load(string, string) (room.Player, bool) { return room.Player{}, false }
func (Noop) Save(string, room.Player)                {}
