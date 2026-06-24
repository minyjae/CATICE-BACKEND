package hub

import (
	"testing"
	"time"
)

// helper: รอรับ lifecycle event (Joined/Left) ถัดไป โดยข้าม presence (Online/Offline)
// — เทสต์ชุดนี้สนใจแค่ Joined/Left; presence ถูกเพิ่มทีหลังและไหลปนมาใน channel เดียวกัน
func recvEvent(t *testing.T, h *Hub) Event {
	t.Helper()
	for {
		select {
		case ev := <-h.events:
			if ev.Kind == Online || ev.Kind == Offline {
				continue
			}
			return ev
		case <-time.After(time.Second):
			t.Fatal("ไม่ได้รับ event")
			return Event{}
		}
	}
}

// register → ต้องได้ Joined event, Broadcast → คนในห้องได้รับ
func TestRegisterAndBroadcast(t *testing.T) {
	h := New()
	go h.Run()

	a := &Client{hub: h, id: "a", room: "lobby", send: make(chan []byte, 8)}
	h.register <- a

	if ev := recvEvent(t, h); ev.Kind != Joined || ev.ClientID != "a" {
		t.Fatalf("ควรได้ Joined ของ a ได้ %+v", ev)
	}

	h.Broadcast("lobby", []byte("hi"))
	select {
	case got := <-a.send:
		if string(got) != "hi" {
			t.Fatalf("ได้ข้อความผิด: %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("a ควรได้รับ broadcast")
	}
}

// SendTo → เฉพาะคนเป้าหมายได้รับ, unregister → ได้ Left event
func TestSendToAndLeave(t *testing.T) {
	h := New()
	go h.Run()

	a := &Client{hub: h, id: "a", room: "lobby", send: make(chan []byte, 8)}
	b := &Client{hub: h, id: "b", room: "lobby", send: make(chan []byte, 8)}
	h.register <- a
	recvEvent(t, h) // Joined a
	h.register <- b
	recvEvent(t, h) // Joined b

	h.SendTo("lobby", "b", []byte("only-b"))

	select {
	case got := <-b.send:
		if string(got) != "only-b" {
			t.Fatalf("b ได้ผิด: %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("b ควรได้รับ")
	}
	select {
	case got := <-a.send:
		t.Fatalf("a ไม่ควรได้ (ส่งเจาะจง b) แต่ได้: %q", got)
	case <-time.After(100 * time.Millisecond):
		// ผ่าน
	}

	// unregister b → ได้ Left
	h.unregister <- b
	if ev := recvEvent(t, h); ev.Kind != Left || ev.ClientID != "b" {
		t.Fatalf("ควรได้ Left ของ b ได้ %+v", ev)
	}
}

// 1 user = 1 connection: เปิดสายใหม่ด้วย id เดิม → สายเก่าต้องถูกเตะ (send ถูกปิด)
// กันบั๊ก: id ซ้ำใน membership → SendTo ส่งผิดสาย + unregister ลบ state ผิด
func TestRegisterKicksDuplicateUser(t *testing.T) {
	h := New()
	go h.Run()

	c1 := &Client{hub: h, id: "u1", room: "lobby", send: make(chan []byte, 8)}
	h.register <- c1
	recvEvent(t, h) // Joined c1

	c2 := &Client{hub: h, id: "u1", room: "lobby", send: make(chan []byte, 8)} // id เดิม
	h.register <- c2
	recvEvent(t, h) // Joined c2 (หลังเตะ c1 แล้ว — close อยู่ก่อน emit event ใน goroutine เดียวกัน)

	// c1.send ต้องถูกปิด (ถูกเตะ) → อ่านได้ ok=false
	select {
	case _, ok := <-c1.send:
		if ok {
			t.Fatal("c1.send ไม่ควรมีข้อมูล — ควรถูกปิด")
		}
	case <-time.After(time.Second):
		t.Fatal("c1.send ควรถูกปิดหลังเปิดสายใหม่ id เดิม")
	}
}
