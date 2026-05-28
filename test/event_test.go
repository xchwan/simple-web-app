package test

import (
	"fmt"
	"net/http"
	"testing"
)

// ===== 輔助函式 =====

func createEvent(t *testing.T, handler http.Handler, token string) map[string]any {
	t.Helper()
	w := request(t, handler, http.MethodPost, "/api/events", map[string]any{
		"name":        "Test Concert",
		"description": "A great show",
		"startAt":     "2026-09-01T20:00:00Z",
	}, token)
	return decode[map[string]any](t, w)
}

// ===== C1：建立活動 =====

func TestCreateEvent_Success(t *testing.T) {
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	w := request(t, router, http.MethodPost, "/api/events", map[string]any{
		"name":        "Rock Concert",
		"description": "Best night ever",
		"startAt":     "2026-09-01T20:00:00Z",
	}, token)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	resp := decode[map[string]any](t, w)
	if resp["name"] != "Rock Concert" {
		t.Errorf("name mismatch: %v", resp["name"])
	}
	if resp["organizerId"] == nil {
		t.Error("expected organizerId in response")
	}
}

func TestCreateEvent_Unauthenticated(t *testing.T) {
	w := request(t, newRouter(), http.MethodPost, "/api/events", map[string]any{
		"name":    "Concert",
		"startAt": "2026-09-01T20:00:00Z",
	}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCreateEvent_FormatInvalid(t *testing.T) {
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	cases := []struct {
		name string
		body map[string]any
	}{
		{"name empty", map[string]any{"name": "", "startAt": "2026-09-01T20:00:00Z"}},
		{"startAt missing", map[string]any{"name": "Concert"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := request(t, router, http.MethodPost, "/api/events", tc.body, token)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
		})
	}
}

// ===== C2：取得 / 搜尋活動 =====

func TestGetEvent_Success(t *testing.T) {
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	event := createEvent(t, router, token)
	id := int(event["id"].(float64))

	w := request(t, router, http.MethodGet, fmt.Sprintf("/api/events/%d", id), nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetEvent_NotFound(t *testing.T) {
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	w := request(t, router, http.MethodGet, "/api/events/99999", nil, token)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSearchEvents_AllEvents(t *testing.T) {
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	createEvent(t, router, token)
	createEvent(t, router, token)

	w := request(t, router, http.MethodGet, "/api/events", nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	events := decode[[]map[string]any](t, w)
	if len(events) < 2 {
		t.Errorf("expected at least 2 events, got %d", len(events))
	}
}

func TestSearchEvents_WithKeyword(t *testing.T) {
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	// 使用不會與其他測試撞名的 unique 關鍵字
	request(t, router, http.MethodPost, "/api/events", map[string]any{
		"name": "XyloFest2099", "startAt": "2026-09-01T20:00:00Z",
	}, token)
	request(t, router, http.MethodPost, "/api/events", map[string]any{
		"name": "Jazz Night", "startAt": "2026-10-01T20:00:00Z",
	}, token)

	w := request(t, router, http.MethodGet, "/api/events?keyword=XyloFest2099", nil, token)

	events := decode[[]map[string]any](t, w)
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
	if events[0]["name"] != "XyloFest2099" {
		t.Errorf("unexpected event: %v", events[0]["name"])
	}
}

// ===== C3：更新活動 =====

func TestUpdateEvent_Success(t *testing.T) {
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	event := createEvent(t, router, token)
	id := int(event["id"].(float64))

	w := request(t, router, http.MethodPatch, fmt.Sprintf("/api/events/%d", id), map[string]any{
		"name": "Updated Concert",
	}, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := decode[map[string]any](t, w)
	if resp["name"] != "Updated Concert" {
		t.Errorf("name not updated: %v", resp["name"])
	}
}

func TestUpdateEvent_Unauthenticated(t *testing.T) {
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	event := createEvent(t, router, token)
	id := int(event["id"].(float64))

	w := request(t, router, http.MethodPatch, fmt.Sprintf("/api/events/%d", id), map[string]any{
		"name": "Hacked",
	}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUpdateEvent_Forbidden(t *testing.T) {
	router := newRouter()
	aliceToken, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	bobToken, _ := registerAndLogin(t, router, "bob@example.com", "Bobby", "pass1234")
	event := createEvent(t, router, aliceToken)
	id := int(event["id"].(float64))

	w := request(t, router, http.MethodPatch, fmt.Sprintf("/api/events/%d", id), map[string]any{
		"name": "Hacked by Bob",
	}, bobToken)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

// ===== C4：刪除活動 =====

func TestDeleteEvent_Success(t *testing.T) {
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	event := createEvent(t, router, token)
	id := int(event["id"].(float64))

	w := request(t, router, http.MethodDelete, fmt.Sprintf("/api/events/%d", id), nil, token)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	// 確認已刪除
	w = request(t, router, http.MethodGet, fmt.Sprintf("/api/events/%d", id), nil, token)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", w.Code)
	}
}

func TestDeleteEvent_Unauthenticated(t *testing.T) {
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	event := createEvent(t, router, token)
	id := int(event["id"].(float64))

	w := request(t, router, http.MethodDelete, fmt.Sprintf("/api/events/%d", id), nil, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestDeleteEvent_Forbidden(t *testing.T) {
	router := newRouter()
	aliceToken, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	bobToken, _ := registerAndLogin(t, router, "bob@example.com", "Bobby", "pass1234")
	event := createEvent(t, router, aliceToken)
	id := int(event["id"].(float64))

	w := request(t, router, http.MethodDelete, fmt.Sprintf("/api/events/%d", id), nil, bobToken)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}
