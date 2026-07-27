// Package apperror defines a single error type that carries an HTTP
// status and a stable machine-readable code, so every layer above the
// repository can return one error type and the HTTP layer knows exactly
// how to render it — no type-switching on driver-specific errors at the
// controller.
package apperror

type AppError struct {
	Code    string // stable, machine-readable — safe to key client logic off
	Message string // safe to show a caller
	Status  int    // HTTP status the delivery layer should respond with
	cause   error  // internal-only, never serialized
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.cause }
