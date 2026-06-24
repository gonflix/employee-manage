package domain

import "errors"

var (
	ErrNotFound            = errors.New("not found")
	ErrInvalidInput        = errors.New("invalid input")
	ErrAuthentication      = errors.New("authentication failed")
	ErrForbidden           = errors.New("forbidden")
	ErrBlacklistedEmployee = errors.New("blacklisted employee")
	ErrConflict            = errors.New("conflict")
	ErrExternalAPI         = errors.New("external api failed")
)
