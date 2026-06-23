package domain

import "errors"

type Status string

const (
	ToDoStatus  Status = "todo"
	DoingStatus Status = "doing"
	DoneStatus  Status = "done"
)

func (s Status) Valid() bool {
	switch s {
	case ToDoStatus, DoingStatus, DoneStatus:
		return true
	}

	return false
}

type Task struct {
	ID        string   `json:"id"`
	BoardID   string   `json:"board_id"` // board ที่ task นี้สังกัด
	Title     string   `json:"title"`
	Detail    string   `json:"detail"`
	TStatus   Status   `json:"status"`
	CreatedBy string   `json:"created_by"` // user id ของผู้สร้าง — server เซ็ตจาก JWT ไม่เชื่อ client
	AssignTo  []string `json:"assign_to"`
}

// errors ของ task — ให้ทุก layer อ้างถึงตัวเดียวกันผ่าน errors.Is
var (
	ErrEmptyTitle    = errors.New("ต้องกรอก title")
	ErrInvalidStatus = errors.New("status ไม่ถูกต้อง")
)
