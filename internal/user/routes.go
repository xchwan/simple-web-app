package user

import (
	"net/http"

	"github.com/redis/go-redis/v9"
	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin"
	"github.com/xchwan/simple-web-framework/plugin/apidoc"
	"github.com/xchwan/simple-web-framework/scope"
	"gorm.io/gorm"
)

// SetupRoutes 向 router 註冊 user 相關的依賴、例外規則與路由。
func SetupRoutes(router *framework.Router, docs *apidoc.DocPlugin) {
	router.AddPlugin(plugin.NewExceptionMapperPlugin().
		On(ErrEmailDuplicate, http.StatusBadRequest, "Duplicate email").
		On(ErrRegisterFormatInvalid, http.StatusBadRequest, "Registration's format incorrect.").
		On(ErrCredentialsInvalid, http.StatusBadRequest, "Credentials Invalid").
		On(ErrLoginFormatInvalid, http.StatusBadRequest, "Login's format incorrect.").
		On(ErrTokenInvalid, http.StatusUnauthorized, "Can't authenticate who you are.").
		On(ErrForbidden, http.StatusForbidden, "Forbidden").
		On(ErrNameFormatInvalid, http.StatusBadRequest, "Name's format invalid."),
	)

	router.Bind("userRepo", func() any {
		db := router.Resolve("db").(*gorm.DB)
		return NewUserDB(db)
	})
	router.Bind("userService", func() any {
		repo := router.Resolve("userRepo").(*UserDB)
		rdb := router.Resolve("redis").(*redis.Client)
		return NewUserService(repo, rdb)
	}, scope.NewHttpRequestScope())
	router.Bind("userHandler", func() any {
		return NewUserHandler()
	})

	h := router.Resolve("userHandler").(*UserHandler)

	// 公開路由（不需登入）
	router.POST("/api/users", apidoc.Doc[RegisterRequest, UserResponse](docs, h.Register, "Register a new user"))
	router.POST("/api/users/login", apidoc.Doc[LoginRequest, LoginResponse](docs, h.Login, "Login"))

	// 需要登入的路由
	protected := router.Group("/api", Auth)
	protected.POST("/users/logout", apidoc.Doc[apidoc.NoBody, apidoc.NoBody](docs, h.Logout, "Logout"))
	protected.PATCH("/users/{userId}", apidoc.Doc[RenameRequest, apidoc.NoBody](docs, h.UpdateName, "Update user name"))
	protected.GET("/users", apidoc.Doc[apidoc.NoBody, []UserResponse](docs, h.SearchUsers, "Search users"))
}
