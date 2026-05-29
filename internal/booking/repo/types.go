package bookingrepo

import "time"

// BookingStatus 代表訂票的狀態。
type BookingStatus string

const (
	StatusConfirmed BookingStatus = "confirmed"
	StatusCancelled BookingStatus = "cancelled"
)

// Booking 代表一筆訂票紀錄。
type Booking struct {
	ID       int           `gorm:"primarykey;autoIncrement"`
	UserID   int           `gorm:"not null;index"`
	TicketID int           `gorm:"not null;uniqueIndex"`
	WalletID int           `gorm:"not null;index"`
	Status   BookingStatus `gorm:"type:enum('confirmed','cancelled');not null;default:'confirmed'"`
	BookedAt time.Time     `gorm:"not null;autoCreateTime"`
}
