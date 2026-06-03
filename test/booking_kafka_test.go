package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin"
	bookingpkg "github.com/xchwan/simple-web-app/internal/booking"
	"github.com/xchwan/simple-web-app/internal/event"
	"github.com/xchwan/simple-web-app/internal/infra"
	"github.com/xchwan/simple-web-app/internal/ticket"
	"github.com/xchwan/simple-web-app/internal/user"
	"github.com/xchwan/simple-web-app/internal/wallet"
)

// skipIfNoKafka 若 KAFKA_BROKERS 未設定則跳過測試。
func skipIfNoKafka(t *testing.T) {
	t.Helper()
	if testKafkaBrokers == nil {
		t.Skip("Kafka not available（需傳入 KAFKA_BROKERS 環境變數）")
	}
}

// kafkaTestEnv 封裝 Kafka 整合測試所需的 router 與 writer。
type kafkaTestEnv struct {
	handler http.Handler
	writer  *kafka.Writer // 可直接 produce 訊息（用於 idempotency 測試）
}

// newKafkaTestEnv 建立包含真實 Kafka writer 的 router，並啟動 consumer goroutine。
//
// 隔離策略：
//   - 不使用 GroupID（避免 consumer group join 的延遲造成 race condition）
//   - 用 DialLeader + ReadLastOffset + SetOffset 精確記錄「test 開始前的 offset」
//   - Consumer 只讀 offset >= latestOffset 的訊息（即本次 test 產生的訊息）
//   - 前幾個 test 的殘留訊息在較早 offset，不會被讀到
func newKafkaTestEnv(t *testing.T) *kafkaTestEnv {
	t.Helper()

	// 在任何訊息產生前，取得目前 partition 0 的最新 offset。
	// 這樣 SetOffset 之後 consumer 只會讀到「這行之後」產生的訊息。
	conn, err := kafka.DialLeader(context.Background(), "tcp", testKafkaBrokers[0], infra.BookingTopic(), 0)
	if err != nil {
		t.Fatalf("kafka dial leader: %v", err)
	}
	latestOffset, err := conn.ReadLastOffset()
	conn.Close()
	if err != nil {
		t.Fatalf("kafka read offset: %v", err)
	}

	// 短 WriteTimeout 避免 writer.Close() 在 cleanup 時 block 過久
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(testKafkaBrokers...),
		Topic:                  infra.BookingTopic(),
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
		WriteTimeout:           3 * time.Second,
	}

	// 不使用 GroupID，直接用 SetOffset 定位（無 group join 延遲）
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  testKafkaBrokers,
		Topic:    infra.BookingTopic(),
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	if err := reader.SetOffset(latestOffset); err != nil {
		t.Fatalf("kafka set offset: %v", err)
	}

	mapper := plugin.NewExceptionMapperPlugin()
	router := framework.NewRouter()
	router.AddPlugin(mapper)
	user.SetupRoutes(router, testDB, testRDB, mapper)
	wallet.SetupRoutes(router, testDB, mapper)
	event.SetupRoutes(router, testDB, nil, mapper)
	ticket.SetupRoutes(router, testDB, mapper)
	consumer := bookingpkg.SetupRoutes(router, testDB, testRDB, writer, reader, mapper)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	go consumer.Run(ctx)

	t.Cleanup(func() {
		cancel()
		time.Sleep(200 * time.Millisecond) // 讓 consumer goroutine 有時間退出
		writer.Close()
		reader.Close()
	})

	return &kafkaTestEnv{handler: router, writer: writer}
}

// waitForBookingStatus 輪詢 DB 直到 booking 達到預期狀態（或逾時）。
func waitForBookingStatus(t *testing.T, bookingID int, expected string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var status string
		testDB.Raw("SELECT status FROM bookings WHERE id = ?", bookingID).Scan(&status)
		if status == expected {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	var got string
	testDB.Raw("SELECT status FROM bookings WHERE id = ?", bookingID).Scan(&got)
	t.Fatalf("booking %d: expected status %q, got %q after %v", bookingID, expected, got, timeout)
}

// ===== Kafka 整合測試 =====

// TestKafka_CreateBooking_Confirmed 驗證非同步訂票的完整流程：
// POST /bookings → 202 pending → Kafka → consumer → DB status = confirmed。
func TestKafka_CreateBooking_Confirmed(t *testing.T) {
	skipIfNoKafka(t)
	withCleanDB(t)
	env := newKafkaTestEnv(t)

	token, _ := registerAndLogin(t, env.handler, "alice@example.com", "Alice", "pass1234")
	ticketID, walletID, _ := setupBookingFixture(t, env.handler, token, 1000)

	// POST /bookings → 應得 202 Accepted，status = pending
	w := request(t, env.handler, http.MethodPost, "/api/bookings", map[string]any{
		"ticketId": ticketID,
		"walletId": walletID,
	}, token)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}
	resp := decode[map[string]any](t, w)
	bookingID := int(resp["id"].(float64))
	if resp["status"] != "pending" {
		t.Errorf("expected pending, got %v", resp["status"])
	}

	// consumer 非同步處理，等待 DB 狀態變成 confirmed
	waitForBookingStatus(t, bookingID, "confirmed", 5*time.Second)
}

