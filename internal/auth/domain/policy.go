package domain

import "errors"

// LeavePolicy = นโยบายวันลา/WFH ของบริษัท (singleton — มีชุดเดียวทั้งองค์กร)
// HR ตั้งค่าได้ผ่าน PUT /policy; ทุกคนที่ login ดูได้ผ่าน GET /policy
type LeavePolicy struct {
	VacationDaysPerYear int `json:"vacation_days_per_year"`
	SickDaysPerYear     int `json:"sick_days_per_year"`
	PersonalDaysPerYear int `json:"personal_days_per_year"`
	WFHDaysPerWeek      int `json:"wfh_days_per_week"`
	WFHDaysPerMonth     int `json:"wfh_days_per_month"`
}

// DefaultPolicy = ค่าที่ใช้เมื่อยังไม่เคยตั้ง policy ใน DB
var DefaultPolicy = LeavePolicy{
	VacationDaysPerYear: 10,
	SickDaysPerYear:     30,
	PersonalDaysPerYear: 3,
	WFHDaysPerWeek:      2,
	WFHDaysPerMonth:     8,
}

var (
	ErrLeaveQuotaExceeded = errors.New("วันลาเกินโควต้าที่กำหนด")
	ErrWFHWeeklyExceeded  = errors.New("WFH เกินโควต้ารายสัปดาห์")
	ErrWFHMonthlyExceeded = errors.New("WFH เกินโควต้ารายเดือน")
	ErrInvalidPolicy      = errors.New("ค่า policy ต้องไม่ติดลบ")
)
