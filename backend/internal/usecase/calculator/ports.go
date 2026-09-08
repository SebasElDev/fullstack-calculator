// Package calculator contains the application rules of the service: it turns a
// requested operation into a validated, normalised result. It depends on the
// domain only — no HTTP, no framework, no configuration.
package calculator

import (
	"context"

	"github.com/SebasElDev/fullstack-calculator/backend/internal/domain"
)

// Calculator is the input port of the application. Delivery mechanisms (the
// HTTP adapter, a CLI, a test double) depend on this interface, never on the
// concrete Service.
type Calculator interface {
	// Calculate performs one operation and returns its normalised result.
	Calculate(ctx context.Context, in Input) (Output, error)
	// Operations lists the supported operations in registry order.
	Operations() []OperationInfo
}

// Input is one calculation request.
type Input struct {
	Operation domain.Operation
	Operands  []float64
}

// Output is the outcome of a successful calculation. The operation and its
// operands are echoed so callers can correlate responses with requests.
type Output struct {
	Operation domain.Operation
	Operands  []float64
	Result    float64
}

// OperationInfo advertises one operation to clients.
type OperationInfo struct {
	Name        domain.Operation
	Symbol      string
	Arity       int
	Description string
}
