package infra

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/segmentio/kafka-go"
)

const BookingRequestsTopic = "booking-requests"
const BookingTopicPartitions = 10

// KafkaBrokers 從環境變數讀取 broker 位址，預設 localhost:9092。
func KafkaBrokers() []string {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}
	return strings.Split(brokers, ",")
}

// EnsureTopic 確保 topic 存在且有指定的 partition 數。
// topic 已存在時不報錯（無法修改既有 partition 數，需手動處理）。
func EnsureTopic(brokers []string, topic string, numPartitions int) error {
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("kafka dial: %w", err)
	}
	defer conn.Close()

	// topic 建立指令要送給 controller broker
	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("kafka controller: %w", err)
	}
	ctrlConn, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return fmt.Errorf("kafka controller dial: %w", err)
	}
	defer ctrlConn.Close()

	topicErrors := ctrlConn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     numPartitions,
		ReplicationFactor: 1,
	})
	for _, te := range topicErrors {
		if te.Err != nil && te.Err != kafka.TopicAlreadyExists {
			return fmt.Errorf("kafka topic error %s: %w", te.Topic, te.Err)
		}
	}
	return nil
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
