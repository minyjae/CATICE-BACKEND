# Catice Backend

Go backend สำหรับ **Catice** — แอป virtual office แบบ realtime ที่ผู้ใช้งานสามารถเดินเป็น avatar ในห้องต่าง ๆ, คุยแชท, ประชุมผ่านวิดีโอ, จัดการ Kanban board และระบบ HR ได้ในที่เดียว

> Frontend (React + TypeScript) อยู่ที่ repo แยก: `CATICE-FRONTEND`

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.22+ |
| REST API | `net/http` (standard library) |
| WebSocket | `gorilla/websocket` |
| Database | PostgreSQL (via GORM) |
| Cache / Position store | Redis (optional) |
| Auth | JWT HS256 (stateless) |
| Container | Docker + Docker Compose |

---

## Features

### Realtime (WebSocket `/ws`)
- **Avatar movement** — ผู้เล่นเดินในห้อง, เปลี่ยน sprite (player / adventurer / soldier / cat)
- **Room management** — เข้าห้อง, ย้ายห้องบน socket เดิมโดยไม่ต้องต่อใหม่
- **Chat** — ส่งข้อความ 3 โหมด: ห้องปัจจุบัน, ทั้งหมด, หรือ DM ส่วนตัว (บันทึก + replay ประวัติตอน join)
- **Presence** — แสดงสถานะ online / offline / in-call แบบ cross-room
- **Kanban board** — สร้าง/เปลี่ยนชื่อ/ลบ board, สร้าง/ย้าย/แก้ไข/ลบ task แบบ realtime
- **Objects** — วางวัตถุตกแต่งในห้อง (in-memory)
- **WebRTC signaling** — relay offer/answer/ICE candidate + call invite/accept/reject/cancel

### REST API
- **Auth** — register, login, logout (stateless JWT), ดูโปรไฟล์ตัวเอง (`/me`)
- **HR: วันหยุดบริษัท** — ดู/เพิ่ม/ลบ public holiday (HR only สำหรับแก้ไข)
- **HR: คำขอลา** — ยื่นลา (vacation/sick/personal), ดูรายการ, อนุมัติ/ปฏิเสธ พร้อม **quota ต่อปี**
- **HR: Work From Home** — ยื่น WFH รายวัน, ดูรายการ, อนุมัติ/ปฏิเสธ พร้อม **quota รายสัปดาห์/เดือน**
- **HR: Leave Policy** — HR ตั้งโควต้าวันลาและ WFH ของบริษัทได้ผ่าน API
- **HR: จัดการพนักงาน** — ดูข้อมูลเต็ม (รวมเงินเดือน), แก้ไขโปรไฟล์, เปลี่ยนตำแหน่ง, ลบพนักงาน (soft delete)
- **Daily Diary** — บันทึกงานประจำวัน, HR/manager ดูของลูกทีมได้

---

## Installation

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/) + Docker Compose
- (Optional) Redis — สำหรับ persist ตำแหน่ง avatar ข้าม restart

### 1. Clone repository

```bash
git clone <repo-url>
cd CATICE-BACKEND
```

### 2. ตั้งค่า Environment Variables

สร้างไฟล์ `.env` หรือ export ตัวแปรดังนี้:

```env
# Required
DATABASE_URL=postgres://user:password@localhost:5432/catice?sslmode=disable

# Optional (default: random string ถ้าไม่ตั้ง — token ไม่ valid ข้าม restart)
JWT_SECRET=your-secret-key

# Optional — ถ้าไม่ตั้ง ตำแหน่ง avatar จะไม่ถูกบันทึก (spawn ใหม่ทุกครั้ง)
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=

# Optional (default: :8080)
ADDR=:8080
```

### 3. รันด้วย Docker Compose (แนะนำ)

```bash
docker compose up -d --build
```

> **สำคัญ:** ต้องใส่ `--build` เสมอ ไม่งั้น Docker จะใช้ image เก่าที่ cache ไว้

เข้าถึง backend ได้ที่ `http://localhost:8080`

### 4. รันแบบ Local (สำหรับ development)

```bash
# ต้องมี Postgres รันอยู่ก่อน
go run ./cmd/catice
```

---

## Development Commands

```bash
go build ./...          # build ทุก package
go vet ./...            # ตรวจ code quality
go test ./...           # รัน tests ทั้งหมด
go test ./internal/hub/ -run TestName -v  # รัน test เดียว
gofmt -w <file>         # format ไฟล์ (ทำหลัง edit ก่อน build)
```

