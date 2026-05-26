package main

import (
	"log"

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
		log.Fatalf("連線資料庫失敗: %v", err)
	}

	if err := database.AutoMigrate(
		&user.User{},
		&wallet.Wallet{},
		&event.Event{},
		&ticket.Ticket{},
		&booking.Booking{},
	); err != nil {
		log.Fatalf("Migration 失敗: %v", err)
	}

	// token 欄位已移至 Redis，從 users 表移除
	if database.Migrator().HasColumn(&user.User{}, "token") {
		if err := database.Migrator().DropColumn(&user.User{}, "token"); err != nil {
			log.Fatalf("移除 token 欄位失敗: %v", err)
		}
		log.Println("已移除 users.token 欄位")
	}

	log.Println("Migration 完成")
}
