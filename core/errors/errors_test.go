package errors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPStatusMapping(t *testing.T) {
	cases := map[Code]int{
		OK:                 200,
		InvalidArgument:    400,
		NotFound:           404,
		AlreadyExists:      409,
		PermissionDenied:   403,
		Unauthenticated:    401,
		ResourceExhausted:  429,
		FailedPrecondition: 400,
		Unimplemented:      501,
		Internal:           500,
		Unavailable:        503,
	}
	for c, want := range cases {
		if got := HTTPStatus(c); got != want {
			t.Errorf("HTTPStatus(%v) = %d, want %d", c, got, want)
		}
	}
}

func TestWriteRESTEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteREST(rec, NotFound, "Secret %s not found", "projects/p/secrets/s")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var body struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != 404 || body.Error.Status != "NOT_FOUND" {
		t.Errorf("unexpected envelope: %+v", body.Error)
	}
	if body.Error.Message != "Secret projects/p/secrets/s not found" {
		t.Errorf("unexpected message: %q", body.Error.Message)
	}
}

func TestCodeString(t *testing.T) {
	if NotFound.String() != "NOT_FOUND" {
		t.Errorf("got %q", NotFound.String())
	}
	if Code(999).String() != "CODE_999" {
		t.Errorf("got %q", Code(999).String())
	}
}
