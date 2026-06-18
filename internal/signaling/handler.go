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
