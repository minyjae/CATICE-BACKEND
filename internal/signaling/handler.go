// Package signaling จัดการ WebRTC signaling (relay offer/answer/ICE ระหว่าง peer)
// เป็น logic บริสุทธิ์: รับ payload + ผู้ส่ง → คืน "ปลายทาง + byte ที่จะส่ง"
// ส่วนการส่งจริงเป็นหน้าที่ของ router (เรียก hub.SendTo) → signaling ไม่ต้องรู้จัก hub
package signaling

import (
	"encoding/json"

	"github/minyjae/catice/internal/protocol"
)

// Relay แกะ signal payload จาก fromID, เติม From, ห่อใหม่เป็น envelope
// คืน (ปลายทาง, byte พร้อมส่ง, ok)
func Relay(fromID string, payload json.RawMessage) (toID string, data []byte, ok bool) {
	var s Signal
	if err := json.Unmarshal(payload, &s); err != nil {
		return "", nil, false
	}
	s.From = fromID // เติมผู้ส่งเอง ปลอดภัยกว่าเชื่อ client
	out, err := protocol.NewEnvelope(protocol.TypeSignal, s)
	if err != nil {
		return "", nil, false
	}
	return s.To, out, true
}

// RelayCall แกะ to ของข้อความคุมสาย, เติม From=fromID, ห่อใหม่เป็น envelope ชนิด t (เดิม)
// ใช้กับ call_invite/accept/reject/cancel — ส่ง unicast ไป to (router เรียก hub.SendTo)
// คืน (ปลายทาง, byte พร้อมส่ง, ok=false ถ้า payload เพี้ยน/ไม่มี to)
func RelayCall(t protocol.MessageType, fromID string, payload json.RawMessage) (toID string, data []byte, ok bool) {
	var c CallPayload
	if err := json.Unmarshal(payload, &c); err != nil || c.To == "" {
		return "", nil, false
	}
	// ขาออกส่งเฉพาะ From (id ผู้ส่งจริงจาก JWT) — ไม่หลุด to/ค่าปลอมจาก client
	out, err := protocol.NewEnvelope(t, CallPayload{From: fromID})
	if err != nil {
		return "", nil, false
	}
	return c.To, out, true
}
