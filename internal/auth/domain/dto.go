package domain

// DTO = รูปร่างข้อมูลที่รับ/ส่งผ่าน REST (json) — แยกจาก domain model (User/Task)
// โดยตั้งใจ เพื่อให้รูปแบบ API เปลี่ยนได้โดยไม่กระทบแก่นของ domain

// ----- auth: payload ที่รับจาก client -----

type LoginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterPayload struct {
	Email    string `json:"email"`
	Role     Role   `json:"role"`
	Password string `json:"password"`
}

// ----- auth: response ที่ตอบกลับ -----

type LoginResponse struct {
	Message string `json:"message"`
	Role    Role   `json:"role,omitempty"`
	Token   string `json:"token,omitempty"` // JWT — client เก็บไว้แล้วแนบ "Authorization: Bearer <token>" ทุก request
}

type RegisterResponse struct {
	Message string `json:"message"`
	Token   string `json:"token,omitempty"` // JWT — สมัครเสร็จ login ให้เลย
}

// PublicUser = ข้อมูล user แบบ "เปิดเผยได้" สำหรับทำ selector มอบหมาย task
//   - Name : ส่วนหน้า @ ของอีเมล (ไม่หลุด email เต็ม/hash)
type PublicUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role Role   `json:"role"`
}

// ----- task: payload ที่รับจาก client -----

// CreateTaskPayload = ข้อมูลที่ใช้สร้าง task (router แปลงจาก WS task_create payload มาให้ service)
//   - createdBy ไม่อยู่ในนี้ → server เซ็ตจาก JWT เอง (ไม่เชื่อ client)
//   - Status ว่างได้ → service จะตั้งดีฟอลต์ "todo" ให้
type CreateTaskPayload struct {
	BoardID  string   `json:"board_id"` // task สร้างใต้บอร์ดไหน
	Title    string   `json:"title"`
	Detail   string   `json:"detail"`
	Status   Status   `json:"status"`
	AssignTo []string `json:"assign_to"`
}
