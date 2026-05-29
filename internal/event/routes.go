package event

import (
	"net/http"

	"github.com/elastic/go-elasticsearch/v8"
	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin"
	"github.com/xchwan/simple-web-framework/plugin/apidoc"
	"github.com/xchwan/simple-web-framework/scope"
	eventdb "github.com/xchwan/simple-web-app/internal/event/db"
	"github.com/xchwan/simple-web-app/internal/user"
	"gorm.io/gorm"
)

// SetupRoutes 向 router 註冊 event 相關的依賴與路由。
//
// esClient 可為 nil，表示不啟用 Elasticsearch；
// 傳入有效 client 時，Search 走 ES 全文索引，其餘寫入同步雙寫 MySQL + ES。
func SetupRoutes(router *framework.Router, database *gorm.DB, esClient *elasticsearch.Client, mapper *plugin.ExceptionMapperPlugin) {
	mapper.
		On(ErrNotFound, http.StatusNotFound, "Event not found").
		On(ErrForbidden, http.StatusForbidden, "Forbidden").
		On(ErrNameFormatInvalid, http.StatusBadRequest, "Event name format invalid").
		On(ErrStartAtInvalid, http.StatusBadRequest, "Event start time invalid")

	mysqlRepo := eventdb.NewMySQLRepository(database)
	var repo eventdb.EventRepository = mysqlRepo
	if esClient != nil {
		repo = eventdb.NewElasticRepository(esClient, mysqlRepo)
	}

	router.Bind("eventService", func() any {
		return NewEventService(repo)
	}, scope.NewHttpRequestScope())

	h := NewEventHandler()

	protected := router.Group("/api", user.Auth)
	protected.POST("/events", apidoc.Doc[CreateEventRequest, EventResponse](h.Create, "Create an event"))
	protected.GET("/events", apidoc.Doc[apidoc.NoBody, []EventResponse](h.Search, "List / search events"))
	protected.GET("/events/{eventId}", apidoc.Doc[apidoc.NoBody, EventResponse](h.Get, "Get event by ID"))
	protected.PATCH("/events/{eventId}", apidoc.Doc[UpdateEventRequest, EventResponse](h.Update, "Update event"))
	protected.DELETE("/events/{eventId}", apidoc.Doc[apidoc.NoBody, apidoc.NoBody](h.Delete, "Delete event"))
}
