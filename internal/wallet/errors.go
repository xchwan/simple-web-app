package wallet

import "errors"

var (
	ErrNotFound          = errors.New("wallet not found")
	ErrForbidden         = errors.New("forbidden")
	ErrNameFormatInvalid = errors.New("wallet name format invalid")
)
