package booking

import bookingdb "github.com/xchwan/simple-web-app/internal/booking/db"

// BookingService 負責訂票相關的業務邏輯。
// 注意：訂票（建立）涉及跨表格交易，待後續實作。
type BookingService struct {
	db bookingdb.BookingRepository
}

// NewBookingService 建立一個 BookingService。
func NewBookingService(db bookingdb.BookingRepository) *BookingService {
	return &BookingService{db: db}
}

// GetByID 取得訂票紀錄，僅本人可存取。
func (s *BookingService) GetByID(callerID, bookingID int) (*bookingdb.Booking, error) {
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
func (s *BookingService) ListByUser(userID int) []*bookingdb.Booking {
	return s.db.FindByUserID(userID)
}
