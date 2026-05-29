package wallet

import (
	"net/http"
	"strconv"

	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-app/internal/user"
	walletrepo "github.com/xchwan/simple-web-app/internal/wallet/repo"
)

// WalletHandler 負責處理錢包相關的 HTTP 請求。
type WalletHandler struct{}

// NewWalletHandler 建立一個 WalletHandler。
func NewWalletHandler() *WalletHandler {
	return &WalletHandler{}
}

func (h *WalletHandler) service(r *http.Request) *WalletService {
	return framework.Get[*WalletService](r, "walletService")
}

// ===== Request / Response DTO =====

type CreateWalletRequest struct {
	Name string `json:"name"`
}

type WalletResponse struct {
	ID      int     `json:"id"`
	UserID  int     `json:"userId"`
	Name    string  `json:"name"`
	Balance float64 `json:"balance"`
}

// ===== Handlers =====

// Create 處理 POST /api/wallets。
func (h *WalletHandler) Create(w http.ResponseWriter, r *http.Request) {
	caller := user.Caller(r)
	var req CreateWalletRequest
	if err := framework.ParseOrRespond(w, r, &req); err != nil {
		return
	}
	wallet, err := h.service(r).Create(caller.ID, req.Name)
	if err != nil {
		framework.HandleError(w, r, err)
		return
	}
	framework.Respond(w, r, http.StatusCreated, toWalletResponse(wallet))
}

// Get 處理 GET /api/wallets/{walletId}。
func (h *WalletHandler) Get(w http.ResponseWriter, r *http.Request) {
	caller := user.Caller(r)
	id, err := strconv.Atoi(framework.PathParam(r, "walletId"))
	if err != nil {
		framework.HandleError(w, r, ErrNotFound)
		return
	}
	wallet, svcErr := h.service(r).GetByID(caller.ID, id)
	if svcErr != nil {
		framework.HandleError(w, r, svcErr)
		return
	}
	framework.Respond(w, r, http.StatusOK, toWalletResponse(wallet))
}

// List 處理 GET /api/wallets。
func (h *WalletHandler) List(w http.ResponseWriter, r *http.Request) {
	caller := user.Caller(r)
	wallets := h.service(r).ListByUser(caller.ID)
	result := make([]WalletResponse, len(wallets))
	for i, wallet := range wallets {
		result[i] = toWalletResponse(wallet)
	}
	framework.Respond(w, r, http.StatusOK, result)
}

func toWalletResponse(w *walletrepo.Wallet) WalletResponse {
	return WalletResponse{ID: w.ID, UserID: w.UserID, Name: w.Name, Balance: w.Balance}
}
