package fiberadapter

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/SebasElDev/fullstack-calculator/backend/internal/adapter/http/dto"
)

// resolveError turns any error into the status code and JSON envelope the
// contract prescribes. It is the single place where that decision is made:
// the error boundary uses it to write the response and the request logger uses
// it to report the status a failing request will end up with.
//
// Errors raised by Fiber itself (an unmatched route, a body that exceeds the
// limit) carry their own status. Since api/openapi.yaml has no NOT_FOUND code,
// client-side Fiber errors are reported as INVALID_REQUEST — the request is
// malformed for this API — while server-side ones fall through to the generic
// INTERNAL_ERROR of dto.MapError, which never leaks internals.
func resolveError(err error) (int, dto.ErrorResponse) {
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) && fiberErr.Code < fiber.StatusInternalServerError {
		return fiberErr.Code, dto.NewErrorResponse(dto.CodeInvalidRequest, fiberErr.Message)
	}
	return dto.MapError(err)
}

// newNotFoundError builds the single "unknown route" error used both by the
// API catch-all and by the static handler when there is no SPA shell to serve.
func newNotFoundError(c fiber.Ctx) error {
	return fiber.NewError(fiber.StatusNotFound, "no route for "+c.Method()+" "+c.Path())
}

// errorHandler is the application's error boundary: every error returned by a
// handler or by the recover middleware becomes a JSON error response here.
func errorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		status, body := resolveError(err)
		if status >= fiber.StatusInternalServerError {
			// The generic message went to the client; the cause goes to the log.
			logger.Error("request failed",
				slog.String("method", c.Method()),
				slog.String("path", c.Path()),
				slog.String("request_id", c.RequestID()),
				slog.Any("error", err),
			)
		}
		return c.Status(status).JSON(body)
	}
}
