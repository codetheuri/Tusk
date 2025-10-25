package errors

import (
	"fmt"
	"strings"
	"unicode"

	"errors"

	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

const (
	MySQLUniqueConstraintError  = 1062
)

// TranslateError maps raw database errors (like unique constraints) into standard appErrors.
// This function should be called inside your repository methods after a DB operation.

func TranslateError(err error) error {
	if err == nil {
		return nil
	}

	// 1. Handle GORM's built-in errors
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return appErrors.NotFoundError("resource not found", err)
	}

	// 2. Handle specific driver errors (MySQL/MariaDB)
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case MySQLUniqueConstraintError:
			// Extract the conflicting field name from the error message.
			// Example MySQL message: "Duplicate entry 'test@example.com' for key 'users.email'"
			fieldName := extractFieldName(mysqlErr.Message)

			// Use the extracted field name in the ConflictError message.
			return appErrors.ConflictError(fmt.Sprintf("%s already exists.", fieldName), err)

			// Add cases for Foreign Key violation (e.g., 1452) or other specific errors here.
		}
	}

	// 3. Default fallback for unknown database errors
	return appErrors.DatabaseError("database operation failed", err)
}

// extractFieldName attempts to parse the column name from a standard MySQL duplicate entry error message.
func extractFieldName(message string) string {
	parts := strings.Split(message, "key '")
	if len(parts) > 1 {

		keyPart := strings.Trim(strings.Split(parts[1], "'")[0], "`")

		if lastDot := strings.LastIndex(keyPart, "."); lastDot != -1 {
			return toTitleCase(keyPart[lastDot+1:])
		}
		return toTitleCase(keyPart)
	}
	return "Unique field"
}

func toTitleCase(s string) string {
	if s == "" {
		return ""
	}
	// Capitalize the first rune and append the rest of the string
	return string(unicode.ToUpper(rune(s[0]))) + s[1:]
}
