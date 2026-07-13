// Package fakes รวม fake implementation ของ repository interface ใน internal/auth/repository
// ใช้แทน GORM จริงตอนเทส service/handler — ไม่ต้องมี Postgres รันอยู่ (ดู CLAUDE.md: DATABASE_URL บังคับสำหรับ production
// แต่ในเทสนี้เราสลับที่เก็บด้วย fake ตามที่ repository.go จงใจออกแบบไว้ให้ทำได้)
//
// ไม่มี lock เพราะเทสรันเดี่ยว (single-threaded) ไม่มี concurrency ให้ป้องกัน
package fakes

import (
	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/repository"
)

// ===================== Users =====================

type Users struct {
	byID map[string]domain.User
}

func NewUsers() *Users {
	return &Users{byID: map[string]domain.User{}}
}

// Seed ใส่ user เริ่มต้นไว้ก่อนเทส (ไม่เช็ค email ซ้ำแบบ Create — ไว้ตั้งข้อมูลตรง ๆ)
func (f *Users) Seed(u domain.User) {
	f.byID[u.ID] = u
}

func (f *Users) Create(u domain.User) error {
	for _, other := range f.byID {
		if other.Email == u.Email {
			return domain.ErrEmailTaken
		}
	}
	f.byID[u.ID] = u
	return nil
}

func (f *Users) Update(u domain.User) error {
	f.byID[u.ID] = u
	return nil
}

func (f *Users) ByEmail(email string) (domain.User, bool) {
	for _, u := range f.byID {
		if u.Email == email {
			return u, true
		}
	}
	return domain.User{}, false
}

func (f *Users) ByID(id string) (domain.User, bool) {
	u, ok := f.byID[id]
	return u, ok
}

func (f *Users) All() []domain.User {
	out := make([]domain.User, 0, len(f.byID))
	for _, u := range f.byID {
		out = append(out, u)
	}
	return out
}

func (f *Users) Delete(id string) error {
	delete(f.byID, id)
	return nil
}

var _ repository.UsersRepository = (*Users)(nil)

// ===================== Holiday =====================

type Holidays struct {
	items map[string]domain.Holiday
}

func NewHolidays() *Holidays {
	return &Holidays{items: map[string]domain.Holiday{}}
}

func (f *Holidays) Seed(h domain.Holiday) {
	f.items[h.ID] = h
}

func (f *Holidays) Create(h domain.Holiday) error {
	f.items[h.ID] = h
	return nil
}

func (f *Holidays) Delete(id string) error {
	delete(f.items, id)
	return nil
}

func (f *Holidays) All() []domain.Holiday {
	out := make([]domain.Holiday, 0, len(f.items))
	for _, h := range f.items {
		out = append(out, h)
	}
	return out
}

var _ repository.HolidayRepository = (*Holidays)(nil)

// ===================== Leave =====================

type Leaves struct {
	items map[string]domain.LeaveRequest
}

func NewLeaves() *Leaves {
	return &Leaves{items: map[string]domain.LeaveRequest{}}
}

func (f *Leaves) Seed(l domain.LeaveRequest) {
	f.items[l.ID] = l
}

func (f *Leaves) Create(l domain.LeaveRequest) error {
	f.items[l.ID] = l
	return nil
}

func (f *Leaves) Update(l domain.LeaveRequest) error {
	f.items[l.ID] = l
	return nil
}

func (f *Leaves) ByID(id string) (domain.LeaveRequest, bool) {
	l, ok := f.items[id]
	return l, ok
}

func (f *Leaves) ByUser(userID string) []domain.LeaveRequest {
	var out []domain.LeaveRequest
	for _, l := range f.items {
		if l.UserID == userID {
			out = append(out, l)
		}
	}
	return out
}

func (f *Leaves) PendingForApprover(approverID string) []domain.LeaveRequest {
	var out []domain.LeaveRequest
	for _, l := range f.items {
		if l.ApproverID == approverID && l.Status == domain.StatusPending {
			out = append(out, l)
		}
	}
	return out
}

