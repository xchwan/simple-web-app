package test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// ===== 輔助函式 =====

func createTickets(t *testing.T, handler http.Handler, token string, eventID int, tickets []map[string]any) []map[string]any {
	t.Helper()
	w := request(t, handler, http.MethodPost, fmt.Sprintf("/api/events/%d/tickets", eventID), map[string]any{
		"tickets": tickets,
	}, token)
	return decode[[]map[string]any](t, w)
}

// ===== D1：批次建立票券 =====

func TestBatchCreateTickets_Success(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	event := createEvent(t, router, token)
	eventID := int(event["id"].(float64))

	w := request(t, router, http.MethodPost, fmt.Sprintf("/api/events/%d/tickets", eventID), map[string]any{
		"tickets": []map[string]any{
			{"seat": "A1", "price": 500},
			{"seat": "A2", "price": 500},
			{"seat": "B1", "price": 300},
		},
	}, token)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	tickets := decode[[]map[string]any](t, w)
	if len(tickets) != 3 {
		t.Errorf("expected 3 tickets, got %d", len(tickets))
	}
	for _, ticket := range tickets {
		if ticket["status"] != "available" {
			t.Errorf("expected status available, got %v", ticket["status"])
		}
	}
}

func TestBatchCreateTickets_Unauthenticated(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	event := createEvent(t, router, token)
	eventID := int(event["id"].(float64))

	w := request(t, router, http.MethodPost, fmt.Sprintf("/api/events/%d/tickets", eventID), map[string]any{
		"tickets": []map[string]any{{"seat": "A1", "price": 500}},
	}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestBatchCreateTickets_Forbidden(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	aliceToken, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	bobToken, _ := registerAndLogin(t, router, "bob@example.com", "Bobby", "pass1234")
	event := createEvent(t, router, aliceToken)
	eventID := int(event["id"].(float64))

	w := request(t, router, http.MethodPost, fmt.Sprintf("/api/events/%d/tickets", eventID), map[string]any{
		"tickets": []map[string]any{{"seat": "A1", "price": 500}},
	}, bobToken)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestBatchCreateTickets_EventNotFound(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	w := request(t, router, http.MethodPost, "/api/events/99999/tickets", map[string]any{
		"tickets": []map[string]any{{"seat": "A1", "price": 500}},
	}, token)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestBatchCreateTickets_FormatInvalid(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	event := createEvent(t, router, token)
	eventID := int(event["id"].(float64))

	cases := []struct {
		name   string
		ticket map[string]any
	}{
		{"seat empty", map[string]any{"seat": "", "price": 100}},
		{"seat too long", map[string]any{"seat": strings.Repeat("X", 51), "price": 100}},
		{"negative price", map[string]any{"seat": "A1", "price": -1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := request(t, router, http.MethodPost, fmt.Sprintf("/api/events/%d/tickets", eventID), map[string]any{
				"tickets": []map[string]any{tc.ticket},
			}, token)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
		})
	}
}

// ===== D2：列出票券 =====

func TestListTickets_ByEvent(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	event := createEvent(t, router, token)
	eventID := int(event["id"].(float64))
	createTickets(t, router, token, eventID, []map[string]any{
		{"seat": "A1", "price": 500},
		{"seat": "A2", "price": 500},
	})

	w := request(t, router, http.MethodGet, fmt.Sprintf("/api/events/%d/tickets", eventID), nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	tickets := decode[[]map[string]any](t, w)
	if len(tickets) != 2 {
		t.Errorf("expected 2 tickets, got %d", len(tickets))
	}
}

func TestListTickets_FilterByStatus(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	event := createEvent(t, router, token)
	eventID := int(event["id"].(float64))
	createTickets(t, router, token, eventID, []map[string]any{
		{"seat": "A1", "price": 500},
		{"seat": "A2", "price": 500},
	})

	w := request(t, router, http.MethodGet, fmt.Sprintf("/api/events/%d/tickets?status=available", eventID), nil, token)

	tickets := decode[[]map[string]any](t, w)
	if len(tickets) != 2 {
		t.Errorf("expected 2 available tickets, got %d", len(tickets))
	}

	w = request(t, router, http.MethodGet, fmt.Sprintf("/api/events/%d/tickets?status=sold", eventID), nil, token)

	tickets = decode[[]map[string]any](t, w)
	if len(tickets) != 0 {
		t.Errorf("expected 0 sold tickets, got %d", len(tickets))
	}
}

// ===== D3：取得票券 =====

func TestGetTicket_Success(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")
	event := createEvent(t, router, token)
	eventID := int(event["id"].(float64))
	tickets := createTickets(t, router, token, eventID, []map[string]any{
		{"seat": "A1", "price": 500},
	})
	ticketID := int(tickets[0]["id"].(float64))

	w := request(t, router, http.MethodGet, fmt.Sprintf("/api/tickets/%d", ticketID), nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := decode[map[string]any](t, w)
	if resp["seat"] != "A1" {
		t.Errorf("seat mismatch: %v", resp["seat"])
	}
}

func TestGetTicket_NotFound(t *testing.T) {
	withCleanDB(t)
	router := newRouter()
	token, _ := registerAndLogin(t, router, "alice@example.com", "Alice", "pass1234")

	w := request(t, router, http.MethodGet, "/api/tickets/99999", nil, token)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
