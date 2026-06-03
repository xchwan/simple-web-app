package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin"
	"github.com/xchwan/simple-web-framework/plugin/apidoc"
	"github.com/xchwan/simple-web-app/internal/booking"
	"github.com/xchwan/simple-web-app/internal/event"
	"github.com/xchwan/simple-web-app/internal/infra"
	"github.com/xchwan/simple-web-app/internal/ticket"
	"github.com/xchwan/simple-web-app/internal/user"
	"github.com/xchwan/simple-web-app/internal/wallet"
)

func main() {
	database, err := infra.Connect()
	if err != nil {
		log.Fatalf("DB 連線失敗: %v", err)
	}

	if err := infra.Migrate(database); err != nil {
		log.Fatalf("Migration 失敗: %v", err)
	}

	rdb := infra.ConnectRedis()
	esClient := infra.ConnectElastic()

	brokers := infra.KafkaBrokers()
	if err := infra.EnsureTopic(brokers, infra.BookingTopic(), infra.BookingTopicPartitions); err != nil {
		log.Printf("kafka topic setup warning: %v", err)
	}
	kafkaWriter := infra.NewKafkaWriter(brokers, infra.BookingTopic())
	kafkaReader := infra.NewKafkaReader(brokers, infra.BookingTopic(), "booking-consumer")
	defer kafkaWriter.Close()
	defer kafkaReader.Close()

	docs := apidoc.NewDocPlugin()
	mapper := plugin.NewExceptionMapperPlugin()

	router := framework.NewRouter()
	router.AddPlugin(docs)
	router.AddPlugin(mapper)

	user.SetupRoutes(router, database, rdb, mapper)
	wallet.SetupRoutes(router, database, mapper)
	event.SetupRoutes(router, database, esClient, mapper)
	ticket.SetupRoutes(router, database, mapper)
	// 統一由 main 管理 signal，ctx 傳給 router 和 consumer
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	consumer := booking.SetupRoutes(router, database, rdb, kafkaWriter, kafkaReader, mapper)

	// consumer 結束時關閉 done channel，方便等待
	consumerDone := make(chan struct{})
	go func() {
		consumer.Run(ctx)
		close(consumerDone)
	}()

	router.GET("/docs", docs.UIHandler())
	router.GET("/openapi.json", docs.SpecHandler())

	// router.Run 收到 ctx 取消後自行 graceful shutdown HTTP server
	if err := router.Run(ctx, ":8080"); err != nil {
		log.Printf("server shutdown: %v", err)
	}

	// HTTP server 關閉後，等 consumer 處理完當前訊息（最多 8 秒）
	select {
	case <-consumerDone:
		log.Println("consumer stopped cleanly")
	case <-time.After(8 * time.Second):
		log.Println("consumer shutdown timed out")
	}
}
