package tonic

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"

	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/validators"
	"github.com/codetheuri/todolist/pkg/web"
)

//Tonic uses reflection to pull docs straight from your routes and struct binding tags — both request and responses.
//It also provides a clean separation between your HTTP layer and business logic by using pure functions as handlers.
//The Adapter function converts these pure functions into standard http.HandlerFunc, handling JSON decoding, validation, and error responses automatically.

type Response struct {
	Data   interface{}
	Status int
}

func NewResponse(data interface{}) *Response {
	return &Response{
		Data:   data,
		Status: http.StatusOK,
	}
}
func NewCreatedResponse(data interface{}) *Response {
	return &Response{Data: data, Status: http.StatusCreated}
}

func NewNoContentResponse() *Response {
	return &Response{
		Data:   nil,
		Status: http.StatusNoContent,
	}
}

type HandlerFunc func(ctx context.Context, input interface{}) (*Response, error)

func Adapter(pureFunc HandlerFunc, inputType interface{}, validator *validators.Validator) http.HandlerFunc {
	inputTypeReflect := reflect.TypeOf(inputType)
	var inputTypeElem reflect.Type
	if inputTypeReflect.Kind() == reflect.Ptr {
		inputTypeElem = inputTypeReflect.Elem()
	} else {
		inputTypeElem = inputTypeReflect
	}
	return func(w http.ResponseWriter, r *http.Request) {
		input := reflect.New(inputTypeElem).Interface()
		if r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(input); err != nil {
				web.RespondError(w, appErrors.ValidationError("invalid request payload format", err, nil), http.StatusBadRequest)
				return
			}
		}

		if validationErrors := validator.Struct(input); validationErrors != nil {
			web.RespondError(w, appErrors.ValidationError("validation failed", nil, validationErrors), http.StatusUnprocessableEntity)
			return
		}

		tonicResponse, err := pureFunc(r.Context(), input)

		if err != nil {
			web.RespondError(w, err, http.StatusInternalServerError)
			return
		}
		if tonicResponse == nil {
			web.RespondError(w, appErrors.InternalServerError("handler returned nil response struct", nil), http.StatusInternalServerError)
			return
		}
		statusCode := tonicResponse.Status
		if statusCode == http.StatusNoContent {

			web.RespondMessage(w, http.StatusNoContent, "Operation successful", "success", "toast")
			return
		}

		web.RespondData(w, statusCode, tonicResponse.Data, "", web.WithoutSuccess())
	}
}