> `go test -race` ใช้ไม่ได้ในโปรเจคนี้ (race detector ต้องการ cgo/gcc)

---

## REST API Reference

Auth header ทุก endpoint (ยกเว้น `/register`, `/login`): `Authorization: Bearer <token>`

### Auth

| Method | Path | Description |
|---|---|---|
| `POST` | `/register` | สมัครสมาชิก → ได้ JWT กลับมาทันที |
| `POST` | `/login` | เข้าสู่ระบบ → JWT |
| `POST` | `/logout` | ออกจากระบบ (stateless: client ลบ token เอง) |
| `GET` | `/me` | ดูโปรไฟล์ตัวเอง |
| `GET` | `/users` | รายชื่อ user ทั้งหมด (สำหรับ task assignment) |
| `PATCH` | `/users/{id}/manager` | ตั้ง/เคลียร์หัวหน้าของ user (HR only) |

**Register/Login body:**
```json
{ "email": "user@example.com", "password": "secret", "role": "developer" }
```

**Roles:** `developer` `hr` `pm` `po` `cto` `uxui`

---

### HR: จัดการพนักงาน (HR only)

| Method | Path | Description |
|---|---|---|
| `GET` | `/hr/users` | รายชื่อพนักงานทั้งหมด + ข้อมูลเต็ม (รวมเงินเดือน) |
| `GET` | `/hr/users/{id}` | ข้อมูลพนักงานคนเดียว |
| `PATCH` | `/hr/users/{id}` | แก้ไขโปรไฟล์ (partial update) |
| `PATCH` | `/hr/users/{id}/role` | เปลี่ยนตำแหน่ง |
| `DELETE` | `/hr/users/{id}` | ลบพนักงานออกจากระบบ (soft delete) |

**Update profile body** (ส่งเฉพาะ field ที่อยากแก้):
```json
{
  "first_name": "สมชาย",
  "last_name": "ใจดี",
  "phone": "0812345678",
  "birth_date": "1995-03-15",
  "address": "กรุงเทพฯ",
  "salary": 55000,
  "start_date": "2023-01-01"
}
```

---

### HR: วันหยุดบริษัท

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/holidays` | ทุกคน | รายการวันหยุดทั้งหมด |
| `POST` | `/holidays` | HR only | เพิ่มวันหยุด `{ "name": "...", "date": "YYYY-MM-DD" }` |
| `DELETE` | `/holidays/{id}` | HR only | ลบวันหยุด |

---

### HR: คำขอลา

| Method | Path | Description |
|---|---|---|
| `POST` | `/leaves` | ยื่นคำขอลา |
| `GET` | `/leaves/mine` | คำขอลาของตัวเอง |
| `GET` | `/leaves/pending` | คำขอที่รอฉันอนุมัติ |
| `POST` | `/leaves/{id}/approve` | อนุมัติคำขอ |
| `POST` | `/leaves/{id}/reject` | ปฏิเสธคำขอ |

**Create leave body:**
```json
{
  "type": "vacation",
  "start_date": "2026-08-01",
  "end_date": "2026-08-03",
  "reason": "ท่องเที่ยว"
}
```

**Leave types:** `vacation` (พักร้อน) / `sick` (ป่วย) / `personal` (กิจส่วนตัว)

ระบบจะ **reject 422** อัตโนมัติถ้าวันลาเกินโควต้าที่บริษัทตั้งไว้

---

### HR: Work From Home

| Method | Path | Description |
|---|---|---|
| `POST` | `/wfh` | ยื่น WFH `{ "date": "YYYY-MM-DD", "reason": "..." }` |
| `GET` | `/wfh/mine` | คำขอ WFH ของตัวเอง |
| `GET` | `/wfh/pending` | คำขอที่รอฉันอนุมัติ |
| `POST` | `/wfh/{id}/approve` | อนุมัติ |
| `POST` | `/wfh/{id}/reject` | ปฏิเสธ |

ระบบเช็ค quota รายสัปดาห์และรายเดือนอัตโนมัติ

---

### HR: Leave Policy (HR only)

| Method | Path | Description |
|---|---|---|
| `GET` | `/policy` | ดู policy ปัจจุบัน |
| `PUT` | `/policy` | แก้ไข policy |

```json
{
  "vacation_days_per_year": 10,
  "sick_days_per_year": 30,
  "personal_days_per_year": 3,
  "wfh_days_per_week": 2,
  "wfh_days_per_month": 8
}
```

---

### Daily Diary

| Method | Path | Description |
|---|---|---|
| `POST` | `/diary` | บันทึก/อัปเดตไดอารี่วันนี้ `{ "date": "YYYY-MM-DD", "content": "..." }` |
| `GET` | `/diary/mine` | ไดอารี่ทั้งหมดของตัวเอง |
| `GET` | `/diary?user_id=&date=` | ดูไดอารี่ของลูกทีม (HR/manager) |

---

## WebSocket API

เชื่อมต่อที่ `ws://localhost:8080/ws?token=<jwt>&room=<room-name>`

