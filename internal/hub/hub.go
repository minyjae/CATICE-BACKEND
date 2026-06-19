// Package hub ดูแล "การเชื่อมต่อ" ล้วน ๆ: ใครต่ออยู่ ใครอยู่ห้องไหน ส่ง byte ออกไป
// ไม่รู้จัก protocol/game logic เลย → เป็นชั้นล่างสุด (transport)
//
// ปม import cycle: ถ้า hub เรียก router ตรง ๆ จะวน (router ก็เรียก hub)
// แก้โดย hub แค่ "เปิดช่อง" Incoming/Events ไว้ → router มาดูดเอง (hub ไม่รู้จัก router)
package hub

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// Inbound คือข้อความดิบจาก client 1 ชิ้น (hub ไม่แกะ แค่แปะว่าใคร/ห้องไหน)
type Inbound struct {
	ClientID string
	Room     string
	Data     []byte
}

// EventKind = ชนิดเหตุการณ์วงจรชีวิตของ client
type EventKind int

const (
	Joined EventKind = iota // client ต่อเข้ามา + เข้าห้องแล้ว
	Left                    // client หลุด/ออกไปแล้ว
)

// Event แจ้ง router ว่ามีคนเข้า/ออก (router จะไปทำ snapshot / broadcast leave)
type Event struct {
	Kind     EventKind
	ClientID string
	Room     string
}

// outbound = คำสั่งส่งออก: id=="" → ทั้งห้อง, ไม่งั้น → คนเดียวเจาะจง
type outbound struct {
	room string
	id   string
	data []byte
}

type Hub struct {
	rooms      map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	out        chan outbound
	incoming   chan Inbound
	events     chan Event
}

// New สร้าง Hub — channel incoming/events/out ใช้ buffer เพื่อกัน deadlock
// (router ส่งเข้า out ขณะ hub ส่งออก events พร้อมกัน — buffer ช่วยให้ไม่ค้างรอกันเป็นวง)
func New() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		out:        make(chan outbound, 1024),
		incoming:   make(chan Inbound, 256),
		events:     make(chan Event, 64),
	}
}

// ---------- ช่องให้ router มาดูด ----------

func (h *Hub) Incoming() <-chan Inbound { return h.incoming }
func (h *Hub) Events() <-chan Event     { return h.events }

// ---------- คำสั่งให้ hub ส่งออก ----------

// Broadcast ส่งให้ทุกคนในห้อง
func (h *Hub) Broadcast(room string, data []byte) {
	h.out <- outbound{room: room, data: data}
}

// SendTo ส่งให้คนเดียวเจาะจง (ตาม id) ในห้อง
func (h *Hub) SendTo(room, id string, data []byte) {
	h.out <- outbound{room: room, id: id, data: data}
}

// Run คือ loop หลัก (goroutine เดียวที่แตะ rooms → ไม่ต้อง lock)
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			if h.rooms[c.room] == nil {
				h.rooms[c.room] = make(map[*Client]bool)
			}
			// 1 user = 1 connection: ถ้า user คนนี้มีสายเก่าค้างอยู่ (refresh เร็ว / เปิด 2 แท็บ)
			// เตะสายเก่าออกก่อน — เอาออกจาก membership + ปิด send
			// (สายเก่าจะตายเอง และ unregister ของมันกลายเป็น no-op เพราะไม่อยู่ใน membership แล้ว
			//  → ไม่ยิง Left ผิด ๆ และ id ใน membership ไม่ซ้ำ → SendTo ส่งถูกสาย)
			for old := range h.rooms[c.room] {
				if old.id == c.id {
					delete(h.rooms[c.room], old)
					close(old.send)
				}
			}
			h.rooms[c.room][c] = true
			h.events <- Event{Kind: Joined, ClientID: c.id, Room: c.room}

		case c := <-h.unregister:
			if set, ok := h.rooms[c.room]; ok {
				if _, ok := set[c]; ok {
					delete(set, c)
					close(c.send)
					if len(set) == 0 {
						delete(h.rooms, c.room)
					}
					h.events <- Event{Kind: Left, ClientID: c.id, Room: c.room}
				}
			}

		case o := <-h.out:
			set := h.rooms[o.room]
			if o.id == "" {
				for c := range set {
					c.send <- o.data
				}
			} else {
				for c := range set {
					if c.id == o.id {
						c.send <- o.data
						break
					}
				}
			}
		}
	}
}

// upgrader แปลง HTTP → WebSocket
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // dev: รับทุก origin
}

// ServeWs จัดการ request /ws: upgrade → สร้าง Client → register → start pumps
//
// userID มาจาก auth (main แกะ JWT แล้วยื่นให้) → ใช้เป็น client.id คงที่
// → refresh แล้ว id ไม่เปลี่ยน (hub ไม่ต้องรู้จัก auth/JWT เลย แค่รับ id ที่ยืนยันแล้ว)
func ServeWs(h *Hub, w http.ResponseWriter, r *http.Request, userID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	room := r.URL.Query().Get("room")
	if room == "" {
		room = "lobby"
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
		id:   userID,
		room: room,
	}

	h.register <- client
	go client.writePump()
	go client.readPump()
}
