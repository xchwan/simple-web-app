package booking

import (
	"net/http"
	"strconv"

	framework "github.com/xchwan/simple-web-framework"
	bookingrepo "github.com/xchwan/simple-web-app/internal/booking/repo"
	"github.com/xchwan/simple-web-app/internal/user"
)

// BookingHandler 負責處理訂票相關的 HTTP 請求。
type BookingHandler struct{}

// NewBookingHandler 建立一個 BookingHandler。
func NewBookingHandler() *BookingHandler {
	return &BookingHandler{}
}

func (h *BookingHandler) service(r *http.Request) *BookingService {
	return framework.Get[*BookingService](r, "bookingService")
}

// ===== Request / Response DTO =====

type CreateBookingRequest struct {
	TicketID int `json:"ticketId"`
	WalletID int `json:"walletId"`
}

type BookingResponse struct {
	ID       int                    `json:"id"`
	UserID   int                    `json:"userId"`
	TicketID int                    `json:"ticketId"`
	WalletID int                    `json:"walletId"`
	Status   bookingrepo.BookingStatus `json:"status"`
	BookedAt interface{}             `json:"bookedAt"`
}

// ===== Handlers =====

// Create 處理 POST /api/bookings。
func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
	caller := user.Caller(r)
	var req CreateBookingRequest
	if err := framework.ParseOrRespond(w, r, &req); err != nil {
		return
	}
	b, svcErr := h.service(r).Create(caller.ID, req.TicketID, req.WalletID)
	if svcErr != nil {
		framework.HandleError(w, r, svcErr)
		return
	}
	framework.Respond(w, r, http.StatusCreated, toBookingResponse(b))
}

// Get 處理 GET /api/bookings/{bookingId}。
func (h *BookingHandler) Get(w http.ResponseWriter, r *http.Request) {
	caller := user.Caller(r)
	id, err := strconv.Atoi(framework.PathParam(r, "bookingId"))
	if err != nil {
		framework.HandleError(w, r, ErrNotFound)
		return
	}
	b, svcErr := h.service(r).GetByID(caller.ID, id)
	if svcErr != nil {
		framework.HandleError(w, r, svcErr)
		return
	}
	framework.Respond(w, r, http.StatusOK, toBookingResponse(b))
}

// List 處理 GET /api/bookings。
func (h *BookingHandler) List(w http.ResponseWriter, r *http.Request) {
	caller := user.Caller(r)
	bookings := h.service(r).ListByUser(caller.ID)
	result := make([]BookingResponse, len(bookings))
	for i, b := range bookings {
		result[i] = toBookingResponse(b)
	}
	framework.Respond(w, r, http.StatusOK, result)
}

func toBookingResponse(b *bookingrepo.Booking) BookingResponse {
	return BookingResponse{
		ID:       b.ID,
		UserID:   b.UserID,
		TicketID: b.TicketID,
		WalletID: b.WalletID,
		Status:   b.Status,
		BookedAt: b.BookedAt,
	}
}
