package ticket

import (
	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin/apidoc"
	"github.com/xchwan/simple-web-framework/scope"
	"github.com/xchwan/simple-web-app/internal/event"
	"github.com/xchwan/simple-web-app/internal/user"
	"gorm.io/gorm"
)

// SetupRoutes 向 router 註冊 ticket 相關的依賴與路由。
// 錯誤對應由 main.go 統一在 ExceptionMapperPlugin 設定。
func SetupRoutes(router *framework.Router, database *gorm.DB) {
	ticketDB := NewTicketDB(database)
	eventDB := event.NewEventDB(database)
	router.Bind("ticketService", func() any {
		return NewTicketService(ticketDB, eventDB)
	}, scope.NewHttpRequestScope())

	h := NewTicketHandler()

	protected := router.Group("/api", user.Auth)
	protected.POST("/events/{eventId}/tickets", apidoc.Doc[BatchCreateTicketsRequest, []TicketResponse](h.BatchCreate, "Batch create tickets for an event"))
	protected.GET("/events/{eventId}/tickets", apidoc.Doc[apidoc.NoBody, []TicketResponse](h.ListByEvent, "List tickets for an event"))
	protected.GET("/tickets/{ticketId}", apidoc.Doc[apidoc.NoBody, TicketResponse](h.Get, "Get ticket by ID"))
}
