package ticket

import (
	"net/http"
	"strconv"

	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-app/internal/user"
)

// TicketHandler 負責處理票券相關的 HTTP 請求。
type TicketHandler struct{}

// NewTicketHandler 建立一個 TicketHandler。
func NewTicketHandler() *TicketHandler {
	return &TicketHandler{}
}

func (h *TicketHandler) service(r *http.Request) *TicketService {
	return framework.Get[*TicketService](r, "ticketService")
}

// ===== Request / Response DTO =====

type TicketInputDTO struct {
	Seat  string  `json:"seat"`
	Price float64 `json:"price"`
}

type BatchCreateTicketsRequest struct {
	Tickets []TicketInputDTO `json:"tickets"`
}

type TicketResponse struct {
	ID      int          `json:"id"`
	EventID int          `json:"eventId"`
	Seat    string       `json:"seat"`
	Price   float64      `json:"price"`
	Status  TicketStatus `json:"status"`
}

// ===== Handlers =====

// BatchCreate 處理 POST /api/events/{eventId}/tickets。
func (h *TicketHandler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	caller := user.Caller(r)
	eventID, err := strconv.Atoi(framework.PathParam(r, "eventId"))
	if err != nil {
		framework.HandleError(w, r, ErrEventNotFound)
		return
	}
	var req BatchCreateTicketsRequest
	if err := framework.ParseOrRespond(w, r, &req); err != nil {
		return
	}
	inputs := make([]TicketInput, len(req.Tickets))
	for i, dto := range req.Tickets {
		inputs[i] = TicketInput(dto)
	}
	tickets, svcErr := h.service(r).BatchCreate(caller.ID, eventID, inputs)
	if svcErr != nil {
		framework.HandleError(w, r, svcErr)
		return
	}
	result := make([]TicketResponse, len(tickets))
	for i, t := range tickets {
		result[i] = toTicketResponse(t)
	}
	framework.Respond(w, r, http.StatusCreated, result)
}

// ListByEvent 處理 GET /api/events/{eventId}/tickets（可選 ?status=available|sold）。
func (h *TicketHandler) ListByEvent(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(framework.PathParam(r, "eventId"))
	if err != nil {
		framework.HandleError(w, r, ErrEventNotFound)
		return
	}
	var status *TicketStatus
	if s := r.URL.Query().Get("status"); s != "" {
		ts := TicketStatus(s)
		status = &ts
	}
	tickets := h.service(r).ListByEvent(eventID, status)
	result := make([]TicketResponse, len(tickets))
	for i, t := range tickets {
		result[i] = toTicketResponse(t)
	}
	framework.Respond(w, r, http.StatusOK, result)
}

// Get 處理 GET /api/tickets/{ticketId}。
func (h *TicketHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(framework.PathParam(r, "ticketId"))
	if err != nil {
		framework.HandleError(w, r, ErrNotFound)
		return
	}
	t, svcErr := h.service(r).GetByID(id)
	if svcErr != nil {
		framework.HandleError(w, r, svcErr)
		return
	}
	framework.Respond(w, r, http.StatusOK, toTicketResponse(t))
}

func toTicketResponse(t *Ticket) TicketResponse {
	return TicketResponse{
		ID:      t.ID,
		EventID: t.EventID,
		Seat:    t.Seat,
		Price:   t.Price,
		Status:  t.Status,
	}
}
