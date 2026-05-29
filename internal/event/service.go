package event

import (
	"log"
	"time"

	eventrepo "github.com/xchwan/simple-web-app/internal/event/repo"
)

// EventService 負責活動相關的業務邏輯。
type EventService struct {
	db *eventrepo.MySQLEventRepository
	es *eventrepo.ElasticEventSearchRepository
}

// NewEventService 建立一個 EventService。
func NewEventService(db *eventrepo.MySQLEventRepository, es *eventrepo.ElasticEventSearchRepository) *EventService {
	return &EventService{db: db, es: es}
}

// Create 建立一個新活動，成功後同步寫入搜尋索引。
func (s *EventService) Create(organizerID int, name, description string, capacity int, startAt time.Time) (*eventrepo.Event, error) {
	if !validateLength(name, 1, 255) {
		return nil, ErrNameFormatInvalid
	}
	if capacity <= 0 {
		return nil, ErrCapacityInvalid
	}
	if startAt.IsZero() {
		return nil, ErrStartAtInvalid
	}
	e := &eventrepo.Event{
		OrganizerID: organizerID,
		Name:        name,
		Description: description,
		Capacity:    capacity,
		StartAt:     startAt,
	}
	if err := s.db.Save(e); err != nil {
		return nil, err
	}
	s.index(e)
	return e, nil
}

// GetByID 依編號取得活動（從 MySQL 查詢）。
func (s *EventService) GetByID(id int) (*eventrepo.Event, error) {
	e, exists := s.db.FindByID(id)
	if !exists {
		return nil, ErrNotFound
	}
	return e, nil
}

// Search 查詢活動列表。
// 若有 ES，走全文索引；若無或 ES 失敗，fallback 到 MySQL LIKE。
func (s *EventService) Search(q eventrepo.EventQuery) []*eventrepo.Event {
	if s.es != nil {
		results, err := s.es.Search(q)
		if err != nil {
			log.Printf("[ES] search 失敗，fallback MySQL: %v", err)
			return s.db.Search(q)
		}
		return results
	}
	return s.db.Search(q)
}

// Update 更新活動，僅主辦人可操作；成功後同步更新搜尋索引。
func (s *EventService) Update(callerID, eventID int, name, description string, capacity int, startAt time.Time) (*eventrepo.Event, error) {
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
	if capacity != 0 {
		if capacity < 0 {
			return nil, ErrCapacityInvalid
		}
		e.Capacity = capacity
	}
	if !startAt.IsZero() {
		e.StartAt = startAt
	}
	if err := s.db.Update(e); err != nil {
		return nil, err
	}
	s.index(e)
	return e, nil
}

// Delete 刪除活動，僅主辦人可操作；成功後從搜尋索引移除。
func (s *EventService) Delete(callerID, eventID int) error {
	e, exists := s.db.FindByID(eventID)
	if !exists {
		return ErrNotFound
	}
	if e.OrganizerID != callerID {
		return ErrForbidden
	}
	s.db.Delete(eventID)
	if s.es != nil {
		s.es.Remove(eventID)
	}
	return nil
}

// index 將 event 同步到搜尋索引；失敗只記 log，不影響主流程。
func (s *EventService) index(e *eventrepo.Event) {
	if s.es == nil {
		return
	}
	if err := s.es.Index(e); err != nil {
		log.Printf("[ES] index 失敗 (id=%d): %v", e.ID, err)
	}
}

func validateLength(s string, min, max int) bool {
	n := len([]rune(s))
	return n >= min && n <= max
}
