package event

import "time"

// EventService 負責活動相關的業務邏輯。
type EventService struct {
	db *EventDB
}

// NewEventService 建立一個 EventService。
func NewEventService(db *EventDB) *EventService {
	return &EventService{db: db}
}

// Create 建立一個新活動。
func (s *EventService) Create(organizerID int, name, description string, startAt time.Time) (*Event, error) {
	if !validateLength(name, 1, 255) {
		return nil, ErrNameFormatInvalid
	}
	if startAt.IsZero() {
		return nil, ErrStartAtInvalid
	}
	e := &Event{
		OrganizerID: organizerID,
		Name:        name,
		Description: description,
		StartAt:     startAt,
	}
	if err := s.db.Save(e); err != nil {
		return nil, err
	}
	return e, nil
}

// GetByID 依編號取得活動。
func (s *EventService) GetByID(id int) (*Event, error) {
	e, exists := s.db.FindByID(id)
	if !exists {
		return nil, ErrNotFound
	}
	return e, nil
}

// Search 依關鍵字查詢活動列表。
func (s *EventService) Search(keyword string) []*Event {
	return s.db.Search(keyword)
}

// Update 更新活動，僅主辦人可操作；只更新非零值欄位。
func (s *EventService) Update(callerID, eventID int, name, description string, startAt time.Time) (*Event, error) {
	e, exists := s.db.FindByID(eventID)
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
	if err := s.db.Update(e); err != nil {
		return nil, err
	}
	return e, nil
}

// Delete 刪除活動，僅主辦人可操作。
func (s *EventService) Delete(callerID, eventID int) error {
	e, exists := s.db.FindByID(eventID)
	if !exists {
		return ErrNotFound
	}
	if e.OrganizerID != callerID {
		return ErrForbidden
	}
	s.db.Delete(eventID)
	return nil
}

func validateLength(s string, min, max int) bool {
	n := len([]rune(s))
	return n >= min && n <= max
}
