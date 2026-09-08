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
// Errors raised by Fiber itself carry their own status. An unmatched route or
// method becomes NOT_FOUND; every other client-side Fiber error (for example a
// body over the limit) is reported as INVALID_REQUEST — the request is
// malformed for this API. Server-side ones fall through to the generic
// INTERNAL_ERROR of dto.MapError, which never leaks internals.
func resolveError(err error) (int, dto.ErrorResponse) {
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) && fiberErr.Code < fiber.StatusInternalServerError {
		code := dto.CodeInvalidRequest
		if fiberErr.Code == fiber.StatusNotFound {
			code = dto.CodeNotFound
		}
		return fiberErr.Code, dto.NewErrorResponse(code, fiberErr.Message)
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
//
// Rejections are logged here as well as in the request logger because some of
// them never reach the middleware chain: fasthttp raises a body-limit error
// while it is still reading the request, so this boundary is the only place
// that sees every request the server turned away.
func errorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		status, body := resolveError(err)
		attrs := []any{
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.String("request_id", c.RequestID()),
		}
		if status >= fiber.StatusInternalServerError {
			// The generic message went to the client; the cause goes to the log.
			logger.Error("request failed", append(attrs, slog.Any("error", err))...)
		} else {
			logger.Warn("request rejected", append(attrs, slog.String("code", string(body.Error.Code)))...)
		}
		return c.Status(status).JSON(body)
	}
}
