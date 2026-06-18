package protocol

import (
	"encoding/json"
	"testing"
)

// ทดสอบ: สร้าง move envelope → แกะกลับ → ต้องได้ค่าเดิม
func TestMoveRoundTrip(t *testing.T) {
	// 1) ฝั่งส่ง: สร้าง envelope ชนิด move ที่ตำแหน่ง (3, 7)
	data, err := NewEnvelope(TypeMove, MovePayload{X: 3, Y: 7})
	if err != nil {
		t.Fatalf("NewEnvelope error: %v", err)
	}

	// 2) ฝั่งรับ: แกะซองชั้นนอกก่อน
	env, err := ParseEnvelope(data)
	if err != nil {
		t.Fatalf("ParseEnvelope error: %v", err)
	}
	if env.Type != TypeMove {
		t.Fatalf("type ไม่ตรง: อยากได้ %q ได้ %q", TypeMove, env.Type)
	}

	// 3) อ่าน Type ได้แล้วว่าเป็น move → ค่อยแกะ Payload เป็น MovePayload
	var move MovePayload
	if err := json.Unmarshal(env.Payload, &move); err != nil {
		t.Fatalf("unmarshal payload error: %v", err)
	}
	if move.X != 3 || move.Y != 7 {
		t.Fatalf("ตำแหน่งไม่ตรง: อยากได้ (3,7) ได้ (%d,%d)", move.X, move.Y)
	}
}

// ทดสอบ JSON ที่ส่งมาหน้าตาตรงตามที่ออกแบบ (frontend ต้องอ่านได้)
func TestEnvelopeJSONShape(t *testing.T) {
	data, _ := NewEnvelope(TypeChat, ChatPayload{Text: "hi"})
	want := `{"type":"chat","payload":{"text":"hi"}}`
	if string(data) != want {
		t.Fatalf("รูปแบบ JSON ไม่ตรง:\n อยากได้ %s\n ได้      %s", want, data)
	}
}
