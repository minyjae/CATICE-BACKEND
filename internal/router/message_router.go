// Package router คือ "ตัวสั่งการบนสุด" — รู้จักทุก package
// ดูดข้อความ/เหตุการณ์จาก hub แล้ว dispatch ไปยัง room / signaling
// (hub ไม่รู้จัก router → ไม่มี import cycle)
package router

import (
	"encoding/json"
	"math/rand"
	"strings"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/service"
	"github/minyjae/catice/internal/hub"
	"github/minyjae/catice/internal/id"
	"github/minyjae/catice/internal/presence"
	"github/minyjae/catice/internal/protocol"
	"github/minyjae/catice/internal/room"
	"github/minyjae/catice/internal/signaling"
)

type Router struct {
	hub       *hub.Hub
	rooms     *room.Manager
	tasks     *service.TaskStore  // task ลง DB ผ่าน service → ทุก client เห็นชุดเดียวกัน
	boards    *service.BoardStore // board (kanban หลายใบ) ลง DB
	chat      *service.ChatStore  // ข้อความแชต (room/all/private) ลง DB + ส่งประวัติตอน join
	positions presence.Store      // ตำแหน่ง client ถาวร (Redis) → reconnect/refresh/logout แล้วยืนที่เดิม

	// presence — แตะเฉพาะใน router goroutine เดียว → ไม่ต้อง lock (เหมือน room.Manager)
	online map[string]bool // userId → มี connection อยู่ในระบบ (ข้ามห้อง)
	inCall map[string]bool // userId → กำลังเปิดสายวิดีโอ (busy)
}

func New(h *hub.Hub, rm *room.Manager, tasks *service.TaskStore, boards *service.BoardStore, chat *service.ChatStore, positions presence.Store) *Router {
	return &Router{
		hub: h, rooms: rm, tasks: tasks, boards: boards, chat: chat, positions: positions,
		online: make(map[string]bool),
		inCall: make(map[string]bool),
	}
}

// Run วนรับ 2 อย่างจาก hub: เหตุการณ์เข้า/ออก และข้อความจาก client
func (rt *Router) Run() {
	for {
		select {
		case ev := <-rt.hub.Events():
			rt.handleEvent(ev)
		case in := <-rt.hub.Incoming():
			rt.handleMessage(in)
		}
	}
}

// ---------- เหตุการณ์เข้า/ออกห้อง ----------

