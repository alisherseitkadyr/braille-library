package domain

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrForbidden          = errors.New("forbidden")
)

// ValidationError is returned when user input is invalid (HTTP 400).
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

func Invalid(msg string) error { return &ValidationError{Msg: msg} }
