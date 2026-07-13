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

// PublicUser = ข้อมูล user แบบ "เปิดเผยได้" สำหรับทำ selector มอบหมาย task / แสดง org chart
//   - Name      : ส่วนหน้า @ ของอีเมล (ไม่หลุด email เต็ม/hash)
//   - ManagerID : ใครเป็นหัวหน้า/ผู้อนุมัติของ user นี้
type PublicUser struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Role      Role   `json:"role"`
	ManagerID string `json:"manager_id,omitempty"`
}

// SetManagerPayload = ตั้ง/เคลียร์หัวหน้าของ user คนหนึ่ง (HR เท่านั้น) — ManagerID ว่างได้ (เคลียร์)
type SetManagerPayload struct {
	ManagerID string `json:"manager_id"`
}

// ----- holiday: payload ที่รับจาก client -----

type CreateHolidayPayload struct {
	Name string `json:"name"`
	Date string `json:"date"` // YYYY-MM-DD
}

// ----- leave: payload ที่รับจาก client -----

// CreateLeavePayload = ข้อมูลที่ใช้ยื่นคำขอลา — UserID ไม่อยู่ในนี้ (server เซ็ตจาก JWT)
type CreateLeavePayload struct {
	Type      LeaveType `json:"type"`
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
	Reason    string    `json:"reason"`
}

// ----- wfh: payload ที่รับจาก client -----

type CreateWFHPayload struct {
	Date   string `json:"date"`
	Reason string `json:"reason"`
}

// ----- leave/wfh: payload ที่ใช้ตัดสินใจคำขอ (approver ใช้) -----

type DecideRequestPayload struct {
	Approve bool `json:"approve"`
}

// ----- diary: payload ที่รับจาก client -----

type UpsertDiaryPayload struct {
	Date    string `json:"date"`
	Content string `json:"content"`
}

// ----- hr user management -----

// UpdateProfilePayload = ข้อมูลที่ HR ส่งมาแก้ไขโปรไฟล์พนักงาน
// ทุก field เป็น pointer — ไม่ส่งมา (nil) = ไม่แก้ field นั้น (partial update)
type UpdateProfilePayload struct {
	FirstName *string  `json:"first_name"`
	LastName  *string  `json:"last_name"`
	Phone     *string  `json:"phone"`
	BirthDate *string  `json:"birth_date"`
	Address   *string  `json:"address"`
	Salary    *float64 `json:"salary"`
	StartDate *string  `json:"start_date"`
}

// ChangeRolePayload = เปลี่ยนตำแหน่งพนักงาน (HR ใช้ตอนเลื่อนขั้น)
type ChangeRolePayload struct {
	Role Role `json:"role"`
}

// HRUserView = ข้อมูล user เต็ม ๆ รวม sensitive fields (HR เท่านั้น)
// แยกจาก PublicUser เพื่อไม่ให้ salary หลุดออก endpoint สาธารณะ
type HRUserView struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"` // email prefix — compat กับ PublicUser
	Role      Role     `json:"role"`
	ManagerID string   `json:"manager_id,omitempty"`
	FirstName string   `json:"first_name,omitempty"`
	LastName  string   `json:"last_name,omitempty"`
	Phone     string   `json:"phone,omitempty"`
	BirthDate string   `json:"birth_date,omitempty"`
	Address   string   `json:"address,omitempty"`
	Salary    *float64 `json:"salary,omitempty"`
	StartDate string   `json:"start_date,omitempty"`
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
