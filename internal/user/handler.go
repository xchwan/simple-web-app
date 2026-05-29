package user

import (
	"net/http"
	"strconv"

	framework "github.com/xchwan/simple-web-framework"
	userrepo "github.com/xchwan/simple-web-app/internal/user/repo"
)

// UserHandler 負責處理會員相關的 HTTP 請求。
type UserHandler struct{}

// NewUserHandler 建立一個 UserHandler。
func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// service 從 request context 的 container 取得 UserService。
func (h *UserHandler) service(r *http.Request) *UserService {
	return framework.Get[*UserService](r, "userService")
}

// ===== Request / Response DTO =====

type RegisterRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RenameRequest struct {
	NewName string `json:"newName"`
}

type UserResponse struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type LoginResponse struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

// ===== Handlers =====

// Register 處理 POST /api/users。
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := framework.ParseOrRespond(w, r, &req); err != nil {
		return
	}
	u, err := h.service(r).Register(req.Email, req.Name, req.Password)
	if err != nil {
		framework.HandleError(w, r, err)
		return
	}
	framework.Respond(w, r, http.StatusCreated, toUserResponse(u))
}

// Login 處理 POST /api/users/login。
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := framework.ParseOrRespond(w, r, &req); err != nil {
		return
	}
	u, token, err := h.service(r).Login(req.Email, req.Password)
	if err != nil {
		framework.HandleError(w, r, err)
		return
	}
	framework.Respond(w, r, http.StatusOK, LoginResponse{
		ID: u.ID, Email: u.Email, Name: u.Name, Token: token,
	})
}

// Logout 處理 POST /api/users/logout（需 Auth middleware）。
func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r)
	h.service(r).Logout(token)
	framework.Respond(w, r, http.StatusNoContent, nil)
}

// UpdateName 處理 PATCH /api/users/{userId}（需 Auth middleware）。
func (h *UserHandler) UpdateName(w http.ResponseWriter, r *http.Request) {
	caller := Caller(r)

	targetID, err := strconv.Atoi(framework.PathParam(r, "userId"))
	if err != nil {
		framework.HandleError(w, r, ErrNameFormatInvalid)
		return
	}

	var req RenameRequest
	if err := framework.ParseOrRespond(w, r, &req); err != nil {
		return
	}

	if err := h.service(r).UpdateName(caller.ID, targetID, req.NewName); err != nil {
		framework.HandleError(w, r, err)
		return
	}
	framework.Respond(w, r, http.StatusNoContent, nil)
}

// SearchUsers 處理 GET /api/users（需 Auth middleware）。
func (h *UserHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")
	users := h.service(r).SearchUsers(keyword)

	result := make([]UserResponse, len(users))
	for i, u := range users {
		result[i] = toUserResponse(u)
	}
	framework.Respond(w, r, http.StatusOK, result)
}

func toUserResponse(u *userrepo.User) UserResponse {
	return UserResponse{ID: u.ID, Email: u.Email, Name: u.Name}
}
