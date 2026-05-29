package booking

import "gorm.io/gorm"

// MySQLBookingRepository 負責訂票資料的 MySQL 存取。
type MySQLBookingRepository struct {
	db *gorm.DB
}

// NewMySQLBookingRepository 建立一個 MySQLBookingRepository。
func NewMySQLBookingRepository(db *gorm.DB) *MySQLBookingRepository {
	return &MySQLBookingRepository{db: db}
}

// FindByID 依訂票編號查詢。
func (r *MySQLBookingRepository) FindByID(id int) (*Booking, bool) {
	var b Booking
	if err := r.db.First(&b, id).Error; err != nil {
		return nil, false
	}
	return &b, true
}

// FindByUserID 查詢指定會員的所有訂票紀錄。
func (r *MySQLBookingRepository) FindByUserID(userID int) []*Booking {
	var bookings []*Booking
	r.db.Where("user_id = ?", userID).Find(&bookings)
	return bookings
}
