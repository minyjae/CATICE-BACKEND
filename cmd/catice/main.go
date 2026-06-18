package main

import (
	"log"
	"net/http"

	"github/minyjae/catice/internal/auth/controller"
	"github/minyjae/catice/internal/auth/repository"
	"github/minyjae/catice/internal/auth/service"
	"github/minyjae/catice/internal/config"
	"github/minyjae/catice/internal/hub"
	"github/minyjae/catice/internal/kanban"
	"github/minyjae/catice/internal/room"
	"github/minyjae/catice/internal/router"
)

func main() {
	cfg := config.Load() // อ่านค่าตั้งจาก env (DATABASE_URL, ADDR)

	// ประกอบชั้นต่าง ๆ เข้าด้วยกัน (dependency ไหลทางเดียว: router → hub/room/...)
	h := hub.New()                 // ชั้น transport (การเชื่อมต่อ)
	go h.Run()                     //
	rm := room.NewManager()        // ชั้น state (ตำแหน่งผู้เล่น)
	board := kanban.NewBoard()     // ชั้น state (kanban)
	rt := router.New(h, rm, board) // ตัวสั่งการ: ดูด hub แล้ว dispatch
	go rt.Run()                    //

	// auth: repository → store(service) → handler(controller)
	// เลือก repository ตาม config: มี DATABASE_URL → Postgres(GORM) ถาวร, ไม่มี → in-memory
	authH := controller.NewHandler(service.NewStore(usersRepo(cfg)), service.NewSessions())
	http.HandleFunc("/register", authH.Register)
	http.HandleFunc("/login", authH.Login)
	http.HandleFunc("/logout", authH.Logout)
	http.HandleFunc("/me", authH.RequireAuth(authH.Me))       // ต้อง login ก่อน (middleware เช็ค cookie)
	http.HandleFunc("/users", authH.RequireAuth(authH.Users)) // รายชื่อ user ทั้งหมด → selector มอบหมาย task

	// route /ws → เช็ค cookie (ต้อง login ก่อน) → upgrade → register เข้า hub
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := authH.UserIDFromRequest(r) // แกะ cookie → userID
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

// usersRepo เลือกที่เก็บ user ตาม config:
//   - มี DATABASE_URL → Postgres ผ่าน GORM (ถาวร, รอด restart ผ่าน volume)
//   - ไม่มี           → in-memory (สะดวกตอน dev/test — restart แล้วหาย)
func usersRepo(cfg config.Config) repository.UsersRepository {
	if cfg.DatabaseURL == "" {
		log.Println("DATABASE_URL ว่าง → ใช้ user store แบบ in-memory (restart แล้วข้อมูลหาย)")
		return repository.NewMemUsers()
	}
	db, err := config.NewGormDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("เชื่อมต่อ Postgres ไม่ได้: %v", err)
	}
	repo, err := repository.NewGormUsers(db)
	if err != nil {
		log.Fatalf("migrate ตาราง users ไม่ได้: %v", err)
	}
	log.Println("ใช้ Postgres (GORM) เก็บ user — ถาวร")
	return repo
}
