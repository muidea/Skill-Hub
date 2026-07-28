package http

import "errors"

// Common HTTP errors used throughout the package
var (
	// ErrURLNotFound is returned when a requested URL is not found
	ErrURLNotFound = errors.New("the requested url was not found on this server")

	// ErrMethodNotAllowed is returned when an HTTP method is not allowed for a route
	ErrMethodNotAllowed = errors.New("no matching http method found")

	// ErrStaticFileNotFound is returned when a static file cannot be found
	ErrStaticFileNotFound = errors.New("static file not found")

	// ErrInvalidProxyTarget is returned when a proxy target URL is invalid
	ErrInvalidProxyTarget = errors.New("illegal proxy target URL")

	// ErrEmptyFilePath is returned when an empty file path is provided
	ErrEmptyFilePath = errors.New("empty file path")
)

// StaticError represents an error with static file serving
type StaticError struct {
	Path string
	Err  error
}

func (e *StaticError) Error() string {
	return "static file error: " + e.Path + ": " + e.Err.Error()
}

func (e *StaticError) Unwrap() error {
	return e.Err
}

// NewStaticError creates a new StaticError
func NewStaticError(path string, err error) error {
	return &StaticError{
		Path: path,
		Err:  err,
	}
}
