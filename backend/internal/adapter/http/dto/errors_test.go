package dto_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/SebasElDev/fullstack-calculator/backend/internal/adapter/http/dto"
	"github.com/SebasElDev/fullstack-calculator/backend/internal/domain"
)

func TestMapError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantCode    dto.ErrorCode
		wantMessage string
	}{
		{
			name:        "malformed body",
			err:         fmt.Errorf("%w: unexpected end of JSON input", dto.ErrInvalidRequest),
			wantStatus:  http.StatusBadRequest,
			wantCode:    dto.CodeInvalidRequest,
			wantMessage: "invalid request body: unexpected end of JSON input",
		},
		{
			name:       "unsupported operation",
			err:        fmt.Errorf("%w: %q", domain.ErrUnsupportedOperation, "cube"),
			wantStatus: http.StatusBadRequest,
			wantCode:   dto.CodeUnsupportedOperation,
		},
		{
			name:       "invalid arity",
			err:        domain.ErrInvalidArity,
			wantStatus: http.StatusBadRequest,
			wantCode:   dto.CodeInvalidOperands,
		},
		{
			name:       "division by zero",
			err:        domain.ErrDivisionByZero,
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   dto.CodeDivisionByZero,
		},
		{
			name:       "undefined result",
			err:        domain.ErrUndefinedResult,
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   dto.CodeUndefinedResult,
		},
		{
			name:       "result out of range",
			err:        domain.ErrResultOutOfRange,
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   dto.CodeResultOutOfRange,
		},
		{
			name:        "unknown error is generic",
			err:         errors.New("connection reset by the database we do not have"),
			wantStatus:  http.StatusInternalServerError,
			wantCode:    dto.CodeInternalError,
			wantMessage: dto.InternalErrorMessage,
		},
		{
			name:        "nil error is treated as internal",
			err:         nil,
			wantStatus:  http.StatusInternalServerError,
			wantCode:    dto.CodeInternalError,
			wantMessage: dto.InternalErrorMessage,
		},
		{
			name:       "deeply wrapped sentinel is still recognised",
			err:        fmt.Errorf("handler: %w", fmt.Errorf("service: %w", domain.ErrDivisionByZero)),
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   dto.CodeDivisionByZero,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			status, body := dto.MapError(tc.err)
			if status != tc.wantStatus {
				t.Errorf("MapError(%v) status = %d; want %d", tc.err, status, tc.wantStatus)
			}
			if body.Error.Code != tc.wantCode {
				t.Errorf("MapError(%v) code = %q; want %q", tc.err, body.Error.Code, tc.wantCode)
			}
			if tc.wantMessage != "" && body.Error.Message != tc.wantMessage {
				t.Errorf("MapError(%v) message = %q; want %q", tc.err, body.Error.Message, tc.wantMessage)
			}
			if body.Error.Message == "" {
				t.Errorf("MapError(%v) produced an empty message", tc.err)
			}
		})
	}
}

// TestMapErrorCoversEveryDomainSentinel fails whenever a new sentinel is added
// to the domain without extending the translation table.
func TestMapErrorCoversEveryDomainSentinel(t *testing.T) {
	t.Parallel()

	sentinels := []error{
		dto.ErrInvalidRequest,
		domain.ErrUnsupportedOperation,
		domain.ErrInvalidArity,
		domain.ErrDivisionByZero,
		domain.ErrUndefinedResult,
		domain.ErrResultOutOfRange,
	}

	for _, sentinel := range sentinels {
		status, body := dto.MapError(sentinel)
		if status == http.StatusInternalServerError {
			t.Errorf("MapError(%v) fell through to 500; the sentinel is not in the table", sentinel)
		}
		if body.Error.Code == dto.CodeInternalError {
			t.Errorf("MapError(%v) returned INTERNAL_ERROR; want a specific code", sentinel)
		}
	}
}

func TestNewErrorResponse(t *testing.T) {
	t.Parallel()

	got := dto.NewErrorResponse(dto.CodeInvalidRequest, "boom")
	want := dto.ErrorResponse{Error: dto.ErrorBody{Code: dto.CodeInvalidRequest, Message: "boom"}}
	if got != want {
		t.Fatalf("NewErrorResponse = %+v; want %+v", got, want)
	}
}
