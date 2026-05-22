package user

import (
	"net/http"

	"github.com/xchwan/simple-web-framework"
	"github.com/xchwan/simple-web-framework/plugin"
	"github.com/xchwan/simple-web-framework/scope"
)

// Register 向 router 註冊 user 相關的依賴、例外規則與路由。
func Register(router *framework.Router) {
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
		return NewUserRepository()
	})
	router.Bind("userService", func() any {
		repo := router.Resolve("userRepo").(*UserRepository)
		return NewUserService(repo)
	}, scope.NewHttpRequestScope())
	router.Bind("userHandler", func() any {
		return NewUserHandler()
	})

	h := router.Resolve("userHandler").(*UserHandler)
	router.POST("/api/users", h.Register)
	router.POST("/api/users/login", h.Login)
	router.PATCH("/api/users/{userId}", h.UpdateName)
	router.GET("/api/users", h.SearchUsers)
}
