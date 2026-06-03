package test

import (
	"fmt"
	"net/http"
	"testing"

	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin"
	"github.com/xchwan/simple-web-app/internal/booking"
	"github.com/xchwan/simple-web-app/internal/event"
	"github.com/xchwan/simple-web-app/internal/ticket"
	"github.com/xchwan/simple-web-app/internal/user"
	"github.com/xchwan/simple-web-app/internal/wallet"
)

// newRouterWithES 建立包含真實 ES client 的 router（ES 整合測試專用）。
// cleanUp() 在每個 test 前後刪除 events-test index；
// SetupRoutes 呼叫 NewElasticSearchRepository → ensureIndex() 重建索引。
func newRouterWithES() http.Handler {
	mapper := plugin.NewExceptionMapperPlugin()
	router := framework.NewRouter()
	router.AddPlugin(mapper)
	user.SetupRoutes(router, testDB, testRDB, mapper)
	wallet.SetupRoutes(router, testDB, mapper)
	event.SetupRoutes(router, testDB, testESClient, mapper) // 真實 ES client
	ticket.SetupRoutes(router, testDB, mapper)
	booking.SetupRoutes(router, testDB, testRDB, nil, nil, mapper)
	return router
}

// skipIfNoES 若 ELASTIC_ADDR 未設定則跳過測試。
func skipIfNoES(t *testing.T) {
	t.Helper()
	if testESClient == nil {
		t.Skip("ES 未設定，跳過（需傳入 ELASTIC_ADDR 環境變數）")
	}
}

// ===== ES 整合測試 =====
// 所有操作都使用 WithRefresh("true")，不需要等待 ES 非同步刷新。

// TestES_SearchByKeyword 驗證 keyword 全文搜尋透過 ES 回傳正確結果。
// 同名的 TestSearchEvents_WithKeyword 走 MySQL LIKE；這裡驗證 ES multi_match。
func TestES_SearchByKeyword(t *testing.T) {
	skipIfNoES(t)
	withCleanDB(t)
	router := newRouterWithES()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	// 建立兩個不同名稱的活動（POST /events 會同時寫入 MySQL 和 ES）
	request(t, router, http.MethodPost, "/api/events", map[string]any{
		"name": "Rock Concert", "startAt": "2026-09-01T20:00:00Z", "capacity": 50,
	}, token)
	request(t, router, http.MethodPost, "/api/events", map[string]any{
		"name": "Jazz Night", "startAt": "2026-10-01T20:00:00Z", "capacity": 50,
	}, token)

	w := request(t, router, http.MethodGet, "/api/events?keyword=Rock", nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	events := decode[[]map[string]any](t, w)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0]["name"] != "Rock Concert" {
		t.Errorf("unexpected event name: %v", events[0]["name"])
	}
}

// TestES_SearchAllEvents 驗證無 keyword 時 ES 回傳所有活動。
func TestES_SearchAllEvents(t *testing.T) {
	skipIfNoES(t)
	withCleanDB(t)
	router := newRouterWithES()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	request(t, router, http.MethodPost, "/api/events", map[string]any{
		"name": "Event A", "startAt": "2026-09-01T20:00:00Z", "capacity": 50,
	}, token)
	request(t, router, http.MethodPost, "/api/events", map[string]any{
		"name": "Event B", "startAt": "2026-10-01T20:00:00Z", "capacity": 50,
	}, token)
	request(t, router, http.MethodPost, "/api/events", map[string]any{
		"name": "Event C", "startAt": "2026-11-01T20:00:00Z", "capacity": 50,
	}, token)

	w := request(t, router, http.MethodGet, "/api/events", nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	events := decode[[]map[string]any](t, w)
	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}
}

// TestES_UpdateEvent_ReIndexed 驗證更新活動名稱後，ES 搜尋能找到新名稱。
func TestES_UpdateEvent_ReIndexed(t *testing.T) {
	skipIfNoES(t)
	withCleanDB(t)
	router := newRouterWithES()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	ev := createEvent(t, router, token) // name = "Test Concert"
	id := int(ev["id"].(float64))

	// 更新名稱
	w := request(t, router, http.MethodPatch, fmt.Sprintf("/api/events/%d", id), map[string]any{
		"name": "Updated Festival",
	}, token)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on update, got %d", w.Code)
	}

	// 用新名稱搜尋，ES 應已更新
	w = request(t, router, http.MethodGet, "/api/events?keyword=Festival", nil, token)
	events := decode[[]map[string]any](t, w)
	if len(events) != 1 {
		t.Fatalf("expected 1 event after update, got %d", len(events))
	}
	if events[0]["name"] != "Updated Festival" {
		t.Errorf("unexpected name: %v", events[0]["name"])
	}

	// 舊名稱不應再搜到
	w = request(t, router, http.MethodGet, "/api/events?keyword=Concert", nil, token)
	old := decode[[]map[string]any](t, w)
	if len(old) != 0 {
		t.Errorf("old name should not be searchable, got %d results", len(old))
	}
}

// TestES_DeleteEvent_Removed 驗證刪除活動後，ES 搜尋不再回傳該活動。
func TestES_DeleteEvent_Removed(t *testing.T) {
	skipIfNoES(t)
	withCleanDB(t)
	router := newRouterWithES()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	ev := createEvent(t, router, token) // name = "Test Concert"
	id := int(ev["id"].(float64))

	// 刪除活動
	w := request(t, router, http.MethodDelete, fmt.Sprintf("/api/events/%d", id), nil, token)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on delete, got %d", w.Code)
	}

	// ES 應已移除
	w = request(t, router, http.MethodGet, "/api/events?keyword=Concert", nil, token)
	events := decode[[]map[string]any](t, w)
	if len(events) != 0 {
		t.Errorf("expected 0 events after delete, got %d", len(events))
	}
}

// TestES_SearchByDescription 驗證 ES multi_match 能搜尋 description 欄位。
func TestES_SearchByDescription(t *testing.T) {
	skipIfNoES(t)
	withCleanDB(t)
	router := newRouterWithES()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	request(t, router, http.MethodPost, "/api/events", map[string]any{
		"name": "Summer Event", "description": "outdoor festival with live music",
		"startAt": "2026-09-01T20:00:00Z", "capacity": 50,
	}, token)
	request(t, router, http.MethodPost, "/api/events", map[string]any{
		"name": "Winter Gala", "description": "indoor formal dinner",
		"startAt": "2026-12-01T20:00:00Z", "capacity": 50,
	}, token)

	// 用 description 關鍵字搜尋
	w := request(t, router, http.MethodGet, "/api/events?keyword=outdoor", nil, token)

	events := decode[[]map[string]any](t, w)
	if len(events) != 1 {
		t.Fatalf("expected 1 event by description, got %d", len(events))
	}
	if events[0]["name"] != "Summer Event" {
		t.Errorf("unexpected event: %v", events[0]["name"])
	}
}
