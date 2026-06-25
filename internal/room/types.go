package room

// Player คือสถานะผู้เล่นในห้อง (game state — ย้ายมาจาก Client เดิม)
// มี json tag ครบ → ส่งเป็น JSON ให้ frontend ได้เลย
type Player struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Sprite string `json:"sprite,omitempty"` // "player" | "adventurer" | "soldier"
}

// Object คือวัตถุตกแต่ง/โต้ตอบในห้อง (เผื่ออนาคต เช่น โต๊ะ ประตู โซนพิเศษ)
type Object struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
}

// Room คือ 1 ห้อง: รวมผู้เล่นและวัตถุในห้องนั้น
type Room struct {
	Name    string
	Players map[string]Player
	Objects []Object
}
