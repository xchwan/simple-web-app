package main

import (
	"log"

	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-app/internal/db"
	"github.com/xchwan/simple-web-app/internal/user"
)

func main() {
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("DB 連線失敗: %v", err)
	}

	rdb := db.ConnectRedis()

	router := framework.NewRouter()
	router.Bind("db", func() any { return database })
	router.Bind("redis", func() any { return rdb })

	user.SetupRoutes(router)

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
