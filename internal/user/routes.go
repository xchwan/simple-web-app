package user

import (
	"github.com/redis/go-redis/v9"
	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin/apidoc"
	"github.com/xchwan/simple-web-framework/scope"
	"gorm.io/gorm"
)

// SetupRoutes 向 router 註冊 user 相關的依賴與路由。
// 錯誤對應由 main.go 統一在 ExceptionMapperPlugin 設定。
func SetupRoutes(router *framework.Router, database *gorm.DB, rdb *redis.Client) {
	userDB := NewUserDB(database)
	router.Bind("userService", func() any {
		return NewUserService(userDB, rdb)
	}, scope.NewHttpRequestScope())

	h := NewUserHandler()

	// 公開路由（不需登入）
	router.POST("/api/users", apidoc.Doc[RegisterRequest, UserResponse](h.Register, "Register a new user"))
	router.POST("/api/users/login", apidoc.Doc[LoginRequest, LoginResponse](h.Login, "Login"))

	// 需要登入的路由
	protected := router.Group("/api", Auth)
	protected.POST("/users/logout", apidoc.Doc[apidoc.NoBody, apidoc.NoBody](h.Logout, "Logout"))
	protected.PATCH("/users/{userId}", apidoc.Doc[RenameRequest, apidoc.NoBody](h.UpdateName, "Update user name"))
	protected.GET("/users", apidoc.Doc[apidoc.NoBody, []UserResponse](h.SearchUsers, "Search users"))
}
