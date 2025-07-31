package web

import "github.com/codetheuri/todolist/pkg/pagination"

type AlertifyPayload struct {
	Message string      `json:"message"`
	Theme   string      `json:"theme"` // e.g., "default", "success", "error"
	Type    interface{} `json:"type"`  // e.g., "success", "error", "
}

//---success response

// wrap actual response data
type Datapayload struct {
	Data interface{} `json:"data"`
}

// with pagination
type ListDatapayload struct {
	Data       interface{}            `json:"data"`
	Pagination *pagination.Metadata `json:"pagination"`
}

type SuccessResponse struct {
	Datapayload     *Datapayload     `json:"datapayload,omitempty"`
	ListDatapayload *ListDatapayload `json:"listdatapayload,omitempty"`
	AlertifyPayload *AlertifyPayload `json:"alertify,omitempty"`
	Metadata        interface{}      `json:"metadata,omitempty"` // for additional data
}

// ---error response
type ErrorPayload struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Errors  interface{} `json:"errors,omitempty"`
}

type APIErrorResponse struct {
	ErrorPayload    *ErrorPayload    `json:"errorpayload"`
	AlertifyPayload *AlertifyPayload `json:"alertify,omitempty"`
}
