package booking

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

// BookingConsumer 從 Kafka 消費訂票請求，交由 BookingService 處理。
type BookingConsumer struct {
	reader  *kafka.Reader
	service *BookingService
}

// NewBookingConsumer 建立一個 BookingConsumer。
func NewBookingConsumer(reader *kafka.Reader, service *BookingService) *BookingConsumer {
	return &BookingConsumer{reader: reader, service: service}
}

// Run 啟動消費迴圈，直到 ctx 被取消。
// 每個 partition 由單一 goroutine 消費，相同 ticketID 的訊息天然序列化。
func (c *BookingConsumer) Run(ctx context.Context) {
	log.Println("[booking consumer] started")
	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("[booking consumer] stopped")
				return
			}
			log.Printf("[booking consumer] fetch error: %v", err)
			continue
		}

		var msg bookingMessage
		if err := json.Unmarshal(m.Value, &msg); err != nil {
			log.Printf("[booking consumer] decode error: %v", err)
		} else {
			c.service.processBooking(msg)
		}

		// 無論成功或失敗都 commit，避免無限重試壞訊息
		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("[booking consumer] commit error: %v", err)
		}
	}
}
