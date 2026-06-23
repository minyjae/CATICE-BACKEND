package service

import (
	"time"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/repository"
	"github/minyjae/catice/internal/id"
)

// historyLimit = จำนวนข้อความล่าสุดต่อ scope ที่ส่งเป็นประวัติตอน join (กันส่งท่วม)
const historyLimit = 50

// ChatStore = business logic ของแชต — บันทึกข้อความ + ดึงประวัติ
type ChatStore struct {
	repo repository.MessageRepository
}

func NewChatStore(repo repository.MessageRepository) *ChatStore {
	return &ChatStore{repo: repo}
}

// Record บันทึกข้อความ (แจก id + เวลา) แล้วคืน message ที่บันทึก ให้ router เอาไป broadcast ต่อ
//   - บันทึกแบบ best-effort: เซฟ DB พลาดก็ยังคืน message (ส่ง realtime ได้อยู่)
func (s *ChatStore) Record(scope, room, fromID, fromName, to, text string) domain.Message {
	m := domain.Message{
		ID:        id.New(),
		Scope:     scope,
		Room:      room,
		FromID:    fromID,
		FromName:  fromName,
		To:        to,
		Text:      text,
		CreatedAt: time.Now().Unix(),
	}
	_ = s.repo.Create(m)
	return m
}

// History รวมประวัติที่ user คนนี้ควรเห็นตอน join: ห้องปัจจุบัน + ทั้งหมด + DM ของตัวเอง
// (แต่ละ scope เรียงเก่า→ใหม่; frontend แยกแท็บตาม scope + dedupe ด้วย message id)
func (s *ChatStore) History(room, userID string) []domain.Message {
	out := s.repo.RoomHistory(room, historyLimit)
	out = append(out, s.repo.AllHistory(historyLimit)...)
	out = append(out, s.repo.PrivateHistory(userID, historyLimit)...)
	return out
}
