package event

import (
	"net/http"
	"strconv"
	"time"

	framework "github.com/xchwan/simple-web-framework"
	eventrepo "github.com/xchwan/simple-web-app/internal/event/repo"
	"github.com/xchwan/simple-web-app/internal/user"
)

// EventHandler 負責處理活動相關的 HTTP 請求。
type EventHandler struct{}

// NewEventHandler 建立一個 EventHandler。
func NewEventHandler() *EventHandler {
	return &EventHandler{}
}

func (h *EventHandler) service(r *http.Request) *EventService {
	return framework.Get[*EventService](r, "eventService")
}

// ===== Request / Response DTO =====

type CreateEventRequest struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	StartAt     time.Time `json:"startAt"`
}

type UpdateEventRequest struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	StartAt     time.Time `json:"startAt"`
}

type EventResponse struct {
	ID          int       `json:"id"`
	OrganizerID int       `json:"organizerId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	StartAt     time.Time `json:"startAt"`
}

// ===== Handlers =====

// Create 處理 POST /api/events。
func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	caller := user.Caller(r)
	var req CreateEventRequest
	if err := framework.ParseOrRespond(w, r, &req); err != nil {
		return
	}
	e, err := h.service(r).Create(caller.ID, req.Name, req.Description, req.StartAt)
	if err != nil {
		framework.HandleError(w, r, err)
		return
	}
	framework.Respond(w, r, http.StatusCreated, toEventResponse(e))
}

// Get 處理 GET /api/events/{eventId}。
func (h *EventHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(framework.PathParam(r, "eventId"))
	if err != nil {
		framework.HandleError(w, r, ErrNotFound)
		return
	}
	e, svcErr := h.service(r).GetByID(id)
	if svcErr != nil {
		framework.HandleError(w, r, svcErr)
		return
	}
	framework.Respond(w, r, http.StatusOK, toEventResponse(e))
}

// Search 處理 GET /api/events（可選 ?keyword=&startFrom=&startTo=）。
func (h *EventHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := eventrepo.EventQuery{
		Keyword: r.URL.Query().Get("keyword"),
	}
	if s := r.URL.Query().Get("startFrom"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			q.StartFrom = &t
		}
	}
	if s := r.URL.Query().Get("startTo"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			q.StartTo = &t
		}
	}
	events := h.service(r).Search(q)
	result := make([]EventResponse, len(events))
	for i, e := range events {
		result[i] = toEventResponse(e)
	}
	framework.Respond(w, r, http.StatusOK, result)
}

// Update 處理 PATCH /api/events/{eventId}。
func (h *EventHandler) Update(w http.ResponseWriter, r *http.Request) {
	caller := user.Caller(r)
	id, err := strconv.Atoi(framework.PathParam(r, "eventId"))
	if err != nil {
		framework.HandleError(w, r, ErrNotFound)
		return
	}
	var req UpdateEventRequest
	if err := framework.ParseOrRespond(w, r, &req); err != nil {
		return
	}
	e, svcErr := h.service(r).Update(caller.ID, id, req.Name, req.Description, req.StartAt)
	if svcErr != nil {
		framework.HandleError(w, r, svcErr)
		return
	}
	framework.Respond(w, r, http.StatusOK, toEventResponse(e))
}

// Delete 處理 DELETE /api/events/{eventId}。
func (h *EventHandler) Delete(w http.ResponseWriter, r *http.Request) {
	caller := user.Caller(r)
	id, err := strconv.Atoi(framework.PathParam(r, "eventId"))
	if err != nil {
		framework.HandleError(w, r, ErrNotFound)
		return
	}
	if svcErr := h.service(r).Delete(caller.ID, id); svcErr != nil {
		framework.HandleError(w, r, svcErr)
		return
	}
	framework.Respond(w, r, http.StatusNoContent, nil)
}

func toEventResponse(e *eventrepo.Event) EventResponse {
	return EventResponse{
		ID:          e.ID,
		OrganizerID: e.OrganizerID,
		Name:        e.Name,
		Description: e.Description,
		StartAt:     e.StartAt,
	}
}
