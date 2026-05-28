package booking

import "errors"

var (
	ErrNotFound  = errors.New("booking not found")
	ErrForbidden = errors.New("forbidden")
)
