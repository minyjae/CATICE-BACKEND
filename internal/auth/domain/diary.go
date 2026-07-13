package domain

import "errors"

// DailyDiary = บันทึกงานประจำวันของ user คนหนึ่ง — 1 user เขียนได้วันละ 1 entry (unique ที่ persistence layer)
type DailyDiary struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Date      string `json:"date"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// ErrEmptyDiaryContent — เนื้อหา diary ต้องไม่ว่าง
var ErrEmptyDiaryContent = errors.New("ต้องกรอกเนื้อหา diary")
