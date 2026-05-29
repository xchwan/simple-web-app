package event

import (
	"time"

	eventdb "github.com/xchwan/simple-web-app/internal/event/db"
)

// EventService 負責活動相關的業務邏輯。
type EventService struct {
	repo eventdb.EventRepository
}

// NewEventService 建立一個 EventService。
func NewEventService(repo eventdb.EventRepository) *EventService {
	return &EventService{repo: repo}
}

// Create 建立一個新活動。
func (s *EventService) Create(organizerID int, name, description string, startAt time.Time) (*eventdb.Event, error) {
	if !validateLength(name, 1, 255) {
		return nil, ErrNameFormatInvalid
	}
	if startAt.IsZero() {
		return nil, ErrStartAtInvalid
	}
	e := &eventdb.Event{
		OrganizerID: organizerID,
		Name:        name,
		Description: description,
		StartAt:     startAt,
	}
	if err := s.repo.Save(e); err != nil {
		return nil, err
	}
	return e, nil
}

// GetByID 依編號取得活動。
func (s *EventService) GetByID(id int) (*eventdb.Event, error) {
	e, exists := s.repo.FindByID(id)
	if !exists {
		return nil, ErrNotFound
	}
	return e, nil
}

// Search 依條件查詢活動列表。
func (s *EventService) Search(q eventdb.EventQuery) []*eventdb.Event {
	return s.repo.Search(q)
}

// Update 更新活動，僅主辦人可操作；只更新非零值欄位。
func (s *EventService) Update(callerID, eventID int, name, description string, startAt time.Time) (*eventdb.Event, error) {
	e, exists := s.repo.FindByID(eventID)
	if !exists {
		return nil, ErrNotFound
	}
	if e.OrganizerID != callerID {
		return nil, ErrForbidden
	}
	if name != "" {
		if !validateLength(name, 1, 255) {
			return nil, ErrNameFormatInvalid
		}
		e.Name = name
	}
	if description != "" {
		e.Description = description
	}
	if !startAt.IsZero() {
		e.StartAt = startAt
	}
	if err := s.repo.Update(e); err != nil {
		return nil, err
	}
	return e, nil
}

// Delete 刪除活動，僅主辦人可操作。
func (s *EventService) Delete(callerID, eventID int) error {
	e, exists := s.repo.FindByID(eventID)
	if !exists {
		return ErrNotFound
	}
	if e.OrganizerID != callerID {
		return ErrForbidden
	}
	s.repo.Delete(eventID)
	return nil
}

func validateLength(s string, min, max int) bool {
	n := len([]rune(s))
	return n >= min && n <= max
}
