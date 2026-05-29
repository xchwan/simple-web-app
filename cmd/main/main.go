package main

import (
	"log"

	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin"
	"github.com/xchwan/simple-web-framework/plugin/apidoc"
	"github.com/xchwan/simple-web-app/internal/booking"
	"github.com/xchwan/simple-web-app/internal/db"
	"github.com/xchwan/simple-web-app/internal/event"
	"github.com/xchwan/simple-web-app/internal/ticket"
	"github.com/xchwan/simple-web-app/internal/user"
	"github.com/xchwan/simple-web-app/internal/wallet"
)

func main() {
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("DB 連線失敗: %v", err)
	}

	if err := db.Migrate(database); err != nil {
		log.Fatalf("Migration 失敗: %v", err)
	}

	rdb := db.ConnectRedis()
	esClient := db.ConnectElastic()

	docs := apidoc.NewDocPlugin()
	mapper := plugin.NewExceptionMapperPlugin()

	router := framework.NewRouter()
	router.AddPlugin(docs)
	router.AddPlugin(mapper)

	user.SetupRoutes(router, database, rdb, mapper)
	wallet.SetupRoutes(router, database, mapper)
	event.SetupRoutes(router, database, esClient, mapper)
	ticket.SetupRoutes(router, database, mapper)
	booking.SetupRoutes(router, database, mapper)

	router.GET("/docs", docs.UIHandler())
	router.GET("/openapi.json", docs.SpecHandler())

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
