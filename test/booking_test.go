package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// ===== E1：列出訂票 =====

func TestListBookings_Success(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	w := request(t, router, http.MethodGet, "/api/bookings", nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	bookings := decode[[]map[string]any](t, w)
	if len(bookings) != 0 {
		t.Errorf("expected empty list, got %d", len(bookings))
	}
}

func TestListBookings_Unauthenticated(t *testing.T) {
	w := request(t, newRouter(), http.MethodGet, "/api/bookings", nil, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ===== E2：取得訂票 =====

func TestGetBooking_NotFound(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	w := request(t, router, http.MethodGet, "/api/bookings/99999", nil, token)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetBooking_Unauthenticated(t *testing.T) {
	w := request(t, newRouter(), http.MethodGet, "/api/bookings/1", nil, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ===== E3：建立訂票 =====

// setupBookingFixture 建立測試所需的活動、票券、錢包，並存款。
// 回傳 ticketID、walletID 與票價。
func setupBookingFixture(t *testing.T, router http.Handler, token string, deposit float64) (ticketID, walletID int, price float64) {
	t.Helper()
	event := createEvent(t, router, token)
	eventID := int(event["id"].(float64))

	tickets := createTickets(t, router, token, eventID, []map[string]any{
		{"seat": "A1", "price": 500.0},
	})
	ticketID = int(tickets[0]["id"].(float64))
	price = tickets[0]["price"].(float64)

	wallet := createWallet(t, router, token, "Main Wallet")
	walletID = int(wallet["id"].(float64))

	if deposit > 0 {
		w := request(t, router, http.MethodPost, fmt.Sprintf("/api/wallets/%d/deposit", walletID), map[string]any{
			"amount": deposit,
		}, token)
		if w.Code != http.StatusOK {
			t.Fatalf("deposit failed: %d", w.Code)
		}
	}
	return ticketID, walletID, price
}

func TestCreateBooking_Success(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	ticketID, walletID, price := setupBookingFixture(t, router, token, 1000.0)

	w := request(t, router, http.MethodPost, "/api/bookings", map[string]any{
		"ticketId": ticketID,
		"walletId": walletID,
	}, token)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := decode[map[string]any](t, w)
	if resp["ticketId"] != float64(ticketID) {
		t.Errorf("ticketId mismatch: %v", resp["ticketId"])
	}
	if resp["status"] != "confirmed" {
		t.Errorf("expected status confirmed, got %v", resp["status"])
	}

	// 確認票券已標為 sold
	tw := request(t, router, http.MethodGet, fmt.Sprintf("/api/tickets/%d", ticketID), nil, token)
	ticket := decode[map[string]any](t, tw)
	if ticket["status"] != "sold" {
		t.Errorf("expected ticket sold, got %v", ticket["status"])
	}

	// 確認錢包已扣款
	ww := request(t, router, http.MethodGet, fmt.Sprintf("/api/wallets/%d", walletID), nil, token)
	wallet := decode[map[string]any](t, ww)
	expectedBalance := 1000.0 - price
	if wallet["balance"] != expectedBalance {
		t.Errorf("expected balance %.2f, got %v", expectedBalance, wallet["balance"])
	}
}

func TestCreateBooking_Unauthenticated(t *testing.T) {
	w := request(t, newRouter(), http.MethodPost, "/api/bookings", map[string]any{
		"ticketId": 1, "walletId": 1,
	}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCreateBooking_TicketNotFound(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	_, walletID, _ := setupBookingFixture(t, router, token, 1000.0)

	w := request(t, router, http.MethodPost, "/api/bookings", map[string]any{
		"ticketId": 99999,
		"walletId": walletID,
	}, token)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCreateBooking_WalletNotFound(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	ticketID, _, _ := setupBookingFixture(t, router, token, 0)

	w := request(t, router, http.MethodPost, "/api/bookings", map[string]any{
		"ticketId": ticketID,
		"walletId": 99999,
	}, token)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCreateBooking_WalletForbidden(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	aliceToken, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	bobToken, _ := registerAndLogin(t, router, "bob@example.com", "Bobby", "pass1234")

	// Alice 建立活動與票券
	ticketID, _, _ := setupBookingFixture(t, router, aliceToken, 0)

	// Bob 建立錢包並存款
	bobWallet := createWallet(t, router, bobToken, "Bob Wallet")
	bobWalletID := int(bobWallet["id"].(float64))
	request(t, router, http.MethodPost, fmt.Sprintf("/api/wallets/%d/deposit", bobWalletID), map[string]any{"amount": 1000.0}, bobToken)

	// Alice 嘗試用 Bob 的錢包訂票
	w := request(t, router, http.MethodPost, "/api/bookings", map[string]any{
		"ticketId": ticketID,
		"walletId": bobWalletID,
	}, aliceToken)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestCreateBooking_TicketUnavailable(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	ticketID, walletID, _ := setupBookingFixture(t, router, token, 2000.0)

	// 第一次訂票成功
	request(t, router, http.MethodPost, "/api/bookings", map[string]any{
		"ticketId": ticketID, "walletId": walletID,
	}, token)

	// 第二次應該失敗
	w := request(t, router, http.MethodPost, "/api/bookings", map[string]any{
		"ticketId": ticketID, "walletId": walletID,
	}, token)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestCreateBooking_InsufficientBalance(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	ticketID, walletID, _ := setupBookingFixture(t, router, token, 100.0) // 票價 500，只存 100

	w := request(t, router, http.MethodPost, "/api/bookings", map[string]any{
		"ticketId": ticketID,
		"walletId": walletID,
	}, token)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
}

// ===== E4：取消訂票 =====

func TestCancelBooking_Success(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	ticketID, walletID, price := setupBookingFixture(t, router, token, 1000.0)

	// 先訂票
	bw := request(t, router, http.MethodPost, "/api/bookings", map[string]any{
		"ticketId": ticketID, "walletId": walletID,
	}, token)
	booking := decode[map[string]any](t, bw)
	bookingID := int(booking["id"].(float64))

	// 取消
	w := request(t, router, http.MethodDelete, fmt.Sprintf("/api/bookings/%d", bookingID), nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := decode[map[string]any](t, w)
	if resp["status"] != "cancelled" {
		t.Errorf("expected status cancelled, got %v", resp["status"])
	}

	// 確認票券已退回 available
	tw := request(t, router, http.MethodGet, fmt.Sprintf("/api/tickets/%d", ticketID), nil, token)
	ticket := decode[map[string]any](t, tw)
	if ticket["status"] != "available" {
		t.Errorf("expected ticket available, got %v", ticket["status"])
	}

	// 確認退款
	ww := request(t, router, http.MethodGet, fmt.Sprintf("/api/wallets/%d", walletID), nil, token)
	wallet := decode[map[string]any](t, ww)
	if wallet["balance"] != 1000.0 {
		t.Errorf("expected balance refunded to 1000, got %v (price was %.2f)", wallet["balance"], price)
	}
}

func TestCancelBooking_Unauthenticated(t *testing.T) {
	w := request(t, newRouter(), http.MethodDelete, "/api/bookings/1", nil, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCancelBooking_NotFound(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	w := request(t, router, http.MethodDelete, "/api/bookings/99999", nil, token)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCancelBooking_Forbidden(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	aliceToken, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	bobToken, _ := registerAndLogin(t, router, "bob@example.com", "Bobby", "pass1234")
	ticketID, walletID, _ := setupBookingFixture(t, router, aliceToken, 1000.0)

	bw := request(t, router, http.MethodPost, "/api/bookings", map[string]any{
		"ticketId": ticketID, "walletId": walletID,
	}, aliceToken)
	booking := decode[map[string]any](t, bw)
	bookingID := int(booking["id"].(float64))

	w := request(t, router, http.MethodDelete, fmt.Sprintf("/api/bookings/%d", bookingID), nil, bobToken)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

// ===== E5：並發測試 =====

// concurrentPost 在 goroutine 內直接發 HTTP request，不透過 request() helper，
// 因為 t.Fatalf 不能從非 test goroutine 呼叫。
func concurrentPost(handler http.Handler, path string, body map[string]any, token string) int {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w.Code
}

// TestConcurrentBooking_SameTicket 驗證 10 條 goroutine 同時搶同一張票，
// 只有一個能成功，其餘回 409。
func TestConcurrentBooking_SameTicket(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	ticketID, walletID, _ := setupBookingFixture(t, router, token, 10000.0)

	const n = 10
	codes := make([]int, n)
	var wg sync.WaitGroup
	ready := make(chan struct{})

	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-ready // 等起跑槍，讓所有 goroutine 盡量同時送出
			codes[idx] = concurrentPost(router, "/api/bookings", map[string]any{
				"ticketId": ticketID,
				"walletId": walletID,
			}, token)
		}(i)
	}

	close(ready) // 同時放行
	wg.Wait()

	successes := 0
	for _, c := range codes {
		if c == http.StatusCreated {
			successes++
		}
	}
	if successes != 1 {
		t.Errorf("expected exactly 1 success out of %d concurrent requests, got %d", n, successes)
	}
}

// TestConcurrentWithdraw_BalanceNeverNegative 驗證 10 條 goroutine 同時提款，
// 餘額不會變成負數。
func TestConcurrentWithdraw_BalanceNeverNegative(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	wallet := createWallet(t, router, token, "Main Wallet")
	walletID := int(wallet["id"].(float64))
	// 存 1000，每次提 200 → 最多 5 次成功
	request(t, router, http.MethodPost, fmt.Sprintf("/api/wallets/%d/deposit", walletID),
		map[string]any{"amount": 1000.0}, token)

	const n = 10
	codes := make([]int, n)
	var wg sync.WaitGroup
	ready := make(chan struct{})
	path := fmt.Sprintf("/api/wallets/%d/withdraw", walletID)

	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-ready
			codes[idx] = concurrentPost(router, path, map[string]any{"amount": 200.0}, token)
		}(i)
	}

	close(ready)
	wg.Wait()

	successes := 0
	for _, c := range codes {
		if c == http.StatusOK {
			successes++
		}
	}
	if successes > 5 {
		t.Errorf("expected at most 5 successful withdrawals (balance=1000, amount=200), got %d", successes)
	}

	// 最終餘額不能是負數
	ww := request(t, router, http.MethodGet, fmt.Sprintf("/api/wallets/%d", walletID), nil, token)
	finalWallet := decode[map[string]any](t, ww)
	if finalWallet["balance"].(float64) < 0 {
		t.Errorf("balance went negative: %v", finalWallet["balance"])
	}
}

func TestCancelBooking_AlreadyCancelled(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	ticketID, walletID, _ := setupBookingFixture(t, router, token, 1000.0)

	bw := request(t, router, http.MethodPost, "/api/bookings", map[string]any{
		"ticketId": ticketID, "walletId": walletID,
	}, token)
	booking := decode[map[string]any](t, bw)
	bookingID := int(booking["id"].(float64))

	// 第一次取消
	request(t, router, http.MethodDelete, fmt.Sprintf("/api/bookings/%d", bookingID), nil, token)

	// 第二次應該 409
	w := request(t, router, http.MethodDelete, fmt.Sprintf("/api/bookings/%d", bookingID), nil, token)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}
