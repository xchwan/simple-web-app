package main

import (
	"log"

	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin"
	"github.com/xchwan/simple-web-app/internal/db"
	"github.com/xchwan/simple-web-app/internal/user"
)

func main() {
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("DB 連線失敗: %v", err)
	}

	rdb := db.ConnectRedis()

	docs := plugin.NewDocPlugin()

	router := framework.NewRouter()
	router.AddPlugin(docs)
	router.Bind("db", func() any { return database })
	router.Bind("redis", func() any { return rdb })

	user.SetupRoutes(router, docs)

	router.GET("/docs", docs.UIHandler())
	router.GET("/openapi.json", docs.SpecHandler())

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
