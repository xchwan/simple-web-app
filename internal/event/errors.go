package event

import "errors"

var (
	ErrNotFound          = errors.New("event not found")
	ErrForbidden         = errors.New("forbidden")
	ErrNameFormatInvalid = errors.New("event name format invalid")
	ErrStartAtInvalid    = errors.New("event start time invalid")
	ErrCapacityInvalid   = errors.New("event capacity must be > 0")
)
