package validate

import (
	"regexp"
	"testing"
)

func TestValidator_Required(t *testing.T) {
	v := New()
	v.Required("name", "").Required("email", "  ")
	if !v.HasErrors() {
		t.Error("expected validation errors for empty fields")
	}
	if len(v.Errors()) != 2 {
		t.Errorf("expected 2 errors, got %d", len(v.Errors()))
	}

	v2 := New()
	v2.Required("name", "John")
	if v2.HasErrors() {
		t.Error("unexpected validation error for non-empty field")
	}
}

func TestValidator_MinLen_MaxLen(t *testing.T) {
	v := New()
	v.MinLen("password", "123", 6).MaxLen("bio", "12345678901", 10)
	if !v.HasErrors() {
		t.Error("expected min len and max len errors")
	}

	v2 := New()
	v2.MinLen("password", "123456", 6).MaxLen("bio", "12345", 10)
	if v2.HasErrors() {
		t.Error("unexpected min len / max len error")
	}
}

func TestValidator_Email(t *testing.T) {
	v := New()
	v.Email("email", "invalid-email")
	if !v.HasErrors() {
		t.Error("expected email validation error")
	}

	v2 := New()
	v2.Email("email", "user@example.com")
	if v2.HasErrors() {
		t.Error("unexpected email validation error")
	}
}

func TestValidator_IsUUID(t *testing.T) {
	v := New()
	v.IsUUID("id", "not-a-uuid")
	if !v.HasErrors() {
		t.Error("expected UUID error")
	}

	v2 := New()
	v2.IsUUID("id", "550e8400-e29b-41d4-a716-446655440000")
	if v2.HasErrors() {
		t.Error("unexpected UUID error")
	}
}

func TestValidator_OneOf(t *testing.T) {
	v := New()
	v.OneOf("role", "guest", "admin", "user")
	if !v.HasErrors() {
		t.Error("expected oneOf error")
	}

	v2 := New()
	v2.OneOf("role", "admin", "admin", "user")
	if v2.HasErrors() {
		t.Error("unexpected oneOf error")
	}
}

func TestValidator_Numbers(t *testing.T) {
	v := New()
	v.GreaterThan("age", 10, 18).MinNum("score", 5, 10).MaxNum("percentage", 105, 100)
	if !v.HasErrors() {
		t.Error("expected number validation errors")
	}
	if len(v.Errors()) != 3 {
		t.Errorf("expected 3 errors, got %d", len(v.Errors()))
	}
}

func TestValidator_Matches(t *testing.T) {
	rx := regexp.MustCompile(`^[a-z]+$`)
	v := New()
	v.Matches("username", "USER123", rx, "must contain lowercase letters only")
	if !v.HasErrors() {
		t.Error("expected regex match error")
	}
}