// TestKafka_CreateBooking_202ThenGetConfirmed 驗證 GET /bookings/{id} 最終回傳 confirmed。
func TestKafka_CreateBooking_202ThenGetConfirmed(t *testing.T) {
	skipIfNoKafka(t)
	withCleanDB(t)
	env := newKafkaTestEnv(t)

	token, _ := registerAndLogin(t, env.handler, "alice@example.com", "Alice", "pass1234")
	ticketID, walletID, _ := setupBookingFixture(t, env.handler, token, 1000)

	w := request(t, env.handler, http.MethodPost, "/api/bookings", map[string]any{
		"ticketId": ticketID,
		"walletId": walletID,
	}, token)
	resp := decode[map[string]any](t, w)
	bookingID := int(resp["id"].(float64))

	// 等 consumer 處理完後，透過 GET 確認最終狀態
	waitForBookingStatus(t, bookingID, "confirmed", 5*time.Second)

	wGet := request(t, env.handler, http.MethodGet, fmt.Sprintf("/api/bookings/%d", bookingID), nil, token)
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wGet.Code)
	}
	getResp := decode[map[string]any](t, wGet)
	if getResp["status"] != "confirmed" {
		t.Errorf("expected confirmed, got %v", getResp["status"])
	}
}

// TestKafka_Idempotency 驗證 Kafka redelivery 的冪等性保護：
// consumer 收到已 confirmed 的 bookingID 訊息，不會把狀態改成 failed。
func TestKafka_Idempotency(t *testing.T) {
	skipIfNoKafka(t)
	withCleanDB(t)
	env := newKafkaTestEnv(t)

	token, _ := registerAndLogin(t, env.handler, "alice@example.com", "Alice", "pass1234")
	ticketID, walletID, _ := setupBookingFixture(t, env.handler, token, 1000)

	// 正常訂票，等待 confirmed
	w := request(t, env.handler, http.MethodPost, "/api/bookings", map[string]any{
		"ticketId": ticketID,
		"walletId": walletID,
	}, token)
	resp := decode[map[string]any](t, w)
	bookingID := int(resp["id"].(float64))
	waitForBookingStatus(t, bookingID, "confirmed", 5*time.Second)

	// 模擬 Kafka redelivery（crash-before-commit 場景）：
	// 直接 produce 一條相同 bookingID 的訊息到 topic
	payload, _ := json.Marshal(map[string]int{
		"bookingId": bookingID,
		"ticketId":  ticketID,
		"walletId":  walletID,
		"callerId":  0,
	})
	if err := env.writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(fmt.Sprintf("%d", ticketID)),
		Value: payload,
	}); err != nil {
		t.Fatalf("failed to produce redelivery message: %v", err)
	}

	// 等 consumer 處理完重複訊息
	time.Sleep(500 * time.Millisecond)

	// 狀態必須還是 confirmed，不能因為重複處理變成 failed
	var status string
	testDB.Raw("SELECT status FROM bookings WHERE id = ?", bookingID).Scan(&status)
	if status != "confirmed" {
		t.Errorf("idempotency broken: expected confirmed after redelivery, got %s", status)
	}
}

// TestKafka_SameTicket_SecondRequestRejected 驗證 Redis 閘門在 Kafka 路徑下仍生效：
// 同一張票第二個請求被 Redis SETNX 拒絕（409），不會進到 Kafka。
func TestKafka_SameTicket_SecondRequestRejected(t *testing.T) {
	skipIfNoKafka(t)
	withCleanDB(t)
	env := newKafkaTestEnv(t)

	aliceToken, _ := registerAndLogin(t, env.handler, "alice@example.com", "Alice", "pass1234")
	bobToken, _ := registerAndLogin(t, env.handler, "bob@example.com", "Bobby", "pass1234")

	// 同一張票
	ticketID, aliceWalletID, _ := setupBookingFixture(t, env.handler, aliceToken, 1000)

	// Bob 也建一個錢包
	bobWallet := createWallet(t, env.handler, bobToken, "Bob Wallet")
	bobWalletID := int(bobWallet["id"].(float64))
	request(t, env.handler, http.MethodPost, fmt.Sprintf("/api/wallets/%d/deposit", bobWalletID), map[string]any{"amount": 1000}, bobToken)

	// Alice 先搶
	w1 := request(t, env.handler, http.MethodPost, "/api/bookings", map[string]any{
		"ticketId": ticketID,
		"walletId": aliceWalletID,
	}, aliceToken)
	if w1.Code != http.StatusAccepted {
		t.Fatalf("alice: expected 202, got %d", w1.Code)
	}

	// Bob 搶同一張票 → Redis SETNX 失敗 → 409
	w2 := request(t, env.handler, http.MethodPost, "/api/bookings", map[string]any{
		"ticketId": ticketID,
		"walletId": bobWalletID,
	}, bobToken)
	if w2.Code != http.StatusConflict {
		t.Errorf("bob: expected 409 conflict, got %d", w2.Code)
	}

	// Alice 的 booking 最終 confirmed
	aliceResp := decode[map[string]any](t, w1)
	waitForBookingStatus(t, int(aliceResp["id"].(float64)), "confirmed", 5*time.Second)
}
