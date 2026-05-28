package booking

import "gorm.io/gorm"

// BookingDB 負責訂票資料的 MySQL 存取。
type BookingDB struct {
	db *gorm.DB
}

// NewBookingDB 建立一個 BookingDB。
func NewBookingDB(db *gorm.DB) *BookingDB {
	return &BookingDB{db: db}
}

// FindByID 依訂票編號查詢。
func (r *BookingDB) FindByID(id int) (*Booking, bool) {
	var b Booking
	if err := r.db.First(&b, id).Error; err != nil {
		return nil, false
	}
	return &b, true
}

// FindByUserID 查詢指定會員的所有訂票紀錄。
func (r *BookingDB) FindByUserID(userID int) []*Booking {
	var bookings []*Booking
	r.db.Where("user_id = ?", userID).Find(&bookings)
	return bookings
}
