package model

import (
	"encoding/json"
	"time"
)

// AppError is a structured application error with an HTTP status code,
// a machine-readable code, a human-readable message, and an optional
// inner error for internal logging.
type AppError struct {
	Status  int
	Code    string
	Message string
	Err     error
}

// Error returns a JSON-like string representation of the error.
func (e *AppError) Error() string {
	b, _ := json.Marshal(map[string]interface{}{
		"status":  e.Status,
		"code":    e.Code,
		"message": e.Message,
	})
	return string(b)
}

// Unwrap returns the inner error (if any) for errors.Is/errors.As.
func (e *AppError) Unwrap() error {
	return e.Err
}

// --- Helper constructors ---

// ErrValidation creates a 400 Bad Request error.
func ErrValidation(msg string, err error) *AppError {
	return &AppError{Status: 400, Code: "ERR_VALIDATION", Message: msg, Err: err}
}

// ErrUnauthorized creates a 401 Unauthorized error.
func ErrUnauthorized(msg string, err error) *AppError {
	return &AppError{Status: 401, Code: "ERR_UNAUTHORIZED", Message: msg, Err: err}
}

// ErrForbidden creates a 403 Forbidden error.
func ErrForbidden(msg string, err error) *AppError {
	return &AppError{Status: 403, Code: "ERR_FORBIDDEN", Message: msg, Err: err}
}

// ErrNotFound creates a 404 Not Found error.
func ErrNotFound(msg string, err error) *AppError {
	return &AppError{Status: 404, Code: "ERR_NOT_FOUND", Message: msg, Err: err}
}

// ErrConflict creates a 409 Conflict error.
func ErrConflict(msg string, err error) *AppError {
	return &AppError{Status: 409, Code: "ERR_CONFLICT", Message: msg, Err: err}
}

// ErrUnprocessable creates a 422 Unprocessable Entity error.
func ErrUnprocessable(msg string, err error) *AppError {
	return &AppError{Status: 422, Code: "ERR_UNPROCESSABLE", Message: msg, Err: err}
}

// ErrInternal creates a 500 Internal Server Error.
func ErrInternal(msg string, err error) *AppError {
	return &AppError{Status: 500, Code: "ERR_INTERNAL", Message: msg, Err: err}
}

// ErrorResponse is the JSON body returned to the client on errors.
type ErrorResponse struct {
	Status    int    `json:"status"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// FormatError converts any error to an *AppError. If it already is an
// *AppError it is returned as-is; otherwise it is wrapped in ErrInternal.
func FormatError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return ErrInternal("An internal error occurred", err)
}

// NewErrorResponse creates an ErrorResponse from an *AppError, populating
// the timestamp with the current time in RFC3339 format.
func NewErrorResponse(err *AppError) ErrorResponse {
	return ErrorResponse{
		Status:    err.Status,
		Code:      err.Code,
		Message:   err.Message,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}


