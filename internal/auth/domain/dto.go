package domain

// ----- payload ที่รับจาก REST (json body) -----

type LoginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterPayload struct {
	Email    string `json:"email"`
	Role     Role   `json:"role"`
	Password string `json:"password"`
}

// ----- response ที่ตอบกลับ -----

type LoginResponse struct {
	Message string `json:"message"`
	Role    Role   `json:"role,omitempty"`
}

type RegisterResponse struct {
	Message string `json:"message"`
}

// PublicUser = ข้อมูล user แบบ "เปิดเผยได้" สำหรับทำ selector มอบหมาย task
//   - Name : ส่วนหน้า @ ของอีเมล (ไม่หลุด email เต็ม/hash)
type PublicUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role Role   `json:"role"`
}
