package response

import (
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, 200, `{"success":true}`)

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
	}
	if rec.Body.String() != `{"success":true}` {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

func TestErrorModel(t *testing.T) {
	errModel := &ErrorModel{
		Status:  400,
		Success: false,
		Message: "Validation failed",
		Errors:  map[string]string{"name": "required"},
	}

	if errModel.GetStatus() != 400 {
		t.Errorf("expected status 400, got %d", errModel.GetStatus())
	}
	if errModel.Error() != "Validation failed" {
		t.Errorf("expected message 'Validation failed', got %s", errModel.Error())
	}
}
