package main

import (
	"log"
	"net/http"

	"gorm.io/gorm"

	"github/minyjae/catice/internal/auth/handler"
	"github/minyjae/catice/internal/auth/repository"
	"github/minyjae/catice/internal/auth/service"
	"github/minyjae/catice/internal/config"
	"github/minyjae/catice/internal/hub"
	"github/minyjae/catice/internal/room"
	"github/minyjae/catice/internal/router"
)

func main() {
	cfg := config.Load() // อ่านค่าตั้งจาก env (DATABASE_URL, ADDR)

	// ---- ชั้นข้อมูล: Postgres (GORM) — ต้องมี DATABASE_URL เสมอ (ไม่มี in-memory fallback) ----
	db := mustDB(cfg)
	usersRepo, err := repository.NewGormUsers(db)
	if err != nil {
		log.Fatalf("migrate ตาราง users ไม่ได้: %v", err)
	}
	tasksRepo, err := repository.NewGormTasks(db)
	if err != nil {
		log.Fatalf("migrate ตาราง tasks ไม่ได้: %v", err)
	}
	taskStore := service.NewTaskStore(tasksRepo) // task ทั้งหมดไปทาง WS (router) — ดู create/move/update/delete ใน message_router

	// ---- ชั้น transport/state (dependency ไหลทางเดียว: router → hub/room/...) ----
	h := hub.New()                     // ชั้น transport (การเชื่อมต่อ)
	go h.Run()                         //
	rm := room.NewManager()            // ชั้น state (ตำแหน่งผู้เล่น)
	rt := router.New(h, rm, taskStore) // ตัวสั่งการ: ดูด hub แล้ว dispatch; task ลง DB ผ่าน taskStore
	go rt.Run()                        //

	// ---- auth (handler) ----
	authH := handler.NewAuthHandler(service.NewStore(usersRepo), service.NewTokens(cfg.JWTSecret))
	http.HandleFunc("/register", authH.Register)
	http.HandleFunc("/login", authH.Login)
	http.HandleFunc("/logout", authH.Logout)
	http.HandleFunc("/me", authH.RequireAuth(authH.Me))       // ต้อง login ก่อน (middleware เช็ค JWT)
	http.HandleFunc("/users", authH.RequireAuth(authH.Users)) // รายชื่อ user ทั้งหมด → selector มอบหมาย task
	// task: สร้าง/ย้าย/แก้/ลบ ทั้งหมดไปทาง WS (/ws) → realtime + บันทึกลง DB (ดู internal/router/message_router.go)

	// route /ws → เช็ค JWT (ต้อง login ก่อน) → upgrade → register เข้า hub
	// browser ตั้ง header บน WebSocket handshake ไม่ได้ → ส่ง token ผ่าน query: /ws?token=<jwt>
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := authH.UserIDFromRequest(r) // แกะ JWT → userID
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		hub.ServeWs(h, w, r, userID) // ส่ง id คงที่เข้าไป
	})

	// route ทดสอบ
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// เสิร์ฟ frontend จาก web/
	http.Handle("/", http.FileServer(http.Dir("web")))

	log.Printf("server listening on http://localhost%s  (ws: %s/ws)", cfg.Addr, cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

// mustDB เปิดการเชื่อมต่อ Postgres (GORM) — บังคับต้องมี DATABASE_URL
// ไม่มี/เชื่อมไม่ได้ → fatal (เลิก in-memory fallback แล้ว ข้อมูล user/task ต้องถาวร)
func mustDB(cfg config.Config) *gorm.DB {
	if cfg.DatabaseURL == "" {
		log.Fatal("ต้องตั้ง DATABASE_URL (ไม่มี in-memory fallback แล้ว)")
	}
	db, err := config.NewGormDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("เชื่อมต่อ Postgres ไม่ได้: %v", err)
	}
	log.Println("ใช้ Postgres (GORM) เก็บ user/task — ถาวร")
	return db
}
