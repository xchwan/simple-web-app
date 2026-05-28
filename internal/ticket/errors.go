package ticket

import "errors"

var (
	ErrNotFound          = errors.New("ticket not found")
	ErrEventNotFound     = errors.New("event not found")
	ErrForbidden         = errors.New("forbidden")
	ErrSeatFormatInvalid = errors.New("seat format invalid")
	ErrPriceInvalid      = errors.New("price must be >= 0")
)
