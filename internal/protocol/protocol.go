package protocol

import "encoding/json"

// MessageType คือชนิดของข้อความที่รับส่งกันระหว่าง client กับ server
// ใช้ string ธรรมดาเพื่อให้อ่าน JSON ได้ง่ายและ debug สะดวก
type MessageType string

const (
	TypeJoin         MessageType = "join"          // ผู้เล่นเข้ามาในห้อง (บอกชื่อ)
	TypeMove         MessageType = "move"          // ผู้เล่นเคลื่อนที่ (บอกตำแหน่งใหม่)
	TypeChat         MessageType = "chat"          // ผู้เล่นพิมพ์แชต
	TypeLeave        MessageType = "leave"         // ผู้เล่นออกจากห้อง (ปกติ server เป็นคนสร้าง)
	TypeWelcome      MessageType = "welcome"       // server บอก client ว่า "id ของคุณคืออะไร" ทันทีที่ต่อ
	TypeSignal       MessageType = "signal"        // WebRTC signaling — ส่งต่อระหว่าง peer 2 คนแบบเจาะจง
	TypeCallInvite   MessageType = "call_invite"   // ชวนเข้าสาย (relay unicast เหมือน signal)
	TypeCallAccept   MessageType = "call_accept"   // ตอบรับคำเชิญ
	TypeCallReject   MessageType = "call_reject"   // ปฏิเสธคำเชิญ
	TypeCallCancel   MessageType = "call_cancel"   // ผู้ชวนยกเลิกก่อนตอบ
	TypeSwitchRoom   MessageType = "switch_room"   // client ขอย้ายห้องบน connection เดิม (ไม่ reconnect)
	TypePresence     MessageType = "presence"      // server แจ้งสถานะ online/in_call ของ user (ขาออก)
	TypeCallStatus   MessageType = "call_status"   // client รายงานสถานะกล้องตัวเอง online↔in-call (ขาเข้า)
	TypeSpriteChange MessageType = "sprite_change" // client เปลี่ยนตัวละคร → relay ทั้งห้อง
	TypeObject       MessageType = "object"
	TypeBoardCreate  MessageType = "board_create"
	TypeBoardRename  MessageType = "board_rename"
	TypeBoardDelete  MessageType = "board_delete"
	TypeTaskCreate   MessageType = "task_create"
	TypeTaskMove     MessageType = "task_move"
	TypeTaskUpdate   MessageType = "task_update"
	TypeTaskDelete   MessageType = "task_delete"
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
	Name   string `json:"name"`
	Sprite string `json:"sprite,omitempty"` // ตัวละครที่เลือก (ว่าง → คงของเดิม/ดีฟอลต์)
}

// SpriteChangePayload : เปลี่ยนตัวละครระหว่างเล่น
//   - ขาเข้า : client ส่งแค่ Sprite
//   - ขาออก : server เติม ID (ผู้ส่ง) แล้ว broadcast ทั้งห้อง
type SpriteChangePayload struct {
	ID     string `json:"id,omitempty"`
	Sprite string `json:"sprite"`
}

// PresencePayload : สถานะของ user 1 คน (server → ทุก client + snapshot ตอน join)
//   - Online : มี connection อยู่ในระบบไหม (ข้ามห้อง)
//   - InCall : กำลังเปิดสายวิดีโออยู่ไหม (busy)
type PresencePayload struct {
	ID     string `json:"id"`
	Online bool   `json:"online"`
	InCall bool   `json:"in_call"`
	Room   string `json:"room,omitempty"` // ห้องปัจจุบัน (ข้ามห้อง) — frontend อัปเดต playerRooms
}

// CallStatusPayload : client รายงานว่าตอนนี้ตัวเอง in-call ไหม (เปิด/ปิดกล้อง)
type CallStatusPayload struct {
	InCall bool `json:"in_call"`
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

// ChatPayload : ข้อความแชตขาเข้า
//   - Scope : "room" (ห้องนี้, ดีฟอลต์ถ้าว่าง) | "all" (ทั้งหมด) | "private" (ส่วนตัว)
//   - To    : userId ปลายทาง (ใช้เฉพาะ scope="private")
type ChatPayload struct {
	Scope string `json:"scope,omitempty"`
	To    string `json:"to,omitempty"`
	Text  string `json:"text"`
}

// LeavePayload : ใครออกจากห้อง (server ใส่ id ให้)
type LeavePayload struct {
	ID string `json:"id"`
}

// ChatBroadcast : ข้อความแชตที่ server ส่งต่อ — "ขาออก" บอกด้วยว่าใครพูด + scope ไหน
//   - frontend ใช้ Scope จัดเข้าแท็บถูก (ห้องนี้/ทั้งหมด/ส่วนตัว)
//   - private: ID=ผู้ส่ง, To=ปลายทาง → คู่สนทนาคืออีกฝั่งของ (ID,To)
type ChatBroadcast struct {
	Mid   string `json:"mid"` // message id — frontend dedupe (ข้อความ live ที่เคยรับ vs ที่มาซ้ำในประวัติ)
	Ts    int64  `json:"ts"`  // unix seconds — เรียงลำดับ/แสดงเวลา
	Scope string `json:"scope"`
	ID    string `json:"id"`   // id ผู้ส่ง
	Name  string `json:"name"` // ชื่อผู้ส่ง
	To    string `json:"to,omitempty"`
	Text  string `json:"text"`
}

// WelcomePayload : server ส่งให้ client ทันทีที่ต่อ เพื่อบอก id ที่ถูกแจกให้
// frontend ใช้ id นี้แยกว่า "ตัวไหนคือเรา" (เช่น ไฮไลต์ตัวเอง / ส่ง move ของเรา)
// X,Y = ตำแหน่ง spawn ที่ server กำหนดให้ (กู้จาก Redis ถ้าเคยเล่น, ไม่งั้นสุ่ม)
// → client เกิดที่ตำแหน่งเดิมหลัง refresh/reconnect แทนที่จะ spawn ใหม่ทุกครั้ง
type WelcomePayload struct {
	ID     string `json:"id"`
	Room   string `json:"room"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Sprite string `json:"sprite,omitempty"` // ตัวละครเดิม (กู้จาก Redis) → reconnect แล้วได้ตัวเดิม
}

// ----- board -----
// board_create ขาเข้า: client ส่งแค่ name (server แจก id) / ขาออก: ส่ง domain.Board (มี id+name)
type BoardCreatePayload struct {
	Name string `json:"name"`
}
type BoardRenamePayload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type BoardDeletePayload struct {
	ID string `json:"id"`
}

type TaskCreatePayload struct {
	BoardID  string   `json:"board_id"` // task สร้างใต้บอร์ดไหน
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
