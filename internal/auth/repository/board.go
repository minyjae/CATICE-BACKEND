package repository

import (
	"gorm.io/gorm"

	"github/minyjae/catice/internal/auth/domain"
)

// BoardModel = persistence model ของ board (ตาราง boards ใน Postgres)
type BoardModel struct {
	ID   string `gorm:"primaryKey"`
	Name string `gorm:"not null"`
}

func (BoardModel) TableName() string { return "boards" }

func boardToDomain(m BoardModel) domain.Board {
	return domain.Board{ID: m.ID, Name: m.Name}
}

func boardFromDomain(b domain.Board) BoardModel {
	return BoardModel{ID: b.ID, Name: b.Name}
}

// gormBoards = impl ของ BoardRepository เก็บลง Postgres ผ่าน GORM (ถาวร)
type gormBoards struct {
	db *gorm.DB
}

// NewGormBoards สร้าง repository + run AutoMigrate สร้าง/อัปเดตตาราง boards
func NewGormBoards(db *gorm.DB) (*gormBoards, error) {
	if err := db.AutoMigrate(&BoardModel{}); err != nil {
		return nil, err
	}
	return &gormBoards{db: db}, nil
}

func (g *gormBoards) Create(b domain.Board) error {
	m := boardFromDomain(b)
	return g.db.Create(&m).Error
}

func (g *gormBoards) Update(b domain.Board) error {
	m := boardFromDomain(b)
	return g.db.Save(&m).Error
}

func (g *gormBoards) Delete(id string) error {
	return g.db.Delete(&BoardModel{}, "id = ?", id).Error
}

func (g *gormBoards) ByID(id string) (domain.Board, bool) {
	var m BoardModel
	if err := g.db.First(&m, "id = ?", id).Error; err != nil {
		return domain.Board{}, false
	}
	return boardToDomain(m), true
}

// All คืน board ทั้งหมด — เรียงตาม id ให้ลำดับคงที่
func (g *gormBoards) All() []domain.Board {
	var ms []BoardModel
	if err := g.db.Order("id").Find(&ms).Error; err != nil {
		return nil
	}
	out := make([]domain.Board, 0, len(ms))
	for _, m := range ms {
		out = append(out, boardToDomain(m))
	}
	return out
}
