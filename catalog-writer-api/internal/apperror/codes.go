package apperror

import "net/http"

func Validation(msg string) *AppError {
	return &AppError{Code: "VALIDATION_ERROR", Message: msg, Status: http.StatusBadRequest}
}

func NotFound(msg string) *AppError {
	return &AppError{Code: "NOT_FOUND", Message: msg, Status: http.StatusNotFound}
}

func Conflict(msg string) *AppError {
	return &AppError{Code: "CONFLICT", Message: msg, Status: http.StatusConflict}
}

func Unauthorized(msg string) *AppError {
	return &AppError{Code: "UNAUTHORIZED", Message: msg, Status: http.StatusUnauthorized}
}

// Internal wraps an unexpected error (DB failure, network error, etc).
// The original error is kept for logging (via Unwrap) but never shown
// to the caller — Message is always the generic string below.
func Internal(cause error) *AppError {
	return &AppError{Code: "INTERNAL_ERROR", Message: "internal server error", Status: http.StatusInternalServerError, cause: cause}
}