func (f *Leaves) ApprovedByUserTypeYear(userID string, leaveType domain.LeaveType, year int) []domain.LeaveRequest {
	var out []domain.LeaveRequest
	for _, l := range f.items {
		if l.UserID == userID && l.Type == leaveType && l.Status == domain.StatusApproved {
			out = append(out, l)
		}
	}
	return out
}

var _ repository.LeaveRepository = (*Leaves)(nil)

// ===================== WFH =====================

type WFH struct {
	items map[string]domain.WFHRequest
}

func NewWFH() *WFH {
	return &WFH{items: map[string]domain.WFHRequest{}}
}

func (f *WFH) Seed(w domain.WFHRequest) {
	f.items[w.ID] = w
}

func (f *WFH) Create(w domain.WFHRequest) error {
	f.items[w.ID] = w
	return nil
}

func (f *WFH) Update(w domain.WFHRequest) error {
	f.items[w.ID] = w
	return nil
}

func (f *WFH) ByID(id string) (domain.WFHRequest, bool) {
	w, ok := f.items[id]
	return w, ok
}

func (f *WFH) ByUser(userID string) []domain.WFHRequest {
	var out []domain.WFHRequest
	for _, w := range f.items {
		if w.UserID == userID {
			out = append(out, w)
		}
	}
	return out
}

func (f *WFH) PendingForApprover(approverID string) []domain.WFHRequest {
	var out []domain.WFHRequest
	for _, w := range f.items {
		if w.ApproverID == approverID && w.Status == domain.StatusPending {
			out = append(out, w)
		}
	}
	return out
}

func (f *WFH) CountApprovedByUserInRange(userID, startDate, endDate string) int {
	count := 0
	for _, w := range f.items {
		if w.UserID == userID && w.Status == domain.StatusApproved &&
			w.Date >= startDate && w.Date <= endDate {
			count++
		}
	}
	return count
}

var _ repository.WFHRepository = (*WFH)(nil)

// ===================== Diary =====================

type Diaries struct {
	items map[string]domain.DailyDiary // key: userID + "|" + date
}

func NewDiaries() *Diaries {
	return &Diaries{items: map[string]domain.DailyDiary{}}
}

func diaryKey(userID, date string) string { return userID + "|" + date }

func (f *Diaries) Seed(d domain.DailyDiary) {
	f.items[diaryKey(d.UserID, d.Date)] = d
}

// Upsert mimic พฤติกรรม ON CONFLICT ของ gormDiaries.Upsert จริง:
// มี (user_id,date) เดิมอยู่แล้ว → ทับ content/updated_at แต่คง id/created_at เดิม ไม่สร้างแถวใหม่
func (f *Diaries) Upsert(d domain.DailyDiary) error {
	key := diaryKey(d.UserID, d.Date)
	if existing, ok := f.items[key]; ok {
		existing.Content = d.Content
		existing.UpdatedAt = d.UpdatedAt
		f.items[key] = existing
		return nil
	}
	f.items[key] = d
	return nil
}

func (f *Diaries) ByUserAndDate(userID, date string) (domain.DailyDiary, bool) {
	d, ok := f.items[diaryKey(userID, date)]
	return d, ok
}

func (f *Diaries) ByUser(userID string, limit int) []domain.DailyDiary {
	var out []domain.DailyDiary
	for _, d := range f.items {
		if d.UserID == userID {
			out = append(out, d)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

var _ repository.DiaryRepository = (*Diaries)(nil)

// ===================== Policy =====================

type Policy struct {
	current domain.LeavePolicy
	set     bool
}

func NewPolicy() *Policy {
	return &Policy{}
}

func (f *Policy) Get() domain.LeavePolicy {
	if f.set {
		return f.current
	}
	return domain.DefaultPolicy
}

func (f *Policy) Save(p domain.LeavePolicy) error {
	f.current = p
	f.set = true
	return nil
}

var _ repository.PolicyRepository = (*Policy)(nil)
