package ticket

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
