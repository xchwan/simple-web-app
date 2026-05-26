package event

import "time"

// Event 代表一個活動。
type Event struct {
	ID          int       `gorm:"primarykey;autoIncrement"`
	OrganizerID int       `gorm:"not null;index"`
	Name        string    `gorm:"not null;size:255"`
	Description string    `gorm:"type:text"`
	StartAt     time.Time `gorm:"not null"`
}
