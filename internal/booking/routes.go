package booking

import (
	"net/http"

	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin"
	"github.com/xchwan/simple-web-framework/plugin/apidoc"
	"github.com/xchwan/simple-web-framework/scope"
	"github.com/xchwan/simple-web-app/internal/user"
	"gorm.io/gorm"
)

// SetupRoutes 向 router 註冊 booking 相關的依賴與路由。
// 注意：訂票（create）需要跨表格交易（ticket 狀態 + wallet 扣款），待後續實作。
func SetupRoutes(router *framework.Router, database *gorm.DB, mapper *plugin.ExceptionMapperPlugin) {
	mapper.
		On(ErrNotFound, http.StatusNotFound, "Booking not found").
		On(ErrForbidden, http.StatusForbidden, "Forbidden")

	bookingDB := NewBookingDB(database)
	router.Bind("bookingService", func() any {
		return NewBookingService(bookingDB)
	}, scope.NewHttpRequestScope())

	h := NewBookingHandler()

	protected := router.Group("/api", user.Auth)
	protected.GET("/bookings", apidoc.Doc[apidoc.NoBody, []BookingResponse](h.List, "List my bookings"))
	protected.GET("/bookings/{bookingId}", apidoc.Doc[apidoc.NoBody, BookingResponse](h.Get, "Get booking by ID"))
}
