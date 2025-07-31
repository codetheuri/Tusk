package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/pagination"
	"github.com/codetheuri/todolist/pkg/validators"
)

// SendJSON writes the given status code and data as JSON to the http.ResponseWriter.
func SendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
		}
	}
}
func RespondError(w http.ResponseWriter, err error, defaultStatus int, opts ...AlertifyOption) {
	apiErrResp := APIErrorResponse{
		ErrorPayload: &ErrorPayload{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "An unexpected error occurred",
		},
	}
	statusCode := defaultStatus
	var appErr appErrors.AppError
	if errors.As(err, &appErr) {
		apiErrResp.ErrorPayload.Code = appErr.Code()
		apiErrResp.ErrorPayload.Message = appErr.Message()
		// apiErrResp.ErrorPayload.Details = appErr.Details()

		//determone status code based on error code
		isValidationError := false
		switch appErr.Code() {
		case "AUTH_ERROR":
			statusCode = http.StatusUnauthorized
		case "NOT_FOUND":
			statusCode = http.StatusNotFound
		case "INVALID_INPUT":
			statusCode = http.StatusBadRequest
		case "FORBIDDEN":
			statusCode = http.StatusForbidden
		case "CONFLICT_ERROR":
			statusCode = http.StatusConflict
		case "CONFIG_ERROR", "DATABASE_ERROR":
			statusCode = http.StatusInternalServerError
		case "UNAUTHORIZED":
			statusCode = http.StatusUnauthorized
		case "VALIDATION_ERROR":
			isValidationError = true

			statusCode = http.StatusUnprocessableEntity
			if valErrors := appErr.GetValidationErrors(); valErrors != nil {
				if fieldErrors, ok := valErrors.([]validators.FieldError); ok {
					apiErrResp.ErrorPayload.Errors = fieldErrors
				} else {
					apiErrResp.ErrorPayload.Errors = valErrors
				}
			}

		case "INTERNAL_SERVER_ERROR":
			statusCode = http.StatusInternalServerError
		default:
			statusCode = http.StatusInternalServerError
			apiErrResp.ErrorPayload.Message = fmt.Sprintf("An unexpected application error occurred: %s", appErr.Message())
		}
		if !isValidationError {
			apiErrResp.AlertifyPayload = &AlertifyPayload{
				Message: appErr.Message(),
				Theme:   "danger",
				Type:    "alert",
			}
		}
	} else {
		statusCode = http.StatusInternalServerError
		apiErrResp.ErrorPayload.Message = "An unexpected server error occurred."
		apiErrResp.AlertifyPayload = &AlertifyPayload{
			Message: "An unexpected server error occurred.",
			Theme:   "danger",
			Type:    "alert",
		}
	}
	for _, opt := range opts {
		opt(&apiErrResp)
	}
	SendJSON(w, statusCode, apiErrResp)
}

func RespondData(w http.ResponseWriter, statusCode int, data interface{}, message string, opts ...SuccessOption) {
	resp := SuccessResponse{
		Datapayload: &Datapayload{
			Data: data,
		},
	}
	if message != "" {
		resp.AlertifyPayload = &AlertifyPayload{
			Message: message,
			Theme:   "success",
			Type:    "alert",
		}
	}
	for _, opt := range opts {
		opt(&resp)
	}
	SendJSON(w, statusCode, resp)
	
}
func RespondListData(w http.ResponseWriter, statusCode int, data interface{}, p *pagination.Metadata) {
	resp := SuccessResponse{
		ListDatapayload: &ListDatapayload{
			Data:       data,
			Pagination: p,
		},
	}
	
	
	SendJSON(w, statusCode, resp)
}
func RespondMessage(w http.ResponseWriter, statusCode int, message string, theme string, typ interface{}) {
	resp := SuccessResponse{
		AlertifyPayload: &AlertifyPayload{
			Message: message,
			Theme:   theme,
			Type:    typ,
		},
	}
	SendJSON(w, statusCode, resp)
}

// ---success response options
type SuccessOption func(*SuccessResponse)

func WithSuccessTheme(theme string) SuccessOption {
	return func(resp *SuccessResponse) {
		if resp.AlertifyPayload == nil {
			resp.AlertifyPayload = &AlertifyPayload{}
		}
		resp.AlertifyPayload.Theme = theme
	}
}	
func WithSuccessType(typ interface{}) SuccessOption {
	return func(resp *SuccessResponse) {
		if resp.AlertifyPayload == nil {
			resp.AlertifyPayload = &AlertifyPayload{}
		}
		resp.AlertifyPayload.Type = typ
	}
}
func WithSuccessMessage(message string) SuccessOption {
	return func(resp *SuccessResponse) {
		if resp.AlertifyPayload == nil {
			resp.AlertifyPayload = &AlertifyPayload{}
		}
		resp.AlertifyPayload.Message = message
	}
}
func WithSuccessOverride(message string, theme string, typ interface{}) SuccessOption {
	return func(resp *SuccessResponse) {
		resp.AlertifyPayload = &AlertifyPayload{
			Message: message,
			Theme:   theme,
			Type:    typ,
		}
	}
}
func WithoutSuccess() SuccessOption {
	return func(resp *SuccessResponse) {
		resp.AlertifyPayload = nil
	}
}
 func WithMetadata(data interface{}) SuccessOption {
	return func(resp *SuccessResponse) {
		resp.Metadata = data
	}
 }

// for error options
type AlertifyOption func(*APIErrorResponse)

func WithAlertifyTheme(theme string) AlertifyOption {
	return func(resp *APIErrorResponse) {
		if resp.AlertifyPayload == nil {
			resp.AlertifyPayload = &AlertifyPayload{}
		}
		resp.AlertifyPayload.Theme = theme
	}
}
func WithAlertifyType(typ interface{}) AlertifyOption {
	return func(resp *APIErrorResponse) {
		if resp.AlertifyPayload == nil {
			resp.AlertifyPayload = &AlertifyPayload{}
		}
		resp.AlertifyPayload.Type = typ
	}
}
func WithAlertifyMessage(message string) AlertifyOption {
	return func(resp *APIErrorResponse) {
		if resp.AlertifyPayload == nil {
			resp.AlertifyPayload = &AlertifyPayload{}
		}
		resp.AlertifyPayload.Message = message
	}
}
func WithAlertifyOverride(message string, theme string, typ string) AlertifyOption {
	return func(resp *APIErrorResponse) {
		resp.AlertifyPayload = &AlertifyPayload{
			Message: message,
			Theme:   theme,
			Type:    typ,
		}
	}
}

func WithoutAlertify() AlertifyOption {
	return func(resp *APIErrorResponse) {
		resp.AlertifyPayload = nil
	}
}
