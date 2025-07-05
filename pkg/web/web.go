package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	appErrors "github.com/codetheuri/todolist/pkg/errors"
)

func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		}
	}
}

func RespondError(w http.ResponseWriter, err error, defaultStatus int) {
	errorResponse := struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{
		Code:    "INTERNAL_SERVER_ERROR",
		Message: "An unexpected error occurred",
	}
	statusCode := defaultStatus

	// cast custom AppError
	var appErr appErrors.AppError

	if errors.As(err, &appErr) {
		errorResponse.Code = appErr.Code()
		errorResponse.Message = appErr.Message()

		// map specific error codes to HTTP status codes
		switch appErr.Code() {
		case "NOT_FOUND":
			statusCode = http.StatusNotFound
		case "INVALID_INPUT":
			statusCode = http.StatusBadRequest
		case "UNAUTHORIZED":
			statusCode = http.StatusUnauthorized
		case "FORBIDDEN":
			statusCode = http.StatusForbidden
		case "CONFLICT_ERROR":
			statusCode = http.StatusConflict
		case "CONFIG_ERROR", "DATABASE_ERROR":
			statusCode = http.StatusInternalServerError
		case "VALIDATION_ERROR":
			statusCode = http.StatusUnprocessableEntity
		default:
			statusCode = http.StatusInternalServerError
		}

	} else {
		errorResponse.Code = "An unexpected error occurred"
	}
	RespondJSON(w, statusCode, errorResponse)
}
