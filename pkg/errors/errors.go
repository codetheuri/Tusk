package errors

import "fmt"

// application error interface
type AppError interface {
	Error() string
	Code() string
	Message() string
	Unwrap() error
	GetValidationErrors() interface{}
}

// basic error implementation
type appError struct {
	code    string
	message string
	err     error

	validationErrors interface{}
}

var _ AppError = (*appError)(nil)

func (e *appError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.code, e.message, e.err)
	}
	return fmt.Sprintf("[%s] %s", e.code, e.message)
}

func (e *appError) Code() string {
	return e.code
}

func (e *appError) Message() string {
	return e.message
}
func (e *appError) Unwrap() error {
	return e.err
}
func (e *appError) GetValidationErrors() interface{} {
	return e.validationErrors
}

// create a new generic application error
func New(code, message string, err error) AppError {
	return &appError{
		code:    code,
		message: message,
		err:     err,
	}
}

// specific err

// config issues
func ConfigError(message string, err error) AppError {
	return New("CONFIG_ERROR", message, err)
}

// database issues
func DatabaseError(message string, err error) AppError {
	return New("DATABASE_ERROR", message, err)
}

// resource not found
func NotFoundError(message string, err error) AppError {
	return New("NOT_FOUND", message, err)
}
// conflict errors (e.g. duplicate entries)
func ConflictError(message string, err error) AppError {
	return New("CONFLICT_ERROR", message, err)
}

// validation issues
func ValidationError(message string, err error, fieldErrors interface{}) AppError {
	return &appError{
		code:             "VALIDATION_ERROR",
		message:          message,
		err:              err,
		validationErrors: fieldErrors,
	}
}

// authentication issues
func AuthError(message string, err error) AppError {
	return New("AUTH_ERROR", message, err)
	// 	return &appError{
	// 	code:    "AUTH_ERROR",
	// 	message: message,
	// 	err:     err,
	// }
}

// authorization issues
func AuthorizationError(message string, err error) AppError {
	return New("AUTHORIZATION_ERROR", message, err)
}

// internal server error
func InternalServerError(message string, err error) AppError {
	return New("INTERNAL_SERVER_ERROR", message, err)
}

// external service error
func ExternalServiceError(message string, err error) AppError {
	return New("EXTERNAL_SERVICE_ERROR", message, err)
}
