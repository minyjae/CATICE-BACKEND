package config

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewGormDB เปิดการเชื่อมต่อ Postgres ผ่าน GORM
//   - retry กันกรณี container `db` ยังขึ้นไม่เสร็จตอน backend เพิ่ง start (docker-compose race)
//   - TranslateError: true → ให้ GORM แปลง error ของ driver เป็น error กลาง (เช่น gorm.ErrDuplicatedKey)
//     ทำให้ repository เช็ค email ซ้ำได้โดยไม่ผูกกับ Postgres error code
func NewGormDB(dsn string) (*gorm.DB, error) {
	cfg := &gorm.Config{
		TranslateError: true,
		Logger:         logger.Default.LogMode(logger.Warn),
	}

	var db *gorm.DB
	var err error
	for range 10 {
		db, err = gorm.Open(postgres.Open(dsn), cfg)
		if err == nil {
			if sqlDB, e := db.DB(); e == nil && sqlDB.Ping() == nil {
				return db, nil // เชื่อมได้ + ping ผ่าน
			}
		}
		time.Sleep(time.Second) // รอ Postgres พร้อมรับ connection
	}
	return nil, err
}
