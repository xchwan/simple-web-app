package main

import (
	"log"

	"github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-app/internal/user"
)

func main() {
	router := framework.NewRouter()

	user.Register(router)

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
