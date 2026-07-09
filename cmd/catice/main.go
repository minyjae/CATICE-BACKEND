package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github/minyjae/catice/internal/auth/handler"
	"github/minyjae/catice/internal/auth/repository"
	"github/minyjae/catice/internal/auth/service"
	"github/minyjae/catice/internal/config"
	"github/minyjae/catice/internal/hub"
	"github/minyjae/catice/internal/presence"
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
	boardsRepo, err := repository.NewGormBoards(db)
	if err != nil {
		log.Fatalf("migrate ตาราง boards ไม่ได้: %v", err)
	}
	messagesRepo, err := repository.NewGormMessages(db)
	if err != nil {
		log.Fatalf("migrate ตาราง messages ไม่ได้: %v", err)
	}
	holidaysRepo, err := repository.NewGormHolidays(db)
	if err != nil {
		log.Fatalf("migrate ตาราง holidays ไม่ได้: %v", err)
	}
	leaveRepo, err := repository.NewGormLeaves(db)
	if err != nil {
		log.Fatalf("migrate ตาราง leave_requests ไม่ได้: %v", err)
	}
	wfhRepo, err := repository.NewGormWFH(db)
	if err != nil {
		log.Fatalf("migrate ตาราง wfh_requests ไม่ได้: %v", err)
	}
	diaryRepo, err := repository.NewGormDiaries(db)
	if err != nil {
		log.Fatalf("migrate ตาราง daily_diaries ไม่ได้: %v", err)
	}
	policyRepo, err := repository.NewGormPolicy(db)
	if err != nil {
		log.Fatalf("migrate ตาราง leave_policy ไม่ได้: %v", err)
	}
	taskStore := service.NewTaskStore(tasksRepo)    // task ทั้งหมดไปทาง WS (router) — ดู create/move/update/delete ใน message_router
	boardStore := service.NewBoardStore(boardsRepo) // board (kanban หลายใบ) ผ่าน WS เช่นกัน
	chatStore := service.NewChatStore(messagesRepo) // แชต room/all/private เก็บลง DB + ส่งประวัติตอน join
	positions := presenceStore(cfg)                 // ตำแหน่ง client ถาวร (Redis) → reconnect/refresh/logout แล้วยืนที่เดิม

	// ---- HR module (REST ล้วน ไม่ผ่าน WS — ดู CLAUDE.md เหตุผลที่ไม่ใช้ realtime layer) ----
	holidayStore := service.NewHolidayStore(holidaysRepo)
	policyStore := service.NewPolicyStore(policyRepo)
	leaveStore := service.NewLeaveStore(leaveRepo, usersRepo, policyStore)
	wfhStore := service.NewWFHStore(wfhRepo, usersRepo, policyStore)
	diaryStore := service.NewDiaryStore(diaryRepo, usersRepo)

	// ---- ชั้น transport/state (dependency ไหลทางเดียว: router → hub/room/...) ----
	h := hub.New()                                                       // ชั้น transport (การเชื่อมต่อ)
	go h.Run()                                                           //
	rm := room.NewManager()                                              // ชั้น state (ตำแหน่งผู้เล่น in-memory)
	rt := router.New(h, rm, taskStore, boardStore, chatStore, positions) // ตัวสั่งการ: ดูด hub แล้ว dispatch
	go rt.Run()                                                          //

	// ---- auth (handler) ----
	authH := handler.NewAuthHandler(service.NewStore(usersRepo), service.NewTokens(cfg.JWTSecret))
	http.HandleFunc("/register", authH.Register)
	http.HandleFunc("/login", authH.Login)
	http.HandleFunc("/logout", authH.Logout)
	http.HandleFunc("/me", authH.RequireAuth(authH.Me))                               // ต้อง login ก่อน (middleware เช็ค JWT)
	http.HandleFunc("/users", authH.RequireAuth(authH.Users))                         // รายชื่อ user ทั้งหมด → selector มอบหมาย task
	http.HandleFunc("PATCH /users/{id}/manager", authH.RequireAuth(authH.SetManager)) // ตั้ง/เคลียร์หัวหน้า (เฉพาะ HR)
	// task: สร้าง/ย้าย/แก้/ลบ ทั้งหมดไปทาง WS (/ws) → realtime + บันทึกลง DB (ดู internal/router/message_router.go)

	// ---- HR module (REST) ----
	holidayH := handler.NewHolidayHandler(holidayStore)
	http.HandleFunc("GET /holidays", authH.RequireAuth(holidayH.List))
	http.HandleFunc("POST /holidays", authH.RequireAuth(holidayH.Create))        // เฉพาะ HR
	http.HandleFunc("DELETE /holidays/{id}", authH.RequireAuth(holidayH.Delete)) // เฉพาะ HR

	leaveH := handler.NewLeaveHandler(leaveStore)
	http.HandleFunc("POST /leaves", authH.RequireAuth(leaveH.Create))
	http.HandleFunc("GET /leaves/mine", authH.RequireAuth(leaveH.Mine))
	http.HandleFunc("GET /leaves/pending", authH.RequireAuth(leaveH.Pending))
	http.HandleFunc("POST /leaves/{id}/approve", authH.RequireAuth(leaveH.Approve))
	http.HandleFunc("POST /leaves/{id}/reject", authH.RequireAuth(leaveH.Reject))

	wfhH := handler.NewWFHHandler(wfhStore)
	http.HandleFunc("POST /wfh", authH.RequireAuth(wfhH.Create))
	http.HandleFunc("GET /wfh/mine", authH.RequireAuth(wfhH.Mine))
	http.HandleFunc("GET /wfh/pending", authH.RequireAuth(wfhH.Pending))
	http.HandleFunc("POST /wfh/{id}/approve", authH.RequireAuth(wfhH.Approve))
	http.HandleFunc("POST /wfh/{id}/reject", authH.RequireAuth(wfhH.Reject))

	policyH := handler.NewPolicyHandler(policyStore)
	http.HandleFunc("GET /policy", authH.RequireAuth(policyH.Get))
	http.HandleFunc("PUT /policy", authH.RequireAuth(policyH.Update)) // เฉพาะ HR

	diaryH := handler.NewDiaryHandler(diaryStore)
	http.HandleFunc("POST /diary", authH.RequireAuth(diaryH.Upsert))
	http.HandleFunc("GET /diary/mine", authH.RequireAuth(diaryH.Mine))
	http.HandleFunc("GET /diary", authH.RequireAuth(diaryH.OfUser)) // ?user_id=&date= — HR/manager ดูของลูกทีม

	// route /ws → เช็ค JWT (ต้อง login ก่อน) → upgrade → register เข้า hub
	// browser ตั้ง header บน WebSocket handshake ไม่ได้ → ส่ง token ผ่าน query: /ws?token=<jwt>
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// ตรวจ JWT + ยืนยันว่า user ยังมีอยู่ใน DB (ไม่ใช่แค่ลายเซ็น)
		// กัน ghost: token ของ user ที่ถูกลบไปแล้ว (เช่นหลัง down -v) ต่อ /ws ไม่ได้
		user, ok := authH.UserFromRequest(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		hub.ServeWs(h, w, r, user.ID) // ส่ง id คงที่เข้าไป
	})

	// route ทดสอบ
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// เสิร์ฟ frontend จาก web/
	http.Handle("/", http.FileServer(http.Dir("web")))

	log.Printf("server listening on http://localhost%s  (ws: %s/ws)", cfg.Addr, cfg.Addr)
	// หุ้มทุก route ด้วย CORS → frontend คนละโดเมน (เช่น Railway) เรียกข้ามโดเมนได้
	if err := http.ListenAndServe(cfg.Addr, withCORS(http.DefaultServeMux)); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

