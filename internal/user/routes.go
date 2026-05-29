package user

import (
	"net/http"

	"github.com/redis/go-redis/v9"
	framework "github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin"
	"github.com/xchwan/simple-web-framework/plugin/apidoc"
	"github.com/xchwan/simple-web-framework/scope"
	userrepo "github.com/xchwan/simple-web-app/internal/user/repo"
	"gorm.io/gorm"
)

// SetupRoutes 向 router 註冊 user 相關的依賴與路由。
func SetupRoutes(router *framework.Router, database *gorm.DB, rdb *redis.Client, mapper *plugin.ExceptionMapperPlugin) {
	mapper.
		On(userrepo.ErrEmailDuplicate, http.StatusBadRequest, "Duplicate email").
		On(ErrRegisterFormatInvalid, http.StatusBadRequest, "Registration format invalid").
		On(ErrCredentialsInvalid, http.StatusBadRequest, "Credentials invalid").
		On(ErrLoginFormatInvalid, http.StatusBadRequest, "Login format invalid").
		On(ErrTokenInvalid, http.StatusUnauthorized, "Can't authenticate who you are").
		On(ErrForbidden, http.StatusForbidden, "Forbidden").
		On(ErrNameFormatInvalid, http.StatusBadRequest, "Name format invalid")

	repo := userrepo.NewMySQLUserRepository(database)
	router.Bind("userService", func() any {
		return NewUserService(repo, rdb)
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
