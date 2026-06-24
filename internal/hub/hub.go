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
	Joined  EventKind = iota // client ต่อเข้ามา + เข้าห้องแล้ว
	Left                     // client หลุด/ออกไปแล้ว
	Online                   // user มี connection จริงครั้งแรก (presence — ไม่ยิงตอน switch_room)
	Offline                  // user ไม่เหลือ connection แล้ว (presence)
)

// Event แจ้ง router ว่ามีคนเข้า/ออก (router จะไปทำ snapshot / broadcast leave)
type Event struct {
	Kind     EventKind
	ClientID string
	Room     string
}

// outbound = คำสั่งส่งออก:
//   - all==true          → ทุก client ทุกห้อง (chat "ทั้งหมด")
//   - room=="" && id!=""  → user คนเดียวข้ามทุกห้อง (chat "ส่วนตัว")
//   - id==""             → ทั้งห้อง (chat "ห้องนี้" / broadcast ทั่วไป)
//   - room+id            → คนเดียวเจาะจงในห้อง
type outbound struct {
	room string
	id   string
	all  bool
	data []byte
}

// switchReq = คำขอย้ายห้องของ client คนหนึ่ง (จาก room → newRoom)
type switchReq struct {
	room    string
	id      string
	newRoom string
}

type Hub struct {
	rooms      map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	out        chan outbound
	incoming   chan Inbound
	events     chan Event
	switchCh   chan switchReq
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
		switchCh:   make(chan switchReq, 64),
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

// BroadcastAll ส่งให้ทุก client ทุกห้อง (chat "ทั้งหมด")
func (h *Hub) BroadcastAll(data []byte) {
	h.out <- outbound{all: true, data: data}
}

// SendToUser ส่งให้ user คนเดียวไม่ว่าอยู่ห้องไหน (chat "ส่วนตัว") — 1 user = 1 connection
func (h *Hub) SendToUser(id string, data []byte) {
	h.out <- outbound{id: id, data: data}
}

// SwitchRoom ย้าย client (id) จากห้อง room → newRoom บน connection เดิม
// hub จะยิง Left (ห้องเก่า) + Joined (ห้องใหม่) → router ทำ broadcast/กู้ตำแหน่งต่อเอง
func (h *Hub) SwitchRoom(room, id, newRoom string) {
	h.switchCh <- switchReq{room: room, id: id, newRoom: newRoom}
}

// userOnline = user id นี้มี connection อยู่ห้องไหนสักห้องไหม (ใช้ตัดสิน presence online/offline)
func (h *Hub) userOnline(id string) bool {
	for _, set := range h.rooms {
		for c := range set {
			if c.id == id {
				return true
			}
		}
	}
	return false
}

// Run คือ loop หลัก (goroutine เดียวที่แตะ rooms → ไม่ต้อง lock)
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			// เช็ค "เคย online ไหม" ก่อนแตะ membership — refresh จะมีสายเก่าค้าง → wasOnline=true → ไม่ยิง Online ซ้ำ
			wasOnline := h.userOnline(c.id)
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
			if !wasOnline {
				h.events <- Event{Kind: Online, ClientID: c.id} // เพิ่ง online ครั้งแรก (ไม่ใช่ refresh)
			}

		case c := <-h.unregister:
			if set, ok := h.rooms[c.room]; ok {
				if _, ok := set[c]; ok {
					delete(set, c)
					close(c.send)
					if len(set) == 0 {
						delete(h.rooms, c.room)
					}
					h.events <- Event{Kind: Left, ClientID: c.id, Room: c.room}
					if !h.userOnline(c.id) {
						h.events <- Event{Kind: Offline, ClientID: c.id} // ไม่เหลือ connection แล้ว → offline
					}
				}
			}

		case o := <-h.out:
			switch {
			case o.all: // ทุก client ทุกห้อง (chat "ทั้งหมด")
				for _, set := range h.rooms {
					for c := range set {
						c.send <- o.data
					}
				}
			case o.room == "": // user ตาม id ข้ามทุกห้อง (chat "ส่วนตัว")
				for _, set := range h.rooms {
					for c := range set {
						if c.id == o.id {
							c.send <- o.data
						}
					}
				}
			case o.id == "": // ทั้งห้อง
				for c := range h.rooms[o.room] {
					c.send <- o.data
				}
			default: // คนเดียวเจาะจงในห้อง
				for c := range h.rooms[o.room] {
					if c.id == o.id {
						c.send <- o.data
						break
					}
				}
			}

		case sw := <-h.switchCh:
			if sw.newRoom == "" || sw.newRoom == sw.room {
				continue // ห้องเดิม/ว่าง → ไม่ต้องทำ
			}
			// หา client ในห้องเก่า
			var cl *Client
			for c := range h.rooms[sw.room] {
				if c.id == sw.id {
					cl = c
					break
				}
			}
			if cl == nil {
				continue // ไม่เจอ (อาจหลุดไปก่อน) → ข้าม
			}

			// 1) ออกจากห้องเก่า → ยิง Left ให้ router broadcast ให้คนในห้องเก่า
			delete(h.rooms[sw.room], cl)
			if len(h.rooms[sw.room]) == 0 {
				delete(h.rooms, sw.room)
			}
			h.events <- Event{Kind: Left, ClientID: cl.id, Room: sw.room}

			// 2) เข้าห้องใหม่ (เตะสายเก่า id เดียวกันที่อาจค้างในห้องใหม่ก่อน เหมือน register)
			cl.setRoom(sw.newRoom)
			if h.rooms[sw.newRoom] == nil {
				h.rooms[sw.newRoom] = make(map[*Client]bool)
			}
			for old := range h.rooms[sw.newRoom] {
				if old.id == cl.id {
					delete(h.rooms[sw.newRoom], old)
					close(old.send)
				}
			}
			h.rooms[sw.newRoom][cl] = true
			h.events <- Event{Kind: Joined, ClientID: cl.id, Room: sw.newRoom}
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
