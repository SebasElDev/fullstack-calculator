package fiberadapter

import (
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/SebasElDev/fullstack-calculator/backend/internal/adapter/http/dto"
	"github.com/SebasElDev/fullstack-calculator/backend/internal/usecase/calculator"
)

// handlers is the thin translation layer between HTTP and the input port:
// parse, delegate, encode. It contains no arithmetic and no business rules, so
// the handler tests can run against a fake Calculator.
type handlers struct {
	calculator calculator.Calculator
	version    string
}

// health answers the liveness probe.
func (h handlers) health(c fiber.Ctx) error {
	return c.JSON(dto.NewHealthResponse(h.version))
}

// operations advertises the registry, in registry order.
func (h handlers) operations(c fiber.Ctx) error {
	return c.JSON(dto.NewOperationsResponse(h.calculator.Operations()))
}

// calculate performs one operation. Every error — a malformed body as much as a
// division by zero — is returned to the error boundary, which owns the mapping
// to status codes.
func (h handlers) calculate(c fiber.Ctx) error {
	if len(c.Body()) > maxRequestBodyBytes {
		return fiber.NewError(fiber.StatusRequestEntityTooLarge,
			fmt.Sprintf("request body exceeds %d bytes", maxRequestBodyBytes))
	}

	request, err := dto.ParseCalculateRequest(c.Body())
	if err != nil {
		return err
	}

	output, err := h.calculator.Calculate(c.Context(), request.ToInput())
	if err != nil {
		return err
	}

	return c.JSON(dto.NewCalculateResponse(output))
}

// notFound answers unknown API routes with the JSON error envelope instead of
// Fiber's plain-text default or the SPA shell.
func (h handlers) notFound(c fiber.Ctx) error {
	return newNotFoundError(c.Method(), c.Path())
}
