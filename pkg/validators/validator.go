package validators

import (
	"fmt"
	"reflect"
	"strings"

	gv "github.com/go-playground/validator/v10"
)

type Validator struct {
	validate *gv.Validate
}

func NewValidator() *Validator {
	v := gv.New()
	v.RegisterTagNameFunc(func(fld reflect.StructField) string{
		name := strings.SplitN(fld.Tag.Get("json"), ",",2)[0]
		if name == "-" || name == ""{
			return fld.Name
		}
		return name
	})
	return &Validator{
		validate: v,
	}
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"error"`
}

func (v *Validator) Struct(s interface{}) []FieldError {
	err := v.validate.Struct(s)
	if err == nil {
		return nil
	}
	var fieldErrors []FieldError
	for _, err := range err.(gv.ValidationErrors) {
		fieldErrors = append(fieldErrors, FieldError{
			Field:   err.Field(),
			Message: parseTag(err),
		})
	}
	return fieldErrors
}

func parseTag(fe gv.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "min":
		return fmt.Sprintf("Minimum length is %s", fe.Param())
	case "max":
		return fmt.Sprintf("Maximum length is %s", fe.Param())
	case "email":
		return "Invalid email format"
	case "url":
		return "Invalid URL format"
	case "len":
		return fmt.Sprintf("Length must be %s", fe.Param())
	case "gt":
		return fmt.Sprintf("Must be greater than %s", fe.Param())
	case "gte":
		return fmt.Sprintf("Must be greater than or equal to %s", fe.Param())
	case "lt":
		return fmt.Sprintf("Must be less than %s", fe.Param())
	case "lte":
		return fmt.Sprintf("Must be less than or equal to %s", fe.Param())
	case "eqfield":
		return fmt.Sprintf("Must be equal to %s", strings.ToLower(fe.Param())) // e.g., password_confirmation
	case "nefield":
		return fmt.Sprintf("Must not be equal to %s", strings.ToLower(fe.Param()))
	case "alpha":
		return "Must contain only alphabetic characters"
	case "numeric":
		return "Must contain only numeric characters"
	case "alphanum":
		return "Must contain only alphanumeric characters"
	case "e164": // For phone numbers
		return "Invalid phone number format (E.164)"
	default:
		return fmt.Sprintf("Invalid value for %s", fe.Tag())
	}
}
