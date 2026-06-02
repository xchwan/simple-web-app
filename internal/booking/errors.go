package booking

import "errors"

var (
	ErrNotFound            = errors.New("booking not found")
	ErrForbidden           = errors.New("forbidden")
	ErrTicketNotFound      = errors.New("ticket not found")
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrTicketUnavailable   = errors.New("ticket is not available")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrAlreadyCancelled    = errors.New("booking already cancelled")
)
