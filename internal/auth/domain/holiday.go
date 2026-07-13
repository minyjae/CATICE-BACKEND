package domain

import "errors"

// Holiday = วันหยุดบริษัท 1 วัน (ปฏิทินรวม ไม่ผูกกับ user คนใดคนหนึ่ง)
//   - Date      : รูปแบบ YYYY-MM-DD (วันที่ตามปฏิทิน ไม่มีความหมายเรื่องเวลา)
//   - CreatedBy : user id ของ HR ที่เพิ่มวันหยุดนี้
type Holiday struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Date      string `json:"date"`
	CreatedBy string `json:"created_by"`
}

// ErrEmptyHolidayName — ชื่อวันหยุดต้องไม่ว่าง
var ErrEmptyHolidayName = errors.New("ต้องตั้งชื่อวันหยุด")
