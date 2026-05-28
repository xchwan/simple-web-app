package test

import (
	"net/http"
	"testing"
)

// ===== E1：列出訂票 =====

func TestListBookings_Success(t *testing.T) {
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
