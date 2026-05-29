package event

import (
	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin/apidoc"
	"github.com/xchwan/simple-web-framework/scope"
	"github.com/xchwan/simple-web-app/internal/user"
	"gorm.io/gorm"
)

// SetupRoutes 向 router 註冊 event 相關的依賴與路由。
// 錯誤對應由 main.go 統一在 ExceptionMapperPlugin 設定。
func SetupRoutes(router *framework.Router, database *gorm.DB) {
	eventDB := NewEventDB(database)
	router.Bind("eventService", func() any {
		return NewEventService(eventDB)
	}, scope.NewHttpRequestScope())

	h := NewEventHandler()

	protected := router.Group("/api", user.Auth)
	protected.POST("/events", apidoc.Doc[CreateEventRequest, EventResponse](h.Create, "Create an event"))
	protected.GET("/events", apidoc.Doc[apidoc.NoBody, []EventResponse](h.Search, "List / search events"))
	protected.GET("/events/{eventId}", apidoc.Doc[apidoc.NoBody, EventResponse](h.Get, "Get event by ID"))
	protected.PATCH("/events/{eventId}", apidoc.Doc[UpdateEventRequest, EventResponse](h.Update, "Update event"))
	protected.DELETE("/events/{eventId}", apidoc.Doc[apidoc.NoBody, apidoc.NoBody](h.Delete, "Delete event"))
}
