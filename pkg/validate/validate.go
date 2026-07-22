// Package validate provides a fluent, chainable field validator for API input.
// It lives in pkg/ (not internal/) because it has zero project-specific dependencies
// and could be reused by any future Salio microservice.
//
// Usage:
//
//	v := validate.New()
//	v.Required("name", input.Name).
//	  MinLen("password", input.Password, 8).
//	  IsUUID("customer_id", input.CustomerID)
//
//	if v.HasErrors() {
//	    respondValidationError(w, v.Errors())
//	    return
//	}
package validate

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var rxEmail = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Validator accumulates field-level errors without panicking on the first failure.
// This gives the user all their mistakes at once, not one at a time.
type Validator struct {
	errs map[string]string
}

// New creates an empty Validator.
func New() *Validator {
	return &Validator{errs: make(map[string]string)}
}

// Required fails if the trimmed value is empty.
func (v *Validator) Required(field, value string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.errs[field] = fmt.Sprintf("%s cannot be blank", humanize(field))
	}
	return v
}

// MinLen fails if the value is shorter than the minimum length.
func (v *Validator) MinLen(field, value string, min int) *Validator {
	if len(value) < min {
		v.errs[field] = fmt.Sprintf("%s must be at least %d characters", humanize(field), min)
	}
	return v
}

// MaxLen fails if the value exceeds the maximum length.
func (v *Validator) MaxLen(field, value string, max int) *Validator {
	if len(value) > max {
		v.errs[field] = fmt.Sprintf("%s must be at most %d characters", humanize(field), max)
	}
	return v
}

// Email fails if the value is not a valid email address.
func (v *Validator) Email(field, value string) *Validator {
	if value == "" {
		return v
	}
	if _, err := mail.ParseAddress(value); err != nil || !rxEmail.MatchString(value) {
		v.errs[field] = fmt.Sprintf("%s must be a valid email address", humanize(field))
	}
	return v
}

// IsUUID fails if the value is not a valid UUID v4 string.
func (v *Validator) IsUUID(field, value string) *Validator {
	if value == "" {
		return v
	}
	if _, err := uuid.Parse(value); err != nil {
		v.errs[field] = fmt.Sprintf("%s must be a valid UUID", humanize(field))
	}
	return v
}

// IsURL fails if the value is not a valid HTTP/HTTPS URL.
func (v *Validator) IsURL(field, value string) *Validator {
	if value == "" {
		return v
	}
	u, err := url.ParseRequestURI(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		v.errs[field] = fmt.Sprintf("%s must be a valid URL", humanize(field))
	}
	return v
}

// OneOf fails if the value is not in the list of allowed values.
func (v *Validator) OneOf(field, value string, allowed ...string) *Validator {
	for _, a := range allowed {
		if value == a {
			return v
		}
	}
	v.errs[field] = fmt.Sprintf("%s must be one of: %s", humanize(field), strings.Join(allowed, ", "))
	return v
}

// GreaterThan fails if the numeric value is not greater than min.
func (v *Validator) GreaterThan(field string, value, min float64) *Validator {
	if value <= min {
		v.errs[field] = fmt.Sprintf("%s must be greater than %g", humanize(field), min)
	}
	return v
}

// MinNum fails if value is less than min.
func (v *Validator) MinNum(field string, value, min float64) *Validator {
	if value < min {
		v.errs[field] = fmt.Sprintf("%s must be at least %g", humanize(field), min)
	}
	return v
}

// MaxNum fails if value is greater than max.
func (v *Validator) MaxNum(field string, value, max float64) *Validator {
	if value > max {
		v.errs[field] = fmt.Sprintf("%s must be at most %g", humanize(field), max)
	}
	return v
}

// Matches fails if the string value does not match the given regex pattern.
func (v *Validator) Matches(field, value string, rx *regexp.Regexp, msg string) *Validator {
	if value == "" {
		return v
	}
	if !rx.MatchString(value) {
		v.errs[field] = fmt.Sprintf("%s %s", humanize(field), msg)
	}
	return v
}

// HasErrors returns true if any validation rule has failed.
func (v *Validator) HasErrors() bool {
	return len(v.errs) > 0
}

// Errors returns the map of field → message for use in the API error response.
func (v *Validator) Errors() map[string]string {
	return v.errs
}

// humanize converts snake_case field names to "field name" for readable messages.
func humanize(field string) string {
	return strings.ReplaceAll(field, "_", " ")
}
