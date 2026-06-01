package ticketrepo

import "gorm.io/gorm"

// MySQLTicketRepository 以 MySQL 實作 TicketRepository。
type MySQLTicketRepository struct {
	db *gorm.DB
}

// NewMySQLTicketRepository 建立一個 MySQLTicketRepository。
func NewMySQLTicketRepository(db *gorm.DB) *MySQLTicketRepository {
	return &MySQLTicketRepository{db: db}
}

// WithTx 回傳一個綁定到 tx 的新 repository 實例，供跨表交易使用。
func (r *MySQLTicketRepository) WithTx(tx *gorm.DB) *MySQLTicketRepository {
	return &MySQLTicketRepository{db: tx}
}

// SaveAll 批次新增票券。
func (r *MySQLTicketRepository) SaveAll(tickets []*Ticket) error {
	return r.db.Create(&tickets).Error
}

// FindByID 依票券編號查詢。
func (r *MySQLTicketRepository) FindByID(id int) (*Ticket, bool) {
	var t Ticket
	if err := r.db.First(&t, id).Error; err != nil {
		return nil, false
	}
	return &t, true
}

// FindByEventID 查詢指定活動的所有票券，可選依狀態過濾。
func (r *MySQLTicketRepository) FindByEventID(eventID int, status *TicketStatus) []*Ticket {
	var tickets []*Ticket
	query := r.db.Where("event_id = ?", eventID)
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	query.Find(&tickets)
	return tickets
}

// CountByEvent 回傳指定活動已建立的票券總數。
func (r *MySQLTicketRepository) CountByEvent(eventID int) int {
	var count int64
	r.db.Model(&Ticket{}).Where("event_id = ?", eventID).Count(&count)
	return int(count)
}

// MarkSold 原子性地將票券標記為已售出（僅 status='available' 時才更新）。
// ok=false 表示票券已被搶走（非 DB 錯誤）。
func (r *MySQLTicketRepository) MarkSold(id int) (ok bool, err error) {
	result := r.db.Exec(
		"UPDATE tickets SET status = ? WHERE id = ? AND status = ?",
		StatusSold, id, StatusAvailable,
	)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
