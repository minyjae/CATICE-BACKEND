# Catice2 — Gather Clone (บันทึกการพัฒนาทีละ Task)

โปรเจกต์ทำ Gather.town เวอร์ชันย่อด้วย **Go + WebSocket** โดยค่อย ๆ เพิ่มโค้ดทีละส่วน
แต่ละ task มีไฟล์บันทึกของตัวเองใน `docs/` ที่อธิบาย **โค้ดทุกบรรทัด** และ **สิ่งที่เพิ่ม/เอาออก**

## รูปแบบของไฟล์ tracking แต่ละ task
1. **เป้าหมายของ task** — ทำไปเพื่ออะไร
2. **โค้ดที่เพิ่ม/แก้** — พร้อมคำอธิบายรายบรรทัด
3. **Diff สรุป** — เพิ่มอะไร / เอาอะไรออก เทียบกับ task ก่อนหน้า
4. **เชื่อมกับ task อื่นอย่างไร**

## Roadmap

### ส่วน WebSocket (backend หลัก)
- [x] **Task 1** — เติม `Hub` ให้สมบูรณ์ → [task-01-hub.md](task-01-hub.md)
- [x] **Task 2** — `Client` + readPump / writePump → [task-02-client.md](task-02-client.md)
- [x] **Task 3** — HTTP → WebSocket upgrade handler → [task-03-handler.md](task-03-handler.md)
- [x] **Task 4** — `main.go` รันเซิร์ฟเวอร์ + ทดสอบเชื่อมต่อ → [task-04-main.md](task-04-main.md)

### ส่วน Game logic (หัวใจของ Gather)
- [x] **Task 5** — ออกแบบ message protocol (JSON: join, move, chat, leave) → [task-05-protocol.md](task-05-protocol.md)
- [x] **Task 6** — เก็บ state ผู้เล่น (id, ชื่อ, ตำแหน่ง x/y) → [task-06-player.md](task-06-player.md)
- [x] **Task 7** — broadcast การเคลื่อนที่ให้ทุกคนเห็น → [task-07-movement.md](task-07-movement.md)
- [x] **Task 8** — ระบบ room/space (แยกห้อง) → [task-08-rooms.md](task-08-rooms.md)
- [x] **+ State sync** — คนเข้าทีหลังเห็นคนเก่า (อุดช่องจาก Task 7/8) → [feature-state-sync.md](feature-state-sync.md)

### ส่วน Frontend
- [x] **Task 9** — หน้าเว็บ + canvas วาดผู้เล่น → [task-09-frontend.md](task-09-frontend.md)
- [x] **Task 10** — รับ keyboard เดินตัวละคร แล้วส่ง move → [task-10-movement.md](task-10-movement.md)
- [x] **Task 11** — (ขั้นสูง) proximity video chat → [task-11-video.md](task-11-video.md)

### Refactor (ไม่ใช่ feature — จัดบ้าน)
- 🧹 จัดโครงสร้างไฟล์เป็น Go layout (`cmd/` + `internal/`) → [refactor-file-structure.md](refactor-file-structure.md) *(หลัง Task 7)*
- 🏛️ แยกเป็น clean architecture (6 package ตามหน้าที่) → [refactor-clean-arch.md](refactor-clean-arch.md) *(หลัง Task 11)*

## โครงไฟล์ปัจจุบัน (clean architecture)
แยกตามชั้นความรับผิดชอบ: dependency ไหลทางเดียว `router → hub/room/signaling/proximity → protocol`
```
catice2/
├── go.mod
├── go.sum
├── cmd/
│   └── catice/
│       └── main.go        # entrypoint: ประกอบ hub + room + router + เสิร์ฟ web/
├── web/                   # frontend (protocol บนสาย เหมือนเดิม ไม่ได้แก้ตอน refactor)
│   ├── index.html         #   login + canvas + แชต + วิดีโอ
│   └── app.js             #   ต่อ ws, วาด canvas, แชต, เดิน, proximity video (WebRTC)
├── internal/
│   ├── protocol/          # wire format — Envelope + payload (ชั้นล่างสุด)
│   ├── proximity/         # distance → volume (pure logic + test)
│   ├── room/              # game state: Room/Player/Object + Manager
│   ├── signaling/         # WebRTC relay (Signal struct + Relay)
│   ├── hub/               # transport: connection registry + pumps + ServeWs
│   │                      #   เปิด Incoming()/Events() ให้ router ดูด (กัน import cycle)
│   └── router/            # ตัวสั่งการบนสุด: ดูด hub → dispatch ไป room/signaling
└── docs/
    ├── README.md          # ไฟล์นี้
    ├── task-01..11-*.md   # บันทึกแต่ละ task
    ├── feature-state-sync.md
    ├── refactor-file-structure.md
    └── refactor-clean-arch.md
```

> หมายเหตุ: หลัง refactor นี้ docs ของ task เก่า (เช่น task-07/08) อ้างถึงไฟล์ `internal/server/*`
> ซึ่งถูกแยกย้ายไปแล้ว — ใช้ดูแนวคิด/logic ได้ ส่วนตำแหน่งไฟล์ล่าสุดดูที่ [refactor-clean-arch.md](refactor-clean-arch.md)

## แผนผัง dependency ระหว่าง package
```
cmd/catice ──► internal/server ──► internal/protocol
   (main)         (hub/client)        (wire format)
                         └──► github.com/gorilla/websocket
```
ลูกศรชี้ทางเดียวเสมอ (ไม่มี cycle): `protocol` ไม่รู้จัก `server`, `server` ไม่รู้จัก `cmd`
