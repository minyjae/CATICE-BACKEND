package repository

import (
	"gorm.io/gorm"

	"github/minyjae/catice/internal/auth/domain"
)

// MessageModel = persistence model ของข้อความแชต (ตาราง messages)
//   - ToID ใช้ชื่อคอลัมน์ to_id (เลี่ยงคำสงวน "to" ของ SQL)
type MessageModel struct {
	ID        string `gorm:"primaryKey"`
	Scope     string `gorm:"index"` // room|all|private
	Room      string `gorm:"index"`
	FromID    string `gorm:"index"`
	FromName  string
	ToID      string `gorm:"index"`
	Text      string
	CreatedAt int64 `gorm:"index"` // unix seconds — เรียงลำดับ
}

func (MessageModel) TableName() string { return "messages" }

func msgToDomain(m MessageModel) domain.Message {
	return domain.Message{
		ID: m.ID, Scope: m.Scope, Room: m.Room,
		FromID: m.FromID, FromName: m.FromName, To: m.ToID,
		Text: m.Text, CreatedAt: m.CreatedAt,
	}
}

func msgFromDomain(d domain.Message) MessageModel {
	return MessageModel{
		ID: d.ID, Scope: d.Scope, Room: d.Room,
		FromID: d.FromID, FromName: d.FromName, ToID: d.To,
		Text: d.Text, CreatedAt: d.CreatedAt,
	}
}

type gormMessages struct {
	db *gorm.DB
}

func NewGormMessages(db *gorm.DB) (*gormMessages, error) {
	if err := db.AutoMigrate(&MessageModel{}); err != nil {
		return nil, err
	}
	return &gormMessages{db: db}, nil
}

func (g *gormMessages) Create(d domain.Message) error {
	m := msgFromDomain(d)
	return g.db.Create(&m).Error
}

func (g *gormMessages) RoomHistory(room string, limit int) []domain.Message {
	return g.query(limit, "scope = ? AND room = ?", "room", room)
}

func (g *gormMessages) AllHistory(limit int) []domain.Message {
	return g.query(limit, "scope = ?", "all")
}

func (g *gormMessages) PrivateHistory(userID string, limit int) []domain.Message {
	return g.query(limit, "scope = ? AND (from_id = ? OR to_id = ?)", "private", userID, userID)
}

// query ดึง N ข้อความล่าสุด (created_at desc + limit) แล้ว reverse → เรียงเก่า→ใหม่ (asc)
func (g *gormMessages) query(limit int, where string, args ...any) []domain.Message {
	var ms []MessageModel
	if err := g.db.Where(where, args...).Order("created_at desc").Limit(limit).Find(&ms).Error; err != nil {
		return nil
	}
	out := make([]domain.Message, len(ms))
	for i, m := range ms {
		out[len(ms)-1-i] = msgToDomain(m) // reverse → เก่าไปใหม่
	}
	return out
}
