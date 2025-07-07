package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/validators"
)

func DataResponse(w http.ResponseWriter ,data interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}

	// if data is a map, return it directly
	// if m, ok := data.(map[string]interface{}); ok {
	// 	return m
	// }

	// otherwise, wrap it in a map with "data" key
	return map[string]interface{}{"datapayload": data}
}
func ErrorResponse(w http.ResponseWriter ,status int,  data interface{}) map[string]interface{} {
	if data == nil{
		return nil
	}
	return map[string]interface{}{"errorpayload": data}
}
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
     // Ensure the data is wrapped in a map
	 response := map[string]interface{}{"data": data}
	//  datapayload := (map[string]interface{}{"datapayload":response})
	 datapayload := DataResponse(w, response)
		
	if datapayload != nil {
		if err := json.NewEncoder(w).Encode(datapayload); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		}
	}
	//  data = DataResponse(w, data)
	// return DataResponse(w,data) // Ensure the response is wrapped in a map
	// return  map[string]interface{}{"datapayload": response} // Ensure the response is wrapped in a map
}

func RespondError(w http.ResponseWriter, err error, defaultStatus int) {
	errorResponse := struct {
		Code    string      `json:"code"`
		Message string      `json:"message"`
		Errors  interface{} `json:"errors,omitempty"`
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

		// if it's a validation error, extract the field errors
		if appErr.Code() == "VALIDATION_ERROR" {
			statusCode = http.StatusUnprocessableEntity
			if valErrors := appErr.GetValidationErrors(); valErrors != nil {
				if fieldErrors, ok := valErrors.([]validators.FieldError); ok {
					errorResponse.Errors = fieldErrors
				}
			}

		} else {
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
		}
	} else {
		errorResponse.Message = "An unexpected error occurred"
	}
	// response :=   ErrorResponse(w, statusCode,errorResponse)
  
	RespondJSON(w, statusCode, errorResponse)
}
