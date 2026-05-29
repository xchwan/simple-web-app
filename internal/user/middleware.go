package user

import (
	"context"
	"net/http"
	"strings"

	framework "github.com/xchwan/simple-web-framework"
	userrepo "github.com/xchwan/simple-web-app/internal/user/repo"
)

type callerKey struct{}

// Auth 是驗證 Bearer token 的 middleware。
// 驗證成功後將已登入的 *userrepo.User 注入 request context，供後續 handler 使用。
func Auth(next framework.HandlerFunc) framework.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		svc := framework.Get[*UserService](r, "userService")
		caller, err := svc.Authenticate(token)
		if err != nil {
			framework.HandleError(w, r, err)
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), callerKey{}, caller))
		next(w, r)
	}
}

// Caller 從 request context 取出已驗證的使用者。
func Caller(r *http.Request) *userrepo.User {
	u, _ := r.Context().Value(callerKey{}).(*userrepo.User)
	return u
}

// extractBearerToken 從 Authorization: Bearer <token> 標頭取出 token。
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}
