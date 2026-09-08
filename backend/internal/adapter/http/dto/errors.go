package dto

import (
	"errors"
	"net/http"

	"github.com/SebasElDev/fullstack-calculator/backend/internal/domain"
)

// ErrorCode is the machine-readable classification of a failure. The values
// mirror the ErrorCode enum of api/openapi.yaml.
type ErrorCode string

// The error codes of the contract.
const (
	CodeInvalidRequest       ErrorCode = "INVALID_REQUEST"
	CodeUnsupportedOperation ErrorCode = "UNSUPPORTED_OPERATION"
	CodeInvalidOperands      ErrorCode = "INVALID_OPERANDS"
	CodeDivisionByZero       ErrorCode = "DIVISION_BY_ZERO"
	CodeUndefinedResult      ErrorCode = "UNDEFINED_RESULT"
	CodeResultOutOfRange     ErrorCode = "RESULT_OUT_OF_RANGE"
	CodeInternalError        ErrorCode = "INTERNAL_ERROR"
)

// InternalErrorMessage is the only message a client ever sees for a 500: the
// real cause is logged, never leaked.
const InternalErrorMessage = "an unexpected error occurred"

// ErrInvalidRequest marks a body that could not be decoded into a
// CalculateRequest. It belongs to the adapter layer: the domain never sees
// malformed JSON.
var ErrInvalidRequest = errors.New("invalid request body")

// ErrorBody is the payload of ErrorResponse.
type ErrorBody struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// ErrorResponse is the error envelope of every non-2xx response.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// NewErrorResponse builds an error envelope.
func NewErrorResponse(code ErrorCode, message string) ErrorResponse {
	return ErrorResponse{Error: ErrorBody{Code: code, Message: message}}
}

// errorMapping is the single translation table between the sentinel errors of
// the inner layers and the HTTP surface. Order matters only in that the first
// match wins; the sentinels are mutually exclusive.
var errorMapping = []struct {
	sentinel error
	status   int
	code     ErrorCode
}{
	{ErrInvalidRequest, http.StatusBadRequest, CodeInvalidRequest},
	{domain.ErrUnsupportedOperation, http.StatusBadRequest, CodeUnsupportedOperation},
	{domain.ErrInvalidArity, http.StatusBadRequest, CodeInvalidOperands},
	{domain.ErrDivisionByZero, http.StatusUnprocessableEntity, CodeDivisionByZero},
	{domain.ErrUndefinedResult, http.StatusUnprocessableEntity, CodeUndefinedResult},
	{domain.ErrResultOutOfRange, http.StatusUnprocessableEntity, CodeResultOutOfRange},
}

// MapError translates an error into the status code and body the contract
// prescribes. Unrecognised errors become a generic 500: their message is never
// exposed.
func MapError(err error) (int, ErrorResponse) {
	for _, mapping := range errorMapping {
		if errors.Is(err, mapping.sentinel) {
			return mapping.status, NewErrorResponse(mapping.code, err.Error())
		}
	}
	return http.StatusInternalServerError, NewErrorResponse(CodeInternalError, InternalErrorMessage)
}
