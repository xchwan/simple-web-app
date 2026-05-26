package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-app/internal/user"
)

// ===== 測試輔助函式 =====

func newRouter() http.Handler {
	router := framework.NewRouter()
	user.Register(router)
	return router
}

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
	router := newRouter()
	_, aliceID := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	bobToken, _ := registerAndLogin(t, router, "bob@example.com", "Bobby", "pass1234")

	// Bob 嘗試修改 Alice 的名稱
	w := request(t, router, http.MethodPatch, fmt.Sprintf("/api/users/%d", int(aliceID)), map[string]any{
		"newName": "AliceHacked",
	}, bobToken)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestUpdateName_FormatInvalid(t *testing.T) {
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
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	registerAndLogin(t, router, "bob@example.com", "Bobby", "pass1234")

	w := request(t, router, http.MethodGet, "/api/users", nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var users []map[string]any
	if err := json.NewDecoder(w.Body).Decode(&users); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestSearchUsers_WithKeyword(t *testing.T) {
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
	if users[0]["name"] != "Alice" {
		t.Errorf("expected Alice, got %v", users[0]["name"])
	}
}

func TestSearchUsers_Unauthenticated(t *testing.T) {
	w := request(t, newRouter(), http.MethodGet, "/api/users", nil, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
