package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin"
	"github.com/xchwan/simple-web-app/internal/booking"
	"github.com/xchwan/simple-web-app/internal/event"
	"github.com/xchwan/simple-web-app/internal/infra"
	"github.com/xchwan/simple-web-app/internal/ticket"
	"github.com/xchwan/simple-web-app/internal/user"
	"github.com/xchwan/simple-web-app/internal/wallet"
	"gorm.io/gorm"
)

// ===== 測試環境設定 =====

// testDB / testRDB 在 TestMain 初始化後由所有測試共用，
// 避免每個 test 各自開新連線池而耗盡 MySQL max_connections。
var (
	testDB  *gorm.DB
	testRDB *redis.Client
)

func newRouter() http.Handler {
	mapper := plugin.NewExceptionMapperPlugin()
	router := framework.NewRouter()
	router.AddPlugin(mapper)
	user.SetupRoutes(router, testDB, testRDB, mapper)
	wallet.SetupRoutes(router, testDB, mapper)
	event.SetupRoutes(router, testDB, nil, mapper) // nil = MySQL-only，不需要 ES
	ticket.SetupRoutes(router, testDB, mapper)
	booking.SetupRoutes(router, testDB, mapper)
	return router
}

// TestMain 若未設定 DB_DSN 則跳過所有 integration test，
// 否則先確保 schema 存在，再清空資料，確保每次執行都是乾淨的狀態。
func TestMain(m *testing.M) {
	if os.Getenv("DB_DSN") == "" {
		fmt.Println("⚠️  跳過 integration tests（請先執行 make up，再用 make test）")
		os.Exit(0)
	}
	database, err := infra.Connect()
	if err != nil {
		panic("DB 連線失敗: " + err.Error())
	}
	if err := infra.Migrate(database); err != nil {
		panic("Migration 失敗: " + err.Error())
	}
	testDB = database
	testRDB = infra.ConnectRedis()
	cleanUp()
	os.Exit(m.Run())
}

// cleanUp 清空所有資料表與 Redis session，確保測試隔離。
func cleanUp() {
	testDB.Exec("DELETE FROM bookings")
	testDB.Exec("DELETE FROM tickets")
	testDB.Exec("DELETE FROM events")
	testDB.Exec("DELETE FROM wallets")
	testDB.Exec("DELETE FROM users")
	testRDB.FlushDB(context.Background())
}

// withCleanDB 在測試前後各清空一次 DB，確保每個測試完全隔離。
func withCleanDB(t *testing.T) {
	t.Helper()
	cleanUp()
	t.Cleanup(cleanUp)
}

// ===== 測試輔助函式 =====

