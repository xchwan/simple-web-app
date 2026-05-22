package user

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/xchwan/simple-web-framework"
)

// UserHandler 負責處理會員相關的 HTTP 請求。
// service 在每個 request 時從 container 動態取得，以支援 HttpRequestScope。
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

type registerRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type renameRequest struct {
	NewName string `json:"newName"`
}

type userResponse struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type loginResponse struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

// ===== Handlers =====

// Register 處理 POST /api/users。
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	framework.ParseRequest(r, &req)
	u, err := h.service(r).Register(req.Email, req.Name, req.Password)
	if err != nil {
		framework.HandleError(w, r, err)
		return
	}
	framework.Respond(w, r, http.StatusCreated, userResponse{
		ID:    u.ID,
		Email: u.Email,
		Name:  u.Name,
	})
}

// Login 處理 POST /api/users/login。
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := framework.ParseOrRespond(w, r, &req); err != nil {
		return
	}
	u, err := h.service(r).Login(req.Email, req.Password)
	if err != nil {
		framework.HandleError(w, r, err)
		return
	}
	framework.Respond(w, r, http.StatusOK, loginResponse{
		ID:    u.ID,
		Email: u.Email,
		Name:  u.Name,
		Token: u.Token,
	})
}

// UpdateName 處理 PATCH /api/users/{userId}。
func (h *UserHandler) UpdateName(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	caller, err := h.service(r).Authenticate(token)
	if err != nil {
		framework.HandleError(w, r, err)
		return
	}

	targetID, err := strconv.Atoi(framework.PathParam(r, "userId"))
	if err != nil {
		framework.HandleError(w, r, ErrNameFormatInvalid)
		return
	}

	var req renameRequest
	if err := framework.ParseOrRespond(w, r, &req); err != nil {
		return
	}

	if err := h.service(r).UpdateName(caller.ID, targetID, req.NewName); err != nil {
		framework.HandleError(w, r, err)
		return
	}
	framework.Respond(w, r, http.StatusNoContent, nil)
}

// SearchUsers 處理 GET /api/users。
func (h *UserHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if _, err := h.service(r).Authenticate(token); err != nil {
		framework.HandleError(w, r, err)
		return
	}

	keyword := r.URL.Query().Get("keyword")
	users := h.service(r).SearchUsers(keyword)

	result := make([]userResponse, len(users))
	for i, u := range users {
		result[i] = userResponse{ID: u.ID, Email: u.Email, Name: u.Name}
	}
	framework.Respond(w, r, http.StatusOK, result)
}

// ===== 私有輔助函式 =====

// extractToken 從 Authorization: Bearer <token> 標頭取出 token。
func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}
