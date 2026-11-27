// Package errors maps google.rpc.Code values to HTTP statuses and renders the
// Google API error envelope shared by every emulated product.
//
// Codes are the numeric values of google.rpc.Code
// (https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto),
// which align one-to-one with gRPC status codes.
package errors

import (
	"encoding/json"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Code is a google.rpc.Code value.
type Code int32

const (
	OK                 Code = 0
	Canceled           Code = 1
	Unknown            Code = 2
	InvalidArgument    Code = 3
	DeadlineExceeded   Code = 4
	NotFound           Code = 5
	AlreadyExists      Code = 6
	PermissionDenied   Code = 7
	ResourceExhausted  Code = 8
	FailedPrecondition Code = 9
	Aborted            Code = 10
	OutOfRange         Code = 11
	Unimplemented      Code = 12
	Internal           Code = 13
	Unavailable        Code = 14
	DataLoss           Code = 15
	Unauthenticated    Code = 16
)

var codeNames = map[Code]string{
	OK:                 "OK",
	Canceled:           "CANCELED",
	Unknown:            "UNKNOWN",
	InvalidArgument:    "INVALID_ARGUMENT",
	DeadlineExceeded:   "DEADLINE_EXCEEDED",
	NotFound:           "NOT_FOUND",
	AlreadyExists:      "ALREADY_EXISTS",
	PermissionDenied:   "PERMISSION_DENIED",
	ResourceExhausted:  "RESOURCE_EXHAUSTED",
	FailedPrecondition: "FAILED_PRECONDITION",
	Aborted:            "ABORTED",
	OutOfRange:         "OUT_OF_RANGE",
	Unimplemented:      "UNIMPLEMENTED",
	Internal:           "INTERNAL",
	Unavailable:        "UNAVAILABLE",
	DataLoss:           "DATA_LOSS",
	Unauthenticated:    "UNAUTHENTICATED",
}

// String returns the google.rpc.Code enum name (e.g. "NOT_FOUND").
func (c Code) String() string {
	if s, ok := codeNames[c]; ok {
		return s
	}
	return fmt.Sprintf("CODE_%d", int32(c))
}

// HTTPStatus maps a google.rpc.Code to the HTTP status used by Google REST APIs.
func HTTPStatus(c Code) int {
	switch c {
	case OK:
		return http.StatusOK
	case Canceled:
		return 499
	case Unknown, Internal, DataLoss:
		return http.StatusInternalServerError
	case InvalidArgument, FailedPrecondition, OutOfRange:
		return http.StatusBadRequest
	case DeadlineExceeded:
		return http.StatusGatewayTimeout
	case NotFound:
		return http.StatusNotFound
	case AlreadyExists, Aborted:
		return http.StatusConflict
	case PermissionDenied:
		return http.StatusForbidden
	case ResourceExhausted:
		return http.StatusTooManyRequests
	case Unimplemented:
		return http.StatusNotImplemented
	case Unavailable:
		return http.StatusServiceUnavailable
	case Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// GRPCCode converts to the numerically identical gRPC status code.
func GRPCCode(c Code) codes.Code { return codes.Code(c) }

type envelope struct {
	Error restError `json:"error"`
}

type restError struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Status  string           `json:"status"`
	Errors  []restErrorEntry `json:"errors"`
}

type restErrorEntry struct {
	Message string `json:"message"`
}

// WriteREST renders the Google API REST error envelope and sets the status code.
func WriteREST(w http.ResponseWriter, c Code, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(HTTPStatus(c))
	_ = json.NewEncoder(w).Encode(envelope{Error: restError{
		Code:    HTTPStatus(c),
		Message: msg,
		Status:  c.String(),
		Errors:  []restErrorEntry{{Message: msg}},
	}})
}

// GRPC returns a gRPC status error carrying the google.rpc.Code.
func GRPC(c Code, format string, args ...any) error {
	return status.Error(GRPCCode(c), fmt.Sprintf(format, args...))
}

// GRPCf is GRPC with printf-style arguments.
func GRPCf(c Code, format string, args ...any) error { return GRPC(c, format, args...) }
