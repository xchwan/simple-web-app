package user

import "errors"

var (
	// ErrEmailDuplicate 定義在 userdb，此處以 re-export 方式提供給 routes 使用。
	ErrRegisterFormatInvalid = errors.New("register format invalid")
	ErrCredentialsInvalid    = errors.New("credentials invalid")
	ErrLoginFormatInvalid    = errors.New("login format invalid")
	ErrTokenInvalid          = errors.New("token invalid")
	ErrForbidden             = errors.New("forbidden")
	ErrNameFormatInvalid     = errors.New("name format invalid")
)
