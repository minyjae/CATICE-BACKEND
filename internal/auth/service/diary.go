package service

import (
	"strings"
	"time"

	"github/minyjae/catice/internal/auth/domain"
	"github/minyjae/catice/internal/auth/repository"
	"github/minyjae/catice/internal/id"
)

// DiaryStore = business logic ของ daily diary (upsert ต่อวัน + ดูของตัวเอง/ของลูกทีม)
type DiaryStore struct {
	repo  repository.DiaryRepository
	users repository.UsersRepository
}

func NewDiaryStore(repo repository.DiaryRepository, users repository.UsersRepository) *DiaryStore {
	return &DiaryStore{repo: repo, users: users}
}

// Upsert บันทึก diary ของวันนั้น — ยื่นซ้ำวันเดิมจะทับเนื้อหาเดิม (ไม่สร้างซ้ำ อาศัย unique index ที่ repo)
func (s *DiaryStore) Upsert(userID string, p domain.UpsertDiaryPayload) (domain.DailyDiary, error) {
	date := strings.TrimSpace(p.Date)
	content := strings.TrimSpace(p.Content)
	if date == "" {
		return domain.DailyDiary{}, domain.ErrInvalidDateRange
	}
	if content == "" {
		return domain.DailyDiary{}, domain.ErrEmptyDiaryContent
	}

	now := time.Now().Unix()
	d := domain.DailyDiary{ID: id.New(), UserID: userID, Date: date, Content: content, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.Upsert(d); err != nil {
		return domain.DailyDiary{}, err
	}
	// อ่านกลับ เพราะถ้าเป็นการอัปเดตทับ ID/CreatedAt ที่แท้จริงคือของ entry เดิม ไม่ใช่ค่าที่เพิ่งสุ่มมา
	saved, _ := s.repo.ByUserAndDate(userID, date)
	return saved, nil
}

// ListMine คืน diary ของ user คนหนึ่ง (ล่าสุดก่อน) จำกัดจำนวน
func (s *DiaryStore) ListMine(userID string, limit int) []domain.DailyDiary {
	return s.repo.ByUser(userID, limit)
}

// OfUser ให้ manager/HR เข้าดู diary ของลูกทีมวันใดวันหนึ่ง — ต้องเป็น HR หรือเป็น manager ของ targetUserID เอง
func (s *DiaryStore) OfUser(callerID string, callerRole domain.Role, targetUserID, date string) (domain.DailyDiary, bool, error) {
	if callerRole != domain.RoleHR {
		target, ok := s.users.ByID(targetUserID)
		if !ok {
			return domain.DailyDiary{}, false, domain.ErrUserNotFound
		}
		if target.ManagerID != callerID {
			return domain.DailyDiary{}, false, domain.ErrForbidden
		}
	}
	d, ok := s.repo.ByUserAndDate(targetUserID, date)
	return d, ok, nil
}