> Browser ตั้ง Authorization header บน WebSocket handshake ไม่ได้ → ส่ง token ผ่าน query string

### Message Format

```json
{ "type": "<message-type>", "payload": { ... } }
```

### Message Types (Client → Server)

| Type | Payload | Description |
|---|---|---|
| `join` | `{ name, sprite }` | ประกาศเข้าห้อง |
| `move` | `{ x, y }` | อัปเดตตำแหน่ง avatar |
| `switch_room` | `{ room }` | ย้ายห้องบน socket เดิม |
| `chat` | `{ scope, text, to? }` | ส่งข้อความ (scope: room/all/private) |
| `call_status` | `{ inCall }` | แจ้งสถานะ in-call |
| `sprite_change` | `{ sprite }` | เปลี่ยน avatar sprite |
| `signal` | `{ to, data }` | WebRTC offer/answer/ICE relay |
| `call_invite` | `{ to }` | เชิญประชุมวิดีโอ |
| `call_accept` | `{ to }` | ยอมรับการเชิญ |
| `call_reject` | `{ to }` | ปฏิเสธการเชิญ |
| `call_cancel` | `{ to }` | ยกเลิกการเชิญ |
| `object` | `{ name, x, y }` | วางวัตถุในห้อง |
| `board_create` | `{ name }` | สร้าง Kanban board |
| `board_rename` | `{ id, name }` | เปลี่ยนชื่อ board |
| `board_delete` | `{ id }` | ลบ board (cascade ลบ task ด้วย) |
| `task_create` | `{ boardId, title, detail, assignTo[] }` | สร้าง task |
| `task_move` | `{ id, status }` | ย้าย task (todo/doing/done) |
| `task_update` | `{ id, title?, detail?, assignTo? }` | แก้ไข task |
| `task_delete` | `{ id }` | ลบ task |

### Message Types (Server → Client)

| Type | Description |
|---|---|
| `welcome` | snapshot ตอน join: ตำแหน่ง, board, task, chat history, presence |
| `move` | broadcast ตำแหน่งใหม่ของผู้เล่น |
| `leave` | ผู้เล่นออกจากห้อง |
| `chat` | ข้อความ chat พร้อม `mid` (message id) สำหรับ dedup |
| `presence` | สถานะ online/offline/in-call ของ user |
| `signal` | WebRTC signaling relay |
| `call_invite/accept/reject/cancel` | สถานะ video call |
| `board_create/rename/delete` | event ของ Kanban board |
| `task_create/move/update/delete` | event ของ task |

---

## Architecture

```
cmd/catice/main.go
    ├── hub (goroutine)          ← transport layer: จัดการ WS connections
    ├── router (goroutine)       ← game/app state: dispatch messages
    │   ├── room.Manager         ← in-memory player/object state
    │   └── presence.Store       ← Redis position store
    └── HTTP handlers            ← REST API (auth, HR module)
```

**หลักสำคัญ:** state ไม่ใช้ mutex — hub goroutine เป็นเจ้าของ connection state, router goroutine เป็นเจ้าของ game state ทั้งหมด (ดูรายละเอียดใน `CLAUDE.md`)

---

## Database Tables

| Table | Description |
|---|---|
| `users` | บัญชีผู้ใช้ + โปรไฟล์พนักงาน (soft delete ด้วย `deleted_at`) |
| `boards` | Kanban boards |
| `tasks` | Tasks ใน board |
| `messages` | ประวัติ chat (room/all/private) |
| `holidays` | วันหยุดบริษัท |
| `leave_requests` | คำขอลา |
| `wfh_requests` | คำขอ WFH |
| `daily_diaries` | บันทึกงานประจำวัน |
| `leave_policy` | นโยบายโควต้าวันลา/WFH (singleton) |

ทุกตาราง auto-migrate ตอน startup

---

## Health Check

```
GET /health → "ok"
```
