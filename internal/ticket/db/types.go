package ticketdb

// TicketStatus 代表票的狀態。
type TicketStatus string

const (
	StatusAvailable TicketStatus = "available"
	StatusSold      TicketStatus = "sold"
)

// Ticket 代表活動中的一張票。
type Ticket struct {
	ID      int          `gorm:"primarykey;autoIncrement"`
	EventID int          `gorm:"not null;index"`
	Seat    string       `gorm:"not null;size:50"`
	Price   float64      `gorm:"type:decimal(10,2);not null"`
	Status  TicketStatus `gorm:"type:enum('available','sold');not null;default:'available'"`
}

// TicketRepository 定義票券資料存取的介面。
type TicketRepository interface {
	SaveAll(tickets []*Ticket) error
	FindByID(id int) (*Ticket, bool)
	FindByEventID(eventID int, status *TicketStatus) []*Ticket
}
