package presence

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github/minyjae/catice/internal/room"
)

// posTTL = อายุของตำแหน่งใน Redis — กันข้อมูลค้างถาวรถ้า user ไม่กลับมาอีก
const posTTL = 7 * 24 * time.Hour

// flushInterval = ความถี่ในการเท pending ลง Redis (throttle)
// ลด = สดกว่าแต่ write ถี่ขึ้น / เพิ่ม = write น้อยลงแต่ตำแหน่งหน่วงนานขึ้น
const flushInterval = 1 * time.Second

// Redis เก็บตำแหน่ง client ลง Redis (key: "pos:{room}:{id}" → JSON ของ Player)
//
// Save เป็น async ผ่าน channel + goroutine เดียว → router (single goroutine) ไม่ต้องรอ Redis
// writeLoop ทำ coalesce (เก็บแค่ตำแหน่งล่าสุดต่อ client) + เท Redis เป็นรอบทุก flushInterval
// เป็น pipeline เดียว → ลดจำนวน write ลง Redis อย่างมากตอน client ขยับถี่/เยอะ
type Redis struct {
	rdb   *redis.Client
	saves chan savePos
}

type savePos struct {
	room string
	p    room.Player
}

// NewRedis สร้าง store + start goroutine เขียนเบื้องหลัง
func NewRedis(rdb *redis.Client) *Redis {
	s := &Redis{rdb: rdb, saves: make(chan savePos, 1024)}
	go s.writeLoop()
	return s
}

func key(roomName, id string) string { return "pos:" + roomName + ":" + id }

// Load อ่านตำแหน่งล่าสุด (sync — เรียกแค่ตอน join ซึ่งไม่ถี่)
func (s *Redis) Load(roomName, id string) (room.Player, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	b, err := s.rdb.Get(ctx, key(roomName, id)).Bytes()
	if err != nil {
		return room.Player{}, false // รวม redis.Nil (ไม่เคยเก็บ) → ไม่เจอ
	}
	var p room.Player
	if json.Unmarshal(b, &p) != nil {
		return room.Player{}, false
	}
	return p, true
}

// Save โยนงานเขียนเข้า channel แล้วคืนทันที (ไม่บล็อก router)
//   - buffer เต็ม → ทิ้ง (move ครั้งถัดไปจะ save ทับเองอยู่แล้ว → ไม่เสียหาย)
func (s *Redis) Save(roomName string, p room.Player) {
	select {
	case s.saves <- savePos{room: roomName, p: p}:
	default:
	}
}

// writeLoop coalesce ตำแหน่งล่าสุดต่อ client ลง pending แล้วเท Redis เป็นรอบทุก flushInterval
//   - case sv  : เขียนทับ key เดิม → ขยับ 10 ครั้งใน 1 วิ เหลือ entry เดียว (ตำแหน่งล่าสุด)
//   - case tick: เท pending ทั้งหมดเป็น pipeline เดียว แล้วเริ่มรอบใหม่
func (s *Redis) writeLoop() {
	pending := make(map[string]savePos) // key = Redis key "pos:{room}:{id}"
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case sv := <-s.saves:
			pending[key(sv.room, sv.p.ID)] = sv // coalesce
		case <-ticker.C:
			if len(pending) == 0 {
				continue
			}
			s.flush(pending)
			pending = make(map[string]savePos) // เคลียร์รอบใหม่
		}
	}
}

// flush เท pending ทั้งหมดลง Redis ใน round-trip เดียวด้วย pipeline
func (s *Redis) flush(pending map[string]savePos) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pipe := s.rdb.Pipeline()
	for k, sv := range pending {
		b, err := json.Marshal(sv.p)
		if err != nil {
			continue
		}
		pipe.Set(ctx, k, b, posTTL) // k เป็น Redis key อยู่แล้ว
	}
	pipe.Exec(ctx)
}
