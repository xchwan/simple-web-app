package test

import (
	"fmt"
	"net/http"
	"testing"
)

// ===== 輔助函式 =====

func createWallet(t *testing.T, handler http.Handler, token, name string) map[string]any {
	t.Helper()
	w := request(t, handler, http.MethodPost, "/api/wallets", map[string]any{
		"name": name,
	}, token)
	return decode[map[string]any](t, w)
}

// ===== B1：建立錢包 =====

func TestCreateWallet_Success(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	w := request(t, router, http.MethodPost, "/api/wallets", map[string]any{
		"name": "Main Wallet",
	}, token)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	resp := decode[map[string]any](t, w)
	if resp["name"] != "Main Wallet" {
		t.Errorf("name mismatch: %v", resp["name"])
	}
	if resp["balance"] != float64(0) {
		t.Errorf("expected balance 0, got %v", resp["balance"])
	}
	if resp["userId"] == nil {
		t.Error("expected userId in response")
	}
}

func TestCreateWallet_Unauthenticated(t *testing.T) {
	w := request(t, newRouter(), http.MethodPost, "/api/wallets", map[string]any{
		"name": "Main Wallet",
	}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCreateWallet_NameFormatInvalid(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	cases := []struct {
		name  string
		wName string
	}{
		{"name empty", ""},
		{"name too long", fmt.Sprintf("%051d", 0)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := request(t, router, http.MethodPost, "/api/wallets", map[string]any{
				"name": tc.wName,
			}, token)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
		})
	}
}

// ===== B2：列出錢包 =====

func TestListWallets_Success(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	createWallet(t, router, token, "Wallet A")
	createWallet(t, router, token, "Wallet B")

	w := request(t, router, http.MethodGet, "/api/wallets", nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	wallets := decode[[]map[string]any](t, w)
	if len(wallets) != 2 {
		t.Errorf("expected 2 wallets, got %d", len(wallets))
	}
}

func TestListWallets_OnlyOwnWallets(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	aliceToken, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	bobToken, _ := registerAndLogin(t, router, "bob@example.com", "Bobby", "pass1234")
	createWallet(t, router, aliceToken, "Alice Wallet")
	createWallet(t, router, bobToken, "Bob Wallet")

	w := request(t, router, http.MethodGet, "/api/wallets", nil, aliceToken)

	wallets := decode[[]map[string]any](t, w)
	if len(wallets) != 1 {
		t.Errorf("expected 1 wallet for Alice, got %d", len(wallets))
	}
	if wallets[0]["name"] != "Alice Wallet" {
		t.Errorf("expected Alice Wallet, got %v", wallets[0]["name"])
	}
}

// ===== B3：取得錢包 =====

func TestGetWallet_Success(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	wallet := createWallet(t, router, token, "Main Wallet")
	id := int(wallet["id"].(float64))

	w := request(t, router, http.MethodGet, fmt.Sprintf("/api/wallets/%d", id), nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetWallet_Forbidden(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	aliceToken, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	bobToken, _ := registerAndLogin(t, router, "bob@example.com", "Bobby", "pass1234")
	wallet := createWallet(t, router, aliceToken, "Alice Wallet")
	id := int(wallet["id"].(float64))

	w := request(t, router, http.MethodGet, fmt.Sprintf("/api/wallets/%d", id), nil, bobToken)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestGetWallet_NotFound(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	w := request(t, router, http.MethodGet, "/api/wallets/99999", nil, token)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// ===== B4：存款 =====

func TestDeposit_Success(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	wallet := createWallet(t, router, token, "Main Wallet")
	id := int(wallet["id"].(float64))

	w := request(t, router, http.MethodPost, fmt.Sprintf("/api/wallets/%d/deposit", id), map[string]any{
		"amount": 500.0,
	}, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := decode[map[string]any](t, w)
	if resp["balance"] != float64(500) {
		t.Errorf("expected balance 500, got %v", resp["balance"])
	}
}

func TestDeposit_Unauthenticated(t *testing.T) {
	w := request(t, newRouter(), http.MethodPost, "/api/wallets/1/deposit", map[string]any{"amount": 100.0}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestDeposit_Forbidden(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	aliceToken, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	bobToken, _ := registerAndLogin(t, router, "bob@example.com", "Bobby", "pass1234")
	wallet := createWallet(t, router, aliceToken, "Alice Wallet")
	id := int(wallet["id"].(float64))

	w := request(t, router, http.MethodPost, fmt.Sprintf("/api/wallets/%d/deposit", id), map[string]any{
		"amount": 100.0,
	}, bobToken)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestDeposit_NotFound(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	w := request(t, router, http.MethodPost, "/api/wallets/99999/deposit", map[string]any{"amount": 100.0}, token)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeposit_AmountInvalid(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	wallet := createWallet(t, router, token, "Main Wallet")
	id := int(wallet["id"].(float64))

	cases := []struct {
		name   string
		amount float64
	}{
		{"zero", 0},
		{"negative", -100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := request(t, router, http.MethodPost, fmt.Sprintf("/api/wallets/%d/deposit", id), map[string]any{
				"amount": tc.amount,
			}, token)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
		})
	}
}

// ===== B5：提款 =====

func TestWithdraw_Success(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	wallet := createWallet(t, router, token, "Main Wallet")
	id := int(wallet["id"].(float64))

	// 先存款
	request(t, router, http.MethodPost, fmt.Sprintf("/api/wallets/%d/deposit", id), map[string]any{"amount": 1000.0}, token)

	w := request(t, router, http.MethodPost, fmt.Sprintf("/api/wallets/%d/withdraw", id), map[string]any{
		"amount": 300.0,
	}, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := decode[map[string]any](t, w)
	if resp["balance"] != float64(700) {
		t.Errorf("expected balance 700, got %v", resp["balance"])
	}
}

func TestWithdraw_Unauthenticated(t *testing.T) {
	w := request(t, newRouter(), http.MethodPost, "/api/wallets/1/withdraw", map[string]any{"amount": 100.0}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestWithdraw_Forbidden(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	aliceToken, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	bobToken, _ := registerAndLogin(t, router, "bob@example.com", "Bobby", "pass1234")
	wallet := createWallet(t, router, aliceToken, "Alice Wallet")
	id := int(wallet["id"].(float64))
	request(t, router, http.MethodPost, fmt.Sprintf("/api/wallets/%d/deposit", id), map[string]any{"amount": 500.0}, aliceToken)

	w := request(t, router, http.MethodPost, fmt.Sprintf("/api/wallets/%d/withdraw", id), map[string]any{
		"amount": 100.0,
	}, bobToken)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestWithdraw_NotFound(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	w := request(t, router, http.MethodPost, "/api/wallets/99999/withdraw", map[string]any{"amount": 100.0}, token)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestWithdraw_AmountInvalid(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	wallet := createWallet(t, router, token, "Main Wallet")
	id := int(wallet["id"].(float64))

	cases := []struct {
		name   string
		amount float64
	}{
		{"zero", 0},
		{"negative", -50},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := request(t, router, http.MethodPost, fmt.Sprintf("/api/wallets/%d/withdraw", id), map[string]any{
				"amount": tc.amount,
			}, token)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
		})
	}
}

func TestWithdraw_InsufficientBalance(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	wallet := createWallet(t, router, token, "Main Wallet")
	id := int(wallet["id"].(float64))
	request(t, router, http.MethodPost, fmt.Sprintf("/api/wallets/%d/deposit", id), map[string]any{"amount": 100.0}, token)

	w := request(t, router, http.MethodPost, fmt.Sprintf("/api/wallets/%d/withdraw", id), map[string]any{
		"amount": 200.0,
	}, token)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
}
