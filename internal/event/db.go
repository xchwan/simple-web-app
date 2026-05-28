package event

import (
	"strings"

	"gorm.io/gorm"
)

// EventDB 負責活動資料的 MySQL 存取。
type EventDB struct {
	db *gorm.DB
}

// NewEventDB 建立一個 EventDB。
func NewEventDB(db *gorm.DB) *EventDB {
	return &EventDB{db: db}
}

// Save 新增一個活動。
func (r *EventDB) Save(e *Event) error {
	return r.db.Create(e).Error
}

// FindByID 依活動編號查詢。
func (r *EventDB) FindByID(id int) (*Event, bool) {
	var e Event
	if err := r.db.First(&e, id).Error; err != nil {
		return nil, false
	}
	return &e, true
}

// Search 依關鍵字過濾活動名稱，空字串回傳所有活動。
func (r *EventDB) Search(keyword string) []*Event {
	var events []*Event
	query := r.db
	if strings.TrimSpace(keyword) != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	query.Find(&events)
	return events
}

// Update 更新活動資料。
func (r *EventDB) Update(e *Event) error {
	return r.db.Save(e).Error
}

// Delete 刪除指定活動。
func (r *EventDB) Delete(id int) {
	r.db.Delete(&Event{}, id)
}
