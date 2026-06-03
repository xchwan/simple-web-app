package booking

import (
	"net/http"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
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
// 回傳 BookingConsumer，由呼叫端以 goroutine 啟動。
func SetupRoutes(
	router *framework.Router,
	database *gorm.DB,
	rdb *redis.Client,
	kafkaWriter *kafka.Writer,
	kafkaReader *kafka.Reader,
	mapper *plugin.ExceptionMapperPlugin,
) *BookingConsumer {
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

	svc := NewBookingService(repo, ticketDB, walletDB, database, rdb, kafkaWriter)

	router.Bind("bookingService", func() any {
		return svc
	}, scope.NewHttpRequestScope())

	h := NewBookingHandler()

	protected := router.Group("/api", user.Auth)
	protected.POST("/bookings", apidoc.Doc[CreateBookingRequest, BookingResponse](h.Queue, "Create a booking (async via Kafka)"))
	protected.GET("/bookings", apidoc.Doc[apidoc.NoBody, []BookingResponse](h.List, "List my bookings"))
	protected.GET("/bookings/{bookingId}", apidoc.Doc[apidoc.NoBody, BookingResponse](h.Get, "Get booking by ID"))
	protected.DELETE("/bookings/{bookingId}", apidoc.Doc[apidoc.NoBody, BookingResponse](h.Cancel, "Cancel a booking"))

	return NewBookingConsumer(kafkaReader, svc)
}
