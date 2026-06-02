package booking

import (
	"net/http"

	"github.com/redis/go-redis/v9"
	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin"
	"github.com/xchwan/simple-web-framework/plugin/apidoc"
	"github.com/xchwan/simple-web-framework/scope"
	bookingrepo "github.com/xchwan/simple-web-app/internal/booking/repo"
	ticketrepo "github.com/xchwan/simple-web-app/internal/ticket/repo"
	"github.com/xchwan/simple-web-app/internal/user"
	walletrepo "github.com/xchwan/simple-web-app/internal/wallet/repo"
	"gorm.io/gorm"
)

// SetupRoutes 向 router 註冊 booking 相關的依賴與路由。
func SetupRoutes(router *framework.Router, database *gorm.DB, rdb *redis.Client, mapper *plugin.ExceptionMapperPlugin) {
	mapper.
		On(ErrNotFound, http.StatusNotFound, "Booking not found").
		On(ErrForbidden, http.StatusForbidden, "Forbidden").
		On(ErrTicketNotFound, http.StatusNotFound, "Ticket not found").
		On(ErrWalletNotFound, http.StatusNotFound, "Wallet not found").
		On(ErrTicketUnavailable, http.StatusConflict, "Ticket is not available").
		On(ErrInsufficientBalance, http.StatusUnprocessableEntity, "Insufficient balance").
		On(ErrAlreadyCancelled, http.StatusConflict, "Booking already cancelled")

	repo := bookingrepo.NewMySQLBookingRepository(database)
	ticketDB := ticketrepo.NewMySQLTicketRepository(database)
	walletDB := walletrepo.NewMySQLWalletRepository(database)

	router.Bind("bookingService", func() any {
		return NewBookingService(repo, ticketDB, walletDB, database, rdb)
	}, scope.NewHttpRequestScope())

	h := NewBookingHandler()

	protected := router.Group("/api", user.Auth)
	protected.POST("/bookings", apidoc.Doc[CreateBookingRequest, BookingResponse](h.Create, "Create a booking"))
	protected.GET("/bookings", apidoc.Doc[apidoc.NoBody, []BookingResponse](h.List, "List my bookings"))
	protected.GET("/bookings/{bookingId}", apidoc.Doc[apidoc.NoBody, BookingResponse](h.Get, "Get booking by ID"))
	protected.DELETE("/bookings/{bookingId}", apidoc.Doc[apidoc.NoBody, BookingResponse](h.Cancel, "Cancel a booking"))
}
