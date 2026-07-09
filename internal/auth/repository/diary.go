package repository

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github/minyjae/catice/internal/auth/domain"
)

// DailyDiaryModel = persistence model ของ daily diary (รูปร่างตาราง daily_diaries ใน Postgres)
//   - uniqueIndex บน (UserID, Date) ร่วมกัน → 1 user เขียนได้วันละ 1 entry (DB การันตี)
type DailyDiaryModel struct {
	ID        string `gorm:"primaryKey"`
	UserID    string `gorm:"uniqueIndex:idx_user_date;not null"`
	Date      string `gorm:"uniqueIndex:idx_user_date;not null"`
	Content   string `gorm:"type:text"`
	CreatedAt int64
	UpdatedAt int64
}

func (DailyDiaryModel) TableName() string { return "daily_diaries" }

func diaryToDomain(m DailyDiaryModel) domain.DailyDiary {
	return domain.DailyDiary{
		ID:        m.ID,
		UserID:    m.UserID,
		Date:      m.Date,
		Content:   m.Content,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func diaryFromDomain(d domain.DailyDiary) DailyDiaryModel {
	return DailyDiaryModel{
		ID:        d.ID,
		UserID:    d.UserID,
		Date:      d.Date,
		Content:   d.Content,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

// gormDiaries = impl ของ DiaryRepository
type gormDiaries struct {
	db *gorm.DB
}

// NewGormDiaries สร้าง repository + run AutoMigrate สร้าง/อัปเดตตาราง daily_diaries
func NewGormDiaries(db *gorm.DB) (*gormDiaries, error) {
	if err := db.AutoMigrate(&DailyDiaryModel{}); err != nil {
		return nil, err
	}
	return &gormDiaries{db: db}, nil
}

// Upsert สร้าง entry ใหม่ถ้ายังไม่มี (user_id,date) นี้ ถ้ามีแล้วอัปเดต content/updated_at ทับ
// อาศัย unique index (user_id,date) + ON CONFLICT ของ Postgres — อะตอมมิก ไม่ต้องอ่านก่อนเขียน
func (g *gormDiaries) Upsert(d domain.DailyDiary) error {
	m := diaryFromDomain(d)
	return g.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"content", "updated_at"}),
	}).Create(&m).Error
}

func (g *gormDiaries) ByUserAndDate(userID, date string) (domain.DailyDiary, bool) {
	var m DailyDiaryModel
	if err := g.db.Where("user_id = ? AND date = ?", userID, date).First(&m).Error; err != nil {
		return domain.DailyDiary{}, false
	}
	return diaryToDomain(m), true
}

// ByUser คืน diary ของ user คนหนึ่ง เรียงตามวันที่ล่าสุดก่อน จำกัดจำนวน
func (g *gormDiaries) ByUser(userID string, limit int) []domain.DailyDiary {
	var ms []DailyDiaryModel
	if err := g.db.Where("user_id = ?", userID).Order("date desc").Limit(limit).Find(&ms).Error; err != nil {
		return nil
	}
	out := make([]domain.DailyDiary, 0, len(ms))
	for _, m := range ms {
		out = append(out, diaryToDomain(m))
	}
	return out
}
