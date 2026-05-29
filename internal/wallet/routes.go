package wallet

import (
	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin/apidoc"
	"github.com/xchwan/simple-web-framework/scope"
	"github.com/xchwan/simple-web-app/internal/user"
	"gorm.io/gorm"
)

// SetupRoutes 向 router 註冊 wallet 相關的依賴與路由。
// 錯誤對應由 main.go 統一在 ExceptionMapperPlugin 設定。
func SetupRoutes(router *framework.Router, database *gorm.DB) {
	walletDB := NewWalletDB(database)
	router.Bind("walletService", func() any {
		return NewWalletService(walletDB)
	}, scope.NewHttpRequestScope())

	h := NewWalletHandler()

	protected := router.Group("/api", user.Auth)
	protected.POST("/wallets", apidoc.Doc[CreateWalletRequest, WalletResponse](h.Create, "Create a wallet"))
	protected.GET("/wallets", apidoc.Doc[apidoc.NoBody, []WalletResponse](h.List, "List my wallets"))
	protected.GET("/wallets/{walletId}", apidoc.Doc[apidoc.NoBody, WalletResponse](h.Get, "Get wallet by ID"))
}
