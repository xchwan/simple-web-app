package wallet

import (
	"net/http"

	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin"
	"github.com/xchwan/simple-web-framework/plugin/apidoc"
	"github.com/xchwan/simple-web-framework/scope"
	"github.com/xchwan/simple-web-app/internal/user"
	"gorm.io/gorm"
)

// SetupRoutes 向 router 註冊 wallet 相關的依賴與路由。
func SetupRoutes(router *framework.Router, database *gorm.DB, mapper *plugin.ExceptionMapperPlugin) {
	mapper.
		On(ErrNotFound, http.StatusNotFound, "Wallet not found").
		On(ErrForbidden, http.StatusForbidden, "Forbidden").
		On(ErrNameFormatInvalid, http.StatusBadRequest, "Wallet name format invalid")

	walletDB := NewMySQLWalletRepository(database)
	router.Bind("walletService", func() any {
		return NewWalletService(walletDB)
	}, scope.NewHttpRequestScope())

	h := NewWalletHandler()

	protected := router.Group("/api", user.Auth)
	protected.POST("/wallets", apidoc.Doc[CreateWalletRequest, WalletResponse](h.Create, "Create a wallet"))
	protected.GET("/wallets", apidoc.Doc[apidoc.NoBody, []WalletResponse](h.List, "List my wallets"))
	protected.GET("/wallets/{walletId}", apidoc.Doc[apidoc.NoBody, WalletResponse](h.Get, "Get wallet by ID"))
}