// withCORS หุ้ม handler ให้ตอบ CORS — รองรับ frontend ที่ host คนละโดเมนกับ backend
// auth ใช้ Bearer token ใน header (ไม่ใช่ cookie) → Allow-Origin "*" ใช้ได้ปลอดภัย
// (WS ไม่ผ่าน preflight; upgrader.CheckOrigin คุม origin ของ WS เอง)
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions { // ตอบ preflight ทันที ไม่ต้องส่งต่อ
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
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

// presenceStore เลือกที่เก็บตำแหน่ง client ตาม config:
//   - มี REDIS_URL → Redis (ถาวร: reconnect/refresh/logout แล้วยืนที่เดิม, รอด restart)
//   - ไม่มี        → Noop (ไม่เก็บ → spawn ใหม่ทุกครั้ง, สะดวกตอน dev)
func presenceStore(cfg config.Config) presence.Store {
	if cfg.RedisAddr == "" {
		log.Println("REDIS_ADDR ว่าง → ไม่เก็บตำแหน่ง client (reconnect แล้ว spawn ใหม่)")
		return presence.Noop{}
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("เชื่อมต่อ Redis ไม่ได้: %v", err)
	}
	log.Println("ใช้ Redis เก็บตำแหน่ง client — reconnect/refresh/logout แล้วยืนที่เดิม")
	return presence.NewRedis(rdb)
}
