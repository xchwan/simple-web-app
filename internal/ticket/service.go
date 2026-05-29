package ticket

import (
	eventdb "github.com/xchwan/simple-web-app/internal/event/db"
	ticketdb "github.com/xchwan/simple-web-app/internal/ticket/db"
)

// TicketInput 代表建立單張票券所需的資料（service 層輸入型別）。
type TicketInput struct {
	Seat  string
	Price float64
}

// TicketService 負責票券相關的業務邏輯。
type TicketService struct {
	db        ticketdb.TicketRepository
	eventRepo eventdb.EventRepository
}

// NewTicketService 建立一個 TicketService。
func NewTicketService(db ticketdb.TicketRepository, eventRepo eventdb.EventRepository) *TicketService {
	return &TicketService{db: db, eventRepo: eventRepo}
}

// BatchCreate 批次建立票券，僅活動主辦人可操作。
func (s *TicketService) BatchCreate(callerID, eventID int, inputs []TicketInput) ([]*ticketdb.Ticket, error) {
	e, exists := s.eventRepo.FindByID(eventID)
	if !exists {
		return nil, ErrEventNotFound
	}
	if e.OrganizerID != callerID {
		return nil, ErrForbidden
	}
	tickets := make([]*ticketdb.Ticket, 0, len(inputs))
	for _, in := range inputs {
		if !validateLength(in.Seat, 1, 50) {
			return nil, ErrSeatFormatInvalid
		}
		if in.Price < 0 {
			return nil, ErrPriceInvalid
		}
		tickets = append(tickets, &ticketdb.Ticket{
			EventID: eventID,
			Seat:    in.Seat,
			Price:   in.Price,
			Status:  ticketdb.StatusAvailable,
		})
	}
	if err := s.db.SaveAll(tickets); err != nil {
		return nil, err
	}
	return tickets, nil
}

// GetByID 依編號取得票券。
func (s *TicketService) GetByID(id int) (*ticketdb.Ticket, error) {
	t, exists := s.db.FindByID(id)
	if !exists {
		return nil, ErrNotFound
	}
	return t, nil
}

// ListByEvent 列出指定活動的票券，可選依狀態過濾。
func (s *TicketService) ListByEvent(eventID int, status *ticketdb.TicketStatus) []*ticketdb.Ticket {
	return s.db.FindByEventID(eventID, status)
}

func validateLength(s string, min, max int) bool {
	n := len([]rune(s))
	return n >= min && n <= max
}
