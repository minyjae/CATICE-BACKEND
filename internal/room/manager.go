// Package room จัดการ "สถานะเกม" ของทุกห้อง (ใครอยู่ห้องไหน ตำแหน่งอะไร)
// แยกออกจาก hub (ที่ดูแลแค่ "การเชื่อมต่อ") → transport กับ state ไม่ปนกัน
package room

// Manager ถือสถานะทุกห้อง
//
// สำคัญ: Manager ถูกแตะโดย goroutine เดียว (router) เท่านั้น → ไม่ต้องใช้ lock
// (หลักการเดียวกับ hub เดิม: ให้เจ้าของคนเดียวจัดการ state)
type Manager struct {
	rooms map[string]*Room
}

func NewManager() *Manager {
	return &Manager{rooms: make(map[string]*Room)}
}

// get คืนห้อง (สร้างใหม่ถ้ายังไม่มี)
func (m *Manager) get(name string) *Room {
	r := m.rooms[name]
	if r == nil {
		r = &Room{Name: name, Players: make(map[string]Player)}
		m.rooms[name] = r
	}
	return r
}

// Add ใส่/อัปเดตผู้เล่นในห้อง
func (m *Manager) Add(roomName string, p Player) {
	m.get(roomName).Players[p.ID] = p
}

// Get อ่านผู้เล่นคนหนึ่ง (ok=false ถ้าไม่มี)
func (m *Manager) Get(roomName, id string) (Player, bool) {
	r := m.rooms[roomName]
	if r == nil {
		return Player{}, false
	}
	p, ok := r.Players[id]
	return p, ok
}

// Others คืนผู้เล่นในห้อง "ยกเว้น exceptID" — ใช้ทำ snapshot ให้คนใหม่
func (m *Manager) Others(roomName, exceptID string) []Player {
	r := m.rooms[roomName]
	if r == nil {
		return nil
	}
	out := make([]Player, 0, len(r.Players))
	for id, p := range r.Players {
		if id != exceptID {
			out = append(out, p)
		}
	}
	return out
}

// Remove เอาผู้เล่นออก + ลบห้องทิ้งถ้าว่าง (กัน map ค้าง)
func (m *Manager) Remove(roomName, id string) {
	r := m.rooms[roomName]
	if r == nil {
		return
	}
	delete(r.Players, id)
	if len(r.Players) == 0 {
		delete(m.rooms, roomName)
	}
}

// AddObject วางวัตถุลงห้อง (สร้างห้องถ้ายังไม่มี)
// Objects เป็น slice → ใส่ด้วย append (ต่างจาก Players ที่เป็น map)
func (m *Manager) AddObject(roomName string, object Object) {
	r := m.get(roomName)                  // หา หรือสร้างห้องถ้าไม่มี (ได้ *Room แน่ ๆ)
	r.Objects = append(r.Objects, object) // ต่อท้าย slice
}

// Objects คืนวัตถุทั้งหมดในห้อง — ใช้ทำ snapshot ให้คนที่เข้ามาทีหลัง
func (m *Manager) Objects(roomName string) []Object {
	r := m.rooms[roomName]
	if r == nil {
		return nil
	}
	return r.Objects
}
