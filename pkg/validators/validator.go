package validator

import (
	"encoding/json"
	"net/http"

	gv "github.com/go-playground/validator/v10"
)

type validator struct {
	validate *gv.Validate
}
func NewValidator() *validator {}
type ValidationError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

func ValidateAndRespond(w http.ResponseWriter, data interface{}) bool {
	err := validate.Struct(data)
	if err == nil {
		return true
	}
 var validationErrors []ValidationError
	for _, err := range err.(validator.ValidationErrors) {
        validationErrors = append(validationErrors, ValidationError{
            Field: err.Field(),
            Error: parseTag(err),
        })
    }
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Validation failed",
		"errors":  validationErrors,
	})

	return false
}

func parseTag(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "min":
		return "Minimum length is " + fe.Param()
	case "max":
		return "Maximum length is " + fe.Param()
	case "email":
		return "Invalid email format"
	case "url":
		return "Invalid URL format"
	default:
		return "invalid value"
	}
}

