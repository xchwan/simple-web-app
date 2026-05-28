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
