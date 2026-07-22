package response

import (
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// Data represents the standardized success payload format
type Data[T any] struct {
	Success bool   `json:"success" default:"true"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// ErrorModel represents the standardized error payload format
type ErrorModel struct {
	Status  int               `json:"-"`
	Success bool              `json:"success" default:"false"`
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors,omitempty"`
}

func (e *ErrorModel) GetStatus() int {
	return e.Status
}

func (e *ErrorModel) Error() string {
	return e.Message
}

// SetupHuma configures Huma to use the custom ErrorModel system
func SetupHuma() {
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		errMap := make(map[string]string)

		for _, err := range errs {
			if detailer, ok := err.(huma.ErrorDetailer); ok {
				d := detailer.ErrorDetail()
				loc := d.Location
				loc = strings.TrimPrefix(loc, "body.")

				// If location is "body" and message is "expected required property <field> to be present", extract field name
				if loc == "body" && strings.Contains(d.Message, "expected required property") {
					parts := strings.Split(d.Message, "expected required property ")
					if len(parts) > 1 {
						fieldParts := strings.Split(parts[1], " to be present")
						if len(fieldParts) > 0 {
							loc = strings.TrimSpace(fieldParts[0])
						}
					}
				}

				if loc == "" || loc == "body" {
					loc = "server"
				}
				errMap[loc] = d.Message
			} else if err != nil {
				errMap["server"] = err.Error()
			}
		}

		if len(errMap) > 0 && (status == http.StatusUnprocessableEntity || status == http.StatusBadRequest) {
			msg = "Validation failed"
		}

		if len(errMap) == 0 {
			errMap = nil
		}

		return &ErrorModel{
			Status:  status,
			Success: false,
			Message: msg,
			Errors:  errMap,
		}
	}
}

// WriteJSON sends a standardized JSON response with status code.
func WriteJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(body))
}

// ValidationError creates a 400 Bad Request error with field-level error mappings.
func ValidationError(msg string, fieldErrors map[string]string) huma.StatusError {
	return &ErrorModel{
		Status:  http.StatusBadRequest,
		Success: false,
		Message: msg,
		Errors:  fieldErrors,
	}
}

// FieldError creates a 400 Bad Request error for a single field.
func FieldError(msg string, field string, fieldMsg string) huma.StatusError {
	return &ErrorModel{
		Status:  http.StatusBadRequest,
		Success: false,
		Message: msg,
		Errors:  map[string]string{field: fieldMsg},
	}
}
