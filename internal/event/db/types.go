package eventdb

import "time"

// Event 代表一個活動。
type Event struct {
	ID          int       `gorm:"primarykey;autoIncrement"`
	OrganizerID int       `gorm:"not null;index"`
	Name        string    `gorm:"not null;size:255"`
	Description string    `gorm:"type:text"`
	StartAt     time.Time `gorm:"not null"`
}

// EventQuery 代表搜尋活動的條件。
type EventQuery struct {
	Keyword   string
	StartFrom *time.Time
	StartTo   *time.Time
}

// EventRepository 定義活動資料存取的介面。
// 底層可以是 MySQL、Elasticsearch 或其他實作。
type EventRepository interface {
	Save(e *Event) error
	FindByID(id int) (*Event, bool)
	Search(q EventQuery) []*Event
	Update(e *Event) error
	Delete(id int)
}
