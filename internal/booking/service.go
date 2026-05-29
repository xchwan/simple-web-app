package booking

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

// BookingService 負責訂票相關的業務邏輯。
// 注意：訂票（建立）涉及跨表格交易，待後續實作。
type BookingService struct {
	db *MySQLBookingRepository
}

// NewBookingService 建立一個 BookingService。
func NewBookingService(db *MySQLBookingRepository) *BookingService {
	return &BookingService{db: db}
}

// GetByID 取得訂票紀錄，僅本人可存取。
func (s *BookingService) GetByID(callerID, bookingID int) (*Booking, error) {
	b, exists := s.db.FindByID(bookingID)
	if !exists {
		return nil, ErrNotFound
	}
	if b.UserID != callerID {
		return nil, ErrForbidden
	}
	return b, nil
}

// ListByUser 列出指定會員的所有訂票紀錄。
func (s *BookingService) ListByUser(userID int) []*Booking {
	return s.db.FindByUserID(userID)
}
