package fiberadapter

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/SebasElDev/fullstack-calculator/backend/internal/adapter/http/dto"
)

// TestNotFoundErrorIsFibersSentinel pins the contract Fiber relies on: a route
// miss raised by this package must be recognised as fiber.ErrNotFound while
// keeping its descriptive message and its NOT_FOUND mapping.
func TestNotFoundErrorIsFibersSentinel(t *testing.T) {
	t.Parallel()

	err := newNotFoundError(http.MethodGet, "/api/v1/nope")

	if !errors.Is(err, fiber.ErrNotFound) {
		t.Fatalf("errors.Is(err, fiber.ErrNotFound) = false; want true")
	}
	if got, want := err.Error(), "no route for GET /api/v1/nope"; got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}

	status, body := resolveError(err)
	if status != http.StatusNotFound {
		t.Errorf("status = %d; want 404", status)
	}
	if body.Error.Code != dto.CodeNotFound {
		t.Errorf("code = %q; want %q", body.Error.Code, dto.CodeNotFound)
	}
	if body.Error.Message != err.Error() {
		t.Errorf("message = %q; want %q", body.Error.Message, err.Error())
	}
}
