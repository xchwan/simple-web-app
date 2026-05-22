package user

import "errors"

var (
	ErrEmailDuplicate       = errors.New("email duplicate")
	ErrRegisterFormatInvalid = errors.New("register format invalid")
	ErrCredentialsInvalid   = errors.New("credentials invalid")
	ErrLoginFormatInvalid   = errors.New("login format invalid")
	ErrTokenInvalid         = errors.New("token invalid")
	ErrForbidden            = errors.New("forbidden")
	ErrNameFormatInvalid    = errors.New("name format invalid")
)
