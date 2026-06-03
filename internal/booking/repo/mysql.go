package bookingrepo

import "gorm.io/gorm"

// MySQLBookingRepository 以 MySQL 實作 BookingRepository。
type MySQLBookingRepository struct {
	db *gorm.DB
}

// NewMySQLBookingRepository 建立一個 MySQLBookingRepository。
func NewMySQLBookingRepository(db *gorm.DB) *MySQLBookingRepository {
	return &MySQLBookingRepository{db: db}
}

// WithTx 回傳一個綁定到 tx 的新 repository 實例，供跨表交易使用。
func (r *MySQLBookingRepository) WithTx(tx *gorm.DB) *MySQLBookingRepository {
	return &MySQLBookingRepository{db: tx}
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

// Save 建立一筆新訂票紀錄。
func (r *MySQLBookingRepository) Save(b *Booking) error {
	return r.db.Create(b).Error
}

// UpdateStatus 更新訂票狀態。
func (r *MySQLBookingRepository) UpdateStatus(id int, status BookingStatus) error {
	return r.db.Model(&Booking{}).Where("id = ?", id).Update("status", status).Error
}

// Cancel 原子性地將訂票標記為已取消（僅 status='confirmed' 時才更新）。
// ok=false 表示已是取消狀態（非 DB 錯誤）。
func (r *MySQLBookingRepository) Cancel(id int) (ok bool, err error) {
	result := r.db.Exec(
		"UPDATE bookings SET status = ? WHERE id = ? AND status = ?",
		StatusCancelled, id, StatusConfirmed,
	)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
