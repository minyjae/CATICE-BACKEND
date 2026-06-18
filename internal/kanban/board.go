// Package kanban เก็บ "สถานะบอร์ด" (task ต่าง ๆ) — domain ล้วน ๆ ไม่พึ่ง network
// แตะโดย goroutine เดียว (router) → ไม่ต้องใช้ lock (หลักการเดียวกับ room.Manager)
package kanban

import "github/minyjae/catice/internal/id"

// Task = การ์ด 1 ใบบนบอร์ด
type Task struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Detail    string   `json:"detail"`
	Status    string   `json:"status"`     // "todo" / "doing" / "done"
	CreatedBy string   `json:"created_by"` // user id (server เซ็ต ไม่เชื่อ client)
	AssignTo  []string `json:"assign_to"`
}

// Board เก็บ task ทั้งหมด — map keyed by id → ย้าย/แก้/ลบด้วย id ได้ทันที (O(1))
type Board struct {
	tasks map[string]Task
}

func NewBoard() *Board {
	return &Board{tasks: make(map[string]Task)}
}

// CreateTask สร้าง task ใหม่: server แจก id + ตั้ง status เริ่มต้น "todo" → คืน task ที่สร้าง
func (b *Board) CreateTask(title, detail, createdBy string, assignTo []string) Task {
	t := Task{
		ID:        id.New(),
		Title:     title,
		Detail:    detail,
		Status:    "todo",
		CreatedBy: createdBy,
		AssignTo:  assignTo,
	}
	b.tasks[t.ID] = t
	return t
}

// MoveTask เปลี่ยน column/status ของ task (ok=false ถ้าไม่มี task นั้น)
func (b *Board) MoveTask(taskID, status string) (Task, bool) {
	t, ok := b.tasks[taskID]
	if !ok {
		return Task{}, false
	}
	t.Status = status
	b.tasks[taskID] = t
	return t, true
}

// UpdateTask แก้รายละเอียด (ok=false ถ้าไม่มี)
func (b *Board) UpdateTask(taskID, title, detail string, assignTo []string) (Task, bool) {
	t, ok := b.tasks[taskID]
	if !ok {
		return Task{}, false
	}
	t.Title, t.Detail, t.AssignTo = title, detail, assignTo
	b.tasks[taskID] = t
	return t, true
}

// DeleteTask ลบ task
func (b *Board) DeleteTask(taskID string) {
	delete(b.tasks, taskID)
}

// Tasks คืน task ทั้งหมด — ใช้ทำ snapshot ให้คนที่เข้ามาทีหลัง
func (b *Board) Tasks() []Task {
	out := make([]Task, 0, len(b.tasks))
	for _, t := range b.tasks {
		out = append(out, t)
	}
	return out
}
