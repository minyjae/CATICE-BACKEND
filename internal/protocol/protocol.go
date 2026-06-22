package protocol

import "encoding/json"

// MessageType คือชนิดของข้อความที่รับส่งกันระหว่าง client กับ server
// ใช้ string ธรรมดาเพื่อให้อ่าน JSON ได้ง่ายและ debug สะดวก
type MessageType string

const (
	TypeJoin       MessageType = "join"    // ผู้เล่นเข้ามาในห้อง (บอกชื่อ)
	TypeMove       MessageType = "move"    // ผู้เล่นเคลื่อนที่ (บอกตำแหน่งใหม่)
	TypeChat       MessageType = "chat"    // ผู้เล่นพิมพ์แชต
	TypeLeave      MessageType = "leave"   // ผู้เล่นออกจากห้อง (ปกติ server เป็นคนสร้าง)
	TypeWelcome    MessageType = "welcome" // server บอก client ว่า "id ของคุณคืออะไร" ทันทีที่ต่อ
	TypeSignal     MessageType = "signal"  // WebRTC signaling — ส่งต่อระหว่าง peer 2 คนแบบเจาะจง
	TypeSwitchRoom MessageType = "switch_room" // client ขอย้ายห้องบน connection เดิม (ไม่ reconnect)
	TypeObject     MessageType = "object"
	TypeTaskCreate MessageType = "task_create"
	TypeTaskMove   MessageType = "task_move"
	TypeTaskUpdate MessageType = "task_update"
	TypeTaskDelete MessageType = "task_delete"
)

// Envelope คือ "ซองจดหมาย" ที่ห่อทุกข้อความ
// - Type    : บอกว่าเป็นข้อความชนิดไหน (ดู MessageType ด้านบน)
// - Payload : เนื้อในแบบดิบ ๆ ยังไม่แกะ จะแกะตาม Type ทีหลัง
//
// json.RawMessage = เก็บ JSON ดิบไว้ก่อน ยังไม่ parse
// ทำให้เราอ่าน Type ก่อน แล้วค่อยเลือกแกะ Payload ให้ตรงชนิด
type Envelope struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// ----- payload ของแต่ละชนิดข้อความ -----

// JoinPayload : ข้อมูลตอนเข้าห้อง
type JoinPayload struct {
	Name string `json:"name"`
}

// SwitchRoomPayload : client ขอย้ายไปห้องใหม่บน connection เดิม
// server ย้าย membership (Left ห้องเก่า + Joined ห้องใหม่) แล้วกู้ตำแหน่งห้องใหม่จาก Redis
// → หลังได้ welcome ห้องใหม่ frontend ส่ง join (ชื่อ) ซ้ำเหมือนตอนต่อครั้งแรก
type SwitchRoomPayload struct {
	Room string `json:"room"`
}

// MovePayload : ตำแหน่งใหม่ของผู้เล่น (พิกัดบน grid)
type MovePayload struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// ChatPayload : ข้อความแชต
type ChatPayload struct {
	Text string `json:"text"`
}

// LeavePayload : ใครออกจากห้อง (server ใส่ id ให้)
type LeavePayload struct {
	ID string `json:"id"`
}

// ChatBroadcast : ข้อความแชตที่ server ส่งต่อให้ทุกคน
// ต่างจาก ChatPayload (ขาเข้า มีแค่ text) ตรงที่ "ขาออก" ต้องบอกด้วยว่าใครพูด
type ChatBroadcast struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Text string `json:"text"`
}

// WelcomePayload : server ส่งให้ client ทันทีที่ต่อ เพื่อบอก id ที่ถูกแจกให้
// frontend ใช้ id นี้แยกว่า "ตัวไหนคือเรา" (เช่น ไฮไลต์ตัวเอง / ส่ง move ของเรา)
// X,Y = ตำแหน่ง spawn ที่ server กำหนดให้ (กู้จาก Redis ถ้าเคยเล่น, ไม่งั้นสุ่ม)
// → client เกิดที่ตำแหน่งเดิมหลัง refresh/reconnect แทนที่จะ spawn ใหม่ทุกครั้ง
type WelcomePayload struct {
	ID   string `json:"id"`
	Room string `json:"room"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
}

type TaskCreatePayload struct {
	Title    string   `json:"title"`
	Detail   string   `json:"detail"`
	AssignTo []string `json:"assign_to"`
}
type TaskMovePayload struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}
type TaskUpdatePayload struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Detail   string   `json:"detail"`
	AssignTo []string `json:"assign_to"`
}
type TaskDeletePayload struct {
	ID string `json:"id"`
}

// หมายเหตุ: struct ของ signal (To/From/Data) ย้ายไปอยู่ package signaling แล้ว
// protocol เก็บแค่ "ชนิดข้อความ" (TypeSignal) ส่วนรูปร่าง payload อยู่ที่ผู้ใช้งาน

// ParseEnvelope แกะ byte ดิบ (จาก socket) → Envelope
// ยังไม่แกะ Payload ลึกลงไป (รอ caller อ่าน Type ก่อน)
func ParseEnvelope(data []byte) (*Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

// NewEnvelope สร้าง Envelope จาก payload object แล้วแปลงเป็น byte พร้อมส่ง
// รับ payload เป็น any (interface{}) เพราะแต่ละชนิดมี struct ต่างกัน
func NewEnvelope(t MessageType, payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return json.Marshal(Envelope{Type: t, Payload: raw})
}
