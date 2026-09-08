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

// routeNotFoundError is the "unknown route" error of this API. It carries a
// descriptive message but still answers true to errors.Is(err, fiber.ErrNotFound),
// which Fiber relies on to tell an ordinary miss from a broken middleware
// chain when it walks the stack for a request the server itself rejected.
type routeNotFoundError struct {
	cause *fiber.Error
}

// newNotFoundError builds the single "unknown route" error used both by the
// API catch-all and by the static handler when there is no SPA shell to serve.
func newNotFoundError(method, path string) error {
	return routeNotFoundError{cause: fiber.NewError(fiber.StatusNotFound, "no route for "+method+" "+path)}
}

func (e routeNotFoundError) Error() string { return e.cause.Message }

// Unwrap exposes the underlying *fiber.Error so resolveError sees its status.
func (e routeNotFoundError) Unwrap() error { return e.cause }

// Is makes every route miss equivalent to Fiber's own sentinel.
func (routeNotFoundError) Is(target error) bool { return target == fiber.ErrNotFound }

// errorHandler is the application's error boundary: every error returned by a
// handler or by the recover middleware becomes a JSON error response here.
//
// The request logger reports the outcome of every request that runs through
// the middleware chain. Requests the HTTP server rejects while still reading
// them (a body over transportBodyLimit) take a different route: Fiber walks the
// middleware chain without a matching handler, so the logger records a miss,
// and only then hands the real error to this boundary. Whenever the status
// about to be sent differs from the one that was logged — or nothing was
// logged at all — the boundary writes the authoritative line itself.
func errorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		status, body := resolveError(err)
		attrs := []any{
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.String("request_id", c.RequestID()),
		}
		loggedStatus, _ := c.Locals(requestLoggedKey).(int)
		switch {
		case status >= fiber.StatusInternalServerError:
			// The generic message went to the client; the cause goes to the log.
			logger.Error("request failed", append(attrs, slog.Any("error", err))...)
		case loggedStatus != status:
			logger.Warn("request rejected", append(attrs, slog.String("code", string(body.Error.Code)))...)
		}
		return c.Status(status).JSON(body)
	}
}
