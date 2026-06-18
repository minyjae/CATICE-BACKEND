package room

import "testing"

func TestManagerAddGetOthers(t *testing.T) {
	m := NewManager()
	m.Add("lobby", Player{ID: "a", Name: "A", X: 1, Y: 2})
	m.Add("lobby", Player{ID: "b", Name: "B", X: 3, Y: 4})

	// Get
	p, ok := m.Get("lobby", "a")
	if !ok || p.X != 1 || p.Y != 2 {
		t.Fatalf("Get a ผิด: %+v ok=%v", p, ok)
	}

	// Others(except a) → ต้องเหลือแค่ b
	others := m.Others("lobby", "a")
	if len(others) != 1 || others[0].ID != "b" {
		t.Fatalf("Others ผิด: %+v", others)
	}
}

func TestManagerRemoveCleansEmptyRoom(t *testing.T) {
	m := NewManager()
	m.Add("lobby", Player{ID: "a"})
	m.Remove("lobby", "a")

	if _, ok := m.Get("lobby", "a"); ok {
		t.Fatal("a ควรถูกลบแล้ว")
	}
	if m.rooms["lobby"] != nil {
		t.Fatal("ห้องว่างควรถูกลบทิ้ง")
	}
}
