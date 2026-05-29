package main

import (
	"log"
	"net/http"

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

	docs := apidoc.NewDocPlugin()

	router := framework.NewRouter()
	router.AddPlugin(docs)
	router.AddPlugin(
		plugin.NewExceptionMapperPlugin().
			// user
			On(user.ErrEmailDuplicate, http.StatusBadRequest, "Duplicate email").
			On(user.ErrRegisterFormatInvalid, http.StatusBadRequest, "Registration format invalid").
			On(user.ErrCredentialsInvalid, http.StatusBadRequest, "Credentials invalid").
			On(user.ErrLoginFormatInvalid, http.StatusBadRequest, "Login format invalid").
			On(user.ErrTokenInvalid, http.StatusUnauthorized, "Can't authenticate who you are").
			On(user.ErrForbidden, http.StatusForbidden, "Forbidden").
			On(user.ErrNameFormatInvalid, http.StatusBadRequest, "Name format invalid").
			// wallet
			On(wallet.ErrNotFound, http.StatusNotFound, "Wallet not found").
			On(wallet.ErrForbidden, http.StatusForbidden, "Forbidden").
			On(wallet.ErrNameFormatInvalid, http.StatusBadRequest, "Wallet name format invalid").
			// event
			On(event.ErrNotFound, http.StatusNotFound, "Event not found").
			On(event.ErrForbidden, http.StatusForbidden, "Forbidden").
			On(event.ErrNameFormatInvalid, http.StatusBadRequest, "Event name format invalid").
			On(event.ErrStartAtInvalid, http.StatusBadRequest, "Event start time invalid").
			// ticket
			On(ticket.ErrNotFound, http.StatusNotFound, "Ticket not found").
			On(ticket.ErrEventNotFound, http.StatusNotFound, "Event not found").
			On(ticket.ErrForbidden, http.StatusForbidden, "Forbidden").
			On(ticket.ErrSeatFormatInvalid, http.StatusBadRequest, "Seat format invalid").
			On(ticket.ErrPriceInvalid, http.StatusBadRequest, "Price must be >= 0").
			// booking
			On(booking.ErrNotFound, http.StatusNotFound, "Booking not found").
			On(booking.ErrForbidden, http.StatusForbidden, "Forbidden"),
	)

	user.SetupRoutes(router, database, rdb)
	wallet.SetupRoutes(router, database)
	event.SetupRoutes(router, database)
	ticket.SetupRoutes(router, database)
	booking.SetupRoutes(router, database)

	router.GET("/docs", docs.UIHandler())
	router.GET("/openapi.json", docs.SpecHandler())

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
