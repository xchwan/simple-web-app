package eventrepo

import (
	"strings"

	"gorm.io/gorm"
)

// MySQLEventRepository 以 MySQL 實作 EventRepository。
type MySQLEventRepository struct {
	db *gorm.DB
}

// NewMySQLRepository 建立一個 MySQLEventRepository。
func NewMySQLRepository(db *gorm.DB) *MySQLEventRepository {
	return &MySQLEventRepository{db: db}
}

func (r *MySQLEventRepository) Save(e *Event) error {
	return r.db.Create(e).Error
}

func (r *MySQLEventRepository) FindByID(id int) (*Event, bool) {
	var e Event
	if err := r.db.First(&e, id).Error; err != nil {
		return nil, false
	}
	return &e, true
}

func (r *MySQLEventRepository) Search(q EventQuery) []*Event {
	var events []*Event
	query := r.db
	if strings.TrimSpace(q.Keyword) != "" {
		query = query.Where("name LIKE ?", "%"+q.Keyword+"%")
	}
	if q.StartFrom != nil {
		query = query.Where("start_at >= ?", *q.StartFrom)
	}
	if q.StartTo != nil {
		query = query.Where("start_at <= ?", *q.StartTo)
	}
	query.Find(&events)
	return events
}

func (r *MySQLEventRepository) Update(e *Event) error {
	return r.db.Save(e).Error
}

func (r *MySQLEventRepository) Delete(id int) {
	r.db.Delete(&Event{}, id)
}
