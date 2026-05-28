package main

import (
	"log"

	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin/apidoc"
	"github.com/xchwan/simple-web-app/internal/db"
	"github.com/xchwan/simple-web-app/internal/user"
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

	docs := apidoc.NewDocPlugin()

	router := framework.NewRouter()
	router.AddPlugin(docs)

	user.SetupRoutes(router, database, rdb)

	router.GET("/docs", docs.UIHandler())
	router.GET("/openapi.json", docs.SpecHandler())

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
