// Package config รวม "การตั้งค่าจาก environment" ไว้ที่เดียว (12-factor)
// แยกออกจาก business logic → เปลี่ยน DSN/พอร์ตได้โดยไม่แตะโค้ดส่วนอื่น
package config

import "os"

// Config = ค่าตั้งทั้งหมดที่อ่านจาก env ตอน boot
type Config struct {
	// DatabaseURL : DSN ของ Postgres (เช่น postgres://user:pass@host:5432/db?sslmode=disable)
	// ว่าง = ไม่ใช้ DB → fallback ไป user store แบบ in-memory (สะดวกตอน dev/test)
	DatabaseURL string
	// Addr : ที่อยู่ที่ server ฟัง (ดีฟอลต์ :8080)
	Addr string
	// JWTSecret : กุญแจลับสำหรับเซ็น/ตรวจ JWT (HS256)
	// ⚠️ production ต้องตั้งผ่าน env ให้เป็นค่าสุ่มยาว ๆ และเก็บเป็นความลับ
	JWTSecret string
	// RedisAddr : host:port ของ Redis สำหรับเก็บตำแหน่ง client (เช่น localhost:6379)
	// ว่าง = ไม่เก็บตำแหน่งถาวร → reconnect/refresh แล้ว spawn ใหม่ (พฤติกรรมเดิม)
	RedisAddr string
	// RedisPassword : รหัสผ่าน Redis (ว่างได้ถ้า Redis ไม่ตั้งรหัส)
	RedisPassword string
}

// Load อ่าน env → Config (มี default ให้ค่าที่ไม่ตั้ง)
func Load() Config {
	return Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		Addr:          getenv("ADDR", ":8080"),
		JWTSecret:     getenv("JWT_SECRET", "dev-secret-change-me"),
		RedisAddr:     os.Getenv("REDIS_ADDR"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
