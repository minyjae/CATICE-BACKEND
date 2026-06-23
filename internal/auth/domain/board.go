package domain

import "errors"

// Board = บอร์ด kanban 1 ใบ (มีหลายใบได้) — task ผูกกับ board ผ่าน BoardID
type Board struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ErrEmptyBoardName — ชื่อบอร์ดต้องไม่ว่าง
var ErrEmptyBoardName = errors.New("ต้องตั้งชื่อ board")