func (rt *Router) handleEvent(ev hub.Event) {
	switch ev.Kind {
	case hub.Joined:
		// 1) สร้างตัวตนในเกม — เฉพาะตอน "ยังไม่มีใน memory" เท่านั้น
		//    (ถ้า reconnect/เปิดสายใหม่ของ user เดิม → คงตำแหน่งเดิมไว้ ไม่ spawn ทับ)
		if _, exists := rt.rooms.Get(ev.Room, ev.ClientID); !exists {
			// ลองกู้ตำแหน่งเดิมจาก Redis ก่อน (รอด refresh/logout/รีสตาร์ท server)
			pl, ok := rt.positions.Load(ev.Room, ev.ClientID)
			if !ok {
				// ไม่เคยมี → สุ่มจุดเกิด (y เริ่มที่ 2 ไม่ให้เกิดทับผนังบน 2 แถวที่ client กันเดิน)
				pl = room.Player{ID: ev.ClientID, X: rand.Intn(10), Y: 2 + rand.Intn(8)}
			}
			rt.rooms.Add(ev.Room, pl)
			rt.positions.Save(ev.Room, pl) // เก็บไว้ (เผื่อ spawn ใหม่ ให้ถาวรตั้งแต่ครั้งแรก)
		}

		// 2) บอก client ว่า id ตัวเองคืออะไร + ตำแหน่ง spawn (welcome)
		//    me = ตำแหน่งที่เพิ่งกู้จาก Redis / สุ่มไว้ด้านบน → ส่งให้ client เกิดที่เดิม
		me, _ := rt.rooms.Get(ev.Room, ev.ClientID)
		if out, err := protocol.NewEnvelope(protocol.TypeWelcome, protocol.WelcomePayload{
			ID: ev.ClientID, Room: ev.Room, X: me.X, Y: me.Y,
		}); err == nil {
			rt.hub.SendTo(ev.Room, ev.ClientID, out)
		}

		// 3) initial state sync: ส่งตำแหน่งคนเก่าทั้งห้อง ให้คนใหม่คนเดียว
		for _, other := range rt.rooms.Others(ev.Room, ev.ClientID) {
			if out, err := protocol.NewEnvelope(protocol.TypeJoin, other); err == nil {
				rt.hub.SendTo(ev.Room, ev.ClientID, out)
			}
		}

		for _, obj := range rt.rooms.Objects(ev.Room) {
			if out, err := protocol.NewEnvelope(protocol.TypeObject, room.Object{
				ID: obj.ID, Name: obj.Name, X: obj.X, Y: obj.Y,
			}); err == nil {
				rt.hub.SendTo(ev.Room, ev.ClientID, out)
			}
		}

		// snapshot board ทั้งหมดก่อน (frontend สร้างบอร์ดให้ครบ) แล้วค่อยส่ง task
		for _, b := range rt.boards.List() {
			if out, err := protocol.NewEnvelope(protocol.TypeBoardCreate, b); err == nil {
				rt.hub.SendTo(ev.Room, ev.ClientID, out)
			}
		}

		// snapshot task ทั้งหมดจาก DB → คนใหม่ (มี board_id, frontend วางลงบอร์ดตาม board_id)
		for _, t := range rt.tasks.List() {
			if out, err := protocol.NewEnvelope(protocol.TypeTaskCreate, t); err == nil {
				rt.hub.SendTo(ev.Room, ev.ClientID, out)
			}
		}

		// snapshot ประวัติแชต: ห้องนี้ + ทั้งหมด + DM ของ user นี้ (frontend แยกตาม scope + dedupe ด้วย mid)
		for _, m := range rt.chat.History(ev.Room, ev.ClientID) {
			if out, err := protocol.NewEnvelope(protocol.TypeChat, chatToBroadcast(m)); err == nil {
				rt.hub.SendTo(ev.Room, ev.ClientID, out)
			}
		}

		// snapshot presence: ใครออนไลน์/อยู่ในสายอยู่บ้าง → ให้คน join คนเดียว
		for id := range rt.online {
			if out, err := protocol.NewEnvelope(protocol.TypePresence, protocol.PresencePayload{
				ID: id, Online: true, InCall: rt.inCall[id],
			}); err == nil {
				rt.hub.SendTo(ev.Room, ev.ClientID, out)
			}
		}

	case hub.Left:
		// ลบ state + บอกทุกคนว่าคนนี้ออกไปแล้ว
		rt.rooms.Remove(ev.Room, ev.ClientID)
		if out, err := protocol.NewEnvelope(protocol.TypeLeave, protocol.LeavePayload{ID: ev.ClientID}); err == nil {
			rt.hub.Broadcast(ev.Room, out)
		}

	case hub.Online:
		rt.online[ev.ClientID] = true
		rt.broadcastPresence(ev.ClientID)

	case hub.Offline:
		delete(rt.online, ev.ClientID)
		delete(rt.inCall, ev.ClientID)
		rt.broadcastPresence(ev.ClientID)
	}
}

// broadcastPresence ส่งสถานะล่าสุดของ user คนหนึ่งให้ทุก client ทุกห้อง
func (rt *Router) broadcastPresence(id string) {
	if out, err := protocol.NewEnvelope(protocol.TypePresence, protocol.PresencePayload{
		ID: id, Online: rt.online[id], InCall: rt.inCall[id],
	}); err == nil {
		rt.hub.BroadcastAll(out)
	}
}

// ---------- ข้อความจาก client ----------

