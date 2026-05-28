package ticket

import "gorm.io/gorm"

// TicketDB 負責票券資料的 MySQL 存取。
type TicketDB struct {
	db *gorm.DB
}

// NewTicketDB 建立一個 TicketDB。
func NewTicketDB(db *gorm.DB) *TicketDB {
	return &TicketDB{db: db}
}

// SaveAll 批次新增票券。
func (r *TicketDB) SaveAll(tickets []*Ticket) error {
	return r.db.Create(&tickets).Error
}

// FindByID 依票券編號查詢。
func (r *TicketDB) FindByID(id int) (*Ticket, bool) {
	var t Ticket
	if err := r.db.First(&t, id).Error; err != nil {
		return nil, false
	}
	return &t, true
}

// FindByEventID 查詢指定活動的所有票券，可選依狀態過濾。
func (r *TicketDB) FindByEventID(eventID int, status *TicketStatus) []*Ticket {
	var tickets []*Ticket
	query := r.db.Where("event_id = ?", eventID)
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	query.Find(&tickets)
	return tickets
}