func request(t *testing.T, handler http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(w.Body).Decode(&v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return v
}

// registerAndLogin 建立一個新會員並登入，回傳 token 和 userID。
func registerAndLogin(t *testing.T, handler http.Handler, email, name, password string) (token string, userID float64) {
	t.Helper()
	request(t, handler, http.MethodPost, "/api/users", map[string]any{
		"email": email, "name": name, "password": password,
	}, "")
	w := request(t, handler, http.MethodPost, "/api/users/login", map[string]any{
		"email": email, "password": password,
	}, "")
	resp := decode[map[string]any](t, w)
	return resp["token"].(string), resp["id"].(float64)
}

// ===== A1：會員註冊 =====

func TestRegister_Success(t *testing.T) {
	withCleanDB(t)
	w := request(t, newRouter(), http.MethodPost, "/api/users", map[string]any{
		"email": "alice@example.com", "name": "Alice", "password": "pass1234",
	}, "")

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	resp := decode[map[string]any](t, w)
	if resp["email"] != "alice@example.com" {
		t.Errorf("email mismatch: %v", resp["email"])
	}
	if resp["name"] != "Alice" {
		t.Errorf("name mismatch: %v", resp["name"])
	}
	if resp["id"] == nil {
		t.Error("expected id in response")
	}
}

func TestRegister_FormatInvalid(t *testing.T) {
	cases := []struct {
		name  string
		email string
		uname string
		pass  string
	}{
		{"email missing @", "invalidemail", "Alice", "pass1234"},
		{"email too short", "a@b", "Alice", "pass1234"},
		{"name too short", "alice@example.com", "Al", "pass1234"},
		{"name too long", "alice@example.com", "AliceAliceAliceAliceAliceAliceAlic", "pass1234"},
		{"password too short", "alice@example.com", "Alice", "abc"},
		{"password too long", "alice@example.com", "Alice", "passwordpasswordpasswordpasswordp"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := request(t, newRouter(), http.MethodPost, "/api/users", map[string]any{
				"email": tc.email, "name": tc.uname, "password": tc.pass,
			}, "")
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
		})
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	request(t, router, http.MethodPost, "/api/users", map[string]any{
		"email": "alice@example.com", "name": "Alice", "password": "pass1234",
	}, "")
	w := request(t, router, http.MethodPost, "/api/users", map[string]any{
		"email": "alice@example.com", "name": "Alice2", "password": "pass5678",
	}, "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ===== A2：會員登入 =====

func TestLogin_Success(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	request(t, router, http.MethodPost, "/api/users", map[string]any{
		"email": "alice@example.com", "name": "Alice", "password": "pass1234",
	}, "")

	w := request(t, router, http.MethodPost, "/api/users/login", map[string]any{
		"email": "alice@example.com", "password": "pass1234",
	}, "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := decode[map[string]any](t, w)
	if resp["token"] == nil || resp["token"] == "" {
		t.Error("expected non-empty token")
	}
	if resp["email"] != "alice@example.com" {
		t.Errorf("email mismatch: %v", resp["email"])
	}
}

func TestLogin_CredentialsInvalid(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	request(t, router, http.MethodPost, "/api/users", map[string]any{
		"email": "alice@example.com", "name": "Alice", "password": "pass1234",
	}, "")

	w := request(t, router, http.MethodPost, "/api/users/login", map[string]any{
		"email": "alice@example.com", "password": "wrongpassword",
	}, "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLogin_FormatInvalid(t *testing.T) {
	cases := []struct {
		name  string
		email string
		pass  string
	}{
		{"email missing @", "invalidemail", "pass1234"},
		{"password too short", "alice@example.com", "abc"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := request(t, newRouter(), http.MethodPost, "/api/users/login", map[string]any{
				"email": tc.email, "password": tc.pass,
			}, "")
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
		})
	}
}

// ===== A3：修改會員名稱 =====

func TestUpdateName_Success(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, id := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	w := request(t, router, http.MethodPatch, fmt.Sprintf("/api/users/%d", int(id)), map[string]any{
		"newName": "AliceNew",
	}, token)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestUpdateName_Unauthenticated(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	_, id := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	w := request(t, router, http.MethodPatch, fmt.Sprintf("/api/users/%d", int(id)), map[string]any{
		"newName": "AliceNew",
	}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUpdateName_Forbidden(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	_, aliceID := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	bobToken, _ := registerAndLogin(t, router, "bob@example.com", "Bobby", "pass1234")

	w := request(t, router, http.MethodPatch, fmt.Sprintf("/api/users/%d", int(aliceID)), map[string]any{
		"newName": "AliceHacked",
	}, bobToken)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestUpdateName_FormatInvalid(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, id := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	cases := []struct {
		name    string
		newName string
	}{
		{"name too short", "Al"},
		{"name too long", "AliceAliceAliceAliceAliceAliceAlic"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := request(t, router, http.MethodPatch, fmt.Sprintf("/api/users/%d", int(id)), map[string]any{
				"newName": tc.newName,
			}, token)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
		})
	}
}

// ===== A4：查詢會員列表 =====

func TestSearchUsers_AllUsers(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	registerAndLogin(t, router, "bob@example.com", "Bobby", "pass1234")

	w := request(t, router, http.MethodGet, "/api/users", nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestSearchUsers_WithKeyword(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	registerAndLogin(t, router, "bob@example.com", "Bobby", "pass1234")

	w := request(t, router, http.MethodGet, "/api/users?keyword=Ali", nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var users []map[string]any
	if err := json.NewDecoder(w.Body).Decode(&users); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
}

func TestSearchUsers_Unauthenticated(t *testing.T) {
	w := request(t, newRouter(), http.MethodGet, "/api/users", nil, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ===== A5：登出 =====

func TestLogout_Success(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	// 登出
	w := request(t, router, http.MethodPost, "/api/users/logout", nil, token)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	// 登出後 token 應失效
	w = request(t, router, http.MethodGet, "/api/users", nil, token)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d", w.Code)
	}
}
