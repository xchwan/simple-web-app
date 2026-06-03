package infra

import (
	"os"
	"strings"

	"github.com/segmentio/kafka-go"
)

const BookingRequestsTopic = "booking-requests"

// KafkaBrokers 從環境變數讀取 broker 位址，預設 localhost:9092。
func KafkaBrokers() []string {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}
	return strings.Split(brokers, ",")
}

// NewKafkaWriter 建立一個以 key hash 分配 partition 的 Kafka producer。
func NewKafkaWriter(brokers []string, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
	}
}

// NewKafkaReader 建立一個 consumer reader。
func NewKafkaReader(brokers []string, topic, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
}
