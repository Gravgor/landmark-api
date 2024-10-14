package errors

import "errors"

var (
	ErrNotFound                 = errors.New("resource not found")
	ErrAlreadyExists            = errors.New("resource already exists")
	ErrInvalidInput             = errors.New("invalid input")
	ErrInsufficientPermission   = errors.New("insufficient permission")
	ErrDatabaseError            = errors.New("database error")
	ErrCacheError               = errors.New("cache error")
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrInsufficientSubscription = errors.New("insufficient subscription")
)

type Error struct {
	Err     error
	Message string
	Code    string
}

func (e *Error) Error() string {
	return e.Message
}

func Wrap(err error, message string) *Error {
	return &Error{
		Err:     err,
		Message: message,
		Code:    "INTERNAL_ERROR",
	}
}