func (rt *Router) handleMessage(in hub.Inbound) {
	env, err := protocol.ParseEnvelope(in.Data)
	if err != nil {
		return
	}

	switch env.Type {
	case protocol.TypeJoin:
		var p protocol.JoinPayload
		if json.Unmarshal(env.Payload, &p) != nil {
			return
		}
		rt.updatePlayer(in.Room, in.ClientID, func(pl *room.Player) { pl.Name = p.Name }, protocol.TypeJoin)

	case protocol.TypeMove:
		var p protocol.MovePayload
		if json.Unmarshal(env.Payload, &p) != nil {
			return
		}
		rt.updatePlayer(in.Room, in.ClientID, func(pl *room.Player) { pl.X = p.X; pl.Y = p.Y }, protocol.TypeMove)

	case protocol.TypeSwitchRoom:
		var p protocol.SwitchRoomPayload
		if json.Unmarshal(env.Payload, &p) != nil {
			return
		}
		if p.Room == "" || p.Room == in.Room {
			return
		}
		// hub ย้าย membership → ยิง Left(ห้องเก่า)+Joined(ห้องใหม่) → handleEvent กู้ตำแหน่ง/sync ต่อเอง
		rt.hub.SwitchRoom(in.Room, in.ClientID, p.Room)

	case protocol.TypeChat:
		var p protocol.ChatPayload
		if json.Unmarshal(env.Payload, &p) != nil {
			return
		}
		if strings.TrimSpace(p.Text) == "" {
			return // ไม่เก็บ/ส่งข้อความว่าง
		}
		scope := p.Scope
		if scope == "" {
			scope = "room" // ดีฟอลต์
		}
		if scope == "private" && p.To == "" {
			return // private ต้องมีปลายทาง
		}
		pl, _ := rt.rooms.Get(in.Room, in.ClientID) // เอาชื่อผู้ส่งมาแนบ

		// บันทึกลง DB → ได้ id + เวลา (room เก็บเฉพาะ scope=room)
		msgRoom := ""
		if scope == "room" {
			msgRoom = in.Room
		}
		msg := rt.chat.Record(scope, msgRoom, in.ClientID, pl.Name, p.To, p.Text)

		out, err := protocol.NewEnvelope(protocol.TypeChat, chatToBroadcast(msg))
		if err != nil {
			return
		}
		switch scope {
		case "all": // ทั้งหมด → ทุกคนทุกห้อง
			rt.hub.BroadcastAll(out)
		case "private": // ส่วนตัว → ปลายทาง + echo กลับให้ผู้ส่งเห็นเอง
			rt.hub.SendToUser(p.To, out)
			rt.hub.SendToUser(in.ClientID, out)
		default: // room → เฉพาะห้องนี้
			rt.hub.Broadcast(in.Room, out)
		}

	case protocol.TypeCallStatus:
		// client รายงานสถานะกล้องตัวเอง (online ↔ in-call) → อัปเดต + broadcast presence
		var p protocol.CallStatusPayload
		if json.Unmarshal(env.Payload, &p) != nil {
			return
		}
		if p.InCall {
			rt.inCall[in.ClientID] = true
		} else {
			delete(rt.inCall, in.ClientID)
		}
		rt.broadcastPresence(in.ClientID)

	case protocol.TypeSignal:
		// relay WebRTC ให้ peer ปลายทางคนเดียว
		if toID, data, ok := signaling.Relay(in.ClientID, env.Payload); ok {
			rt.hub.SendTo(in.Room, toID, data)
		}

	case protocol.TypeCallInvite, protocol.TypeCallAccept, protocol.TypeCallReject, protocol.TypeCallCancel:
		// คุมการเชิญสาย — relay unicast ไป to (from เติมจาก ClientID/JWT); ปลายทางออฟไลน์ → SendTo เงียบ
		if toID, data, ok := signaling.RelayCall(env.Type, in.ClientID, env.Payload); ok {
			rt.hub.SendTo(in.Room, toID, data)
		}

	case protocol.TypeObject:
		var p room.Object
		if json.Unmarshal(env.Payload, &p) != nil {
			return
		}

		p.ID = id.New()

		rt.rooms.AddObject(in.Room, room.Object{ID: p.ID, Name: p.Name, X: p.X, Y: p.Y})

		if out, err := protocol.NewEnvelope(protocol.TypeObject, p); err == nil {
			rt.hub.Broadcast(in.Room, out)
		}

	case protocol.TypeBoardCreate:
		var p protocol.BoardCreatePayload
		if json.Unmarshal(env.Payload, &p) != nil {
			return
		}
		if board, err := rt.boards.Create(p.Name); err == nil {
			if out, err := protocol.NewEnvelope(protocol.TypeBoardCreate, board); err == nil {
				rt.hub.Broadcast(in.Room, out)
			}
		}

	case protocol.TypeBoardRename:
		var p protocol.BoardRenamePayload
		if json.Unmarshal(env.Payload, &p) != nil {
			return
		}
		if board, ok := rt.boards.Rename(p.ID, p.Name); ok {
			if out, err := protocol.NewEnvelope(protocol.TypeBoardRename, board); err == nil {
				rt.hub.Broadcast(in.Room, out)
			}
		}

	case protocol.TypeBoardDelete:
		var p protocol.BoardDeletePayload
		if json.Unmarshal(env.Payload, &p) != nil {
			return
		}
		rt.boards.Delete(p.ID)
		rt.tasks.DeleteByBoard(p.ID) // cascade: ลบ task ของบอร์ดด้วย
		if out, err := protocol.NewEnvelope(protocol.TypeBoardDelete, p); err == nil {
			rt.hub.Broadcast(in.Room, out)
		}

	case protocol.TypeTaskCreate:
		var p protocol.TaskCreatePayload
		if json.Unmarshal(env.Payload, &p) != nil {
			return
		}
		if !rt.boards.Exists(p.BoardID) { // กัน task ลอย (สร้างใต้บอร์ดที่ไม่มี)
			return
		}
		// createdBy = ผู้ส่ง (จาก auth/JWT) ไม่เชื่อ client; service แจก id + status "todo" + บันทึกลง DB
		task, err := rt.tasks.Create(in.ClientID, domain.CreateTaskPayload{
			BoardID: p.BoardID, Title: p.Title, Detail: p.Detail, AssignTo: p.AssignTo,
		})
		if err != nil {
			return
		}
		if out, err := protocol.NewEnvelope(protocol.TypeTaskCreate, task); err == nil {
			rt.hub.Broadcast(in.Room, out)
		}

	case protocol.TypeTaskMove:
		var p protocol.TaskMovePayload
		if json.Unmarshal(env.Payload, &p) != nil {
			return
		}
		if task, ok := rt.tasks.Move(p.ID, domain.Status(p.Status)); ok {
			if out, err := protocol.NewEnvelope(protocol.TypeTaskMove, task); err == nil {
				rt.hub.Broadcast(in.Room, out)
			}
		}

	case protocol.TypeTaskUpdate:
		var p protocol.TaskUpdatePayload
		if json.Unmarshal(env.Payload, &p) != nil {
			return
		}
		if task, ok := rt.tasks.Update(p.ID, p.Title, p.Detail, p.AssignTo); ok {
			if out, err := protocol.NewEnvelope(protocol.TypeTaskUpdate, task); err == nil {
				rt.hub.Broadcast(in.Room, out)
			}
		}

	case protocol.TypeTaskDelete:
		var p protocol.TaskDeletePayload
		if json.Unmarshal(env.Payload, &p) != nil {
			return
		}
		rt.tasks.Delete(p.ID)
		if out, err := protocol.NewEnvelope(protocol.TypeTaskDelete, p); err == nil {
			rt.hub.Broadcast(in.Room, out)
		}
	}
}

// chatToBroadcast แปลง domain.Message → payload ที่ส่งให้ client (live + ประวัติใช้ตัวเดียวกัน)
func chatToBroadcast(m domain.Message) protocol.ChatBroadcast {
	return protocol.ChatBroadcast{
		Mid: m.ID, Ts: m.CreatedAt, Scope: m.Scope,
		ID: m.FromID, Name: m.FromName, To: m.To, Text: m.Text,
	}
}

// updatePlayer: อ่าน player ปัจจุบัน → แก้ตาม mutate → เก็บกลับ → broadcast ทั้งห้อง
func (rt *Router) updatePlayer(roomName, id string, mutate func(*room.Player), typ protocol.MessageType) {
	pl, ok := rt.rooms.Get(roomName, id)
	if !ok {
		pl = room.Player{ID: id}
	}
	mutate(&pl)
	rt.rooms.Add(roomName, pl)
	rt.positions.Save(roomName, pl) // write-through → ตำแหน่ง/ชื่อล่าสุดลง Redis (async ไม่หน่วง loop)

	if out, err := protocol.NewEnvelope(typ, pl); err == nil {
		rt.hub.Broadcast(roomName, out)
	}
}
