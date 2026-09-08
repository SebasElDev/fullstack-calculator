// Package domain holds the entities of the calculator: the operation registry,
// the pure arithmetic functions and the result value object. It is the
// innermost layer of the architecture and depends on the standard library only.
package domain

import "fmt"

// Operation is the canonical identifier of an arithmetic operation. The values
// mirror the OperationName enum of api/openapi.yaml.
type Operation string

// The supported operations, in keypad order.
const (
	Add      Operation = "add"
	Subtract Operation = "subtract"
	Multiply Operation = "multiply"
	Divide   Operation = "divide"
	Power    Operation = "power"
	Modulo   Operation = "modulo"
	Negate   Operation = "negate"
	Sqrt     Operation = "sqrt"
	Square   Operation = "square"
	Percent  Operation = "percent"
)

// Arities supported by the registry.
const (
	ArityUnary  = 1
	ArityBinary = 2
)

// Spec describes one operation: how it is advertised and how it is computed.
// Adding an operation to the calculator means adding one pure function and one
// Spec entry to the registry — no other layer changes.
type Spec struct {
	Name        Operation
	Symbol      string
	Arity       int
	Description string
	Apply       func(operands ...float64) (float64, error)
}

// Validate reports whether the operand count matches the operation's arity.
func (s Spec) Validate(operands []float64) error {
	return requireArity(s.Name, operands, s.Arity)
}

// requireArity is the single arity guard of the domain: it is used both by
// Spec.Validate (called by the use case) and by the arithmetic functions
// themselves, so that they are safe to call directly.
func requireArity(name Operation, operands []float64, want int) error {
	if len(operands) == want {
		return nil
	}
	return fmt.Errorf("%w: operation %q expects %d operands, got %d", ErrInvalidArity, name, want, len(operands))
}

// registry is the ordered source of truth for the operations this service
// supports. The order matches the OperationName enum in api/openapi.yaml and
// is the order clients receive from GET /api/v1/operations.
var registry = []Spec{
	{Name: Add, Symbol: "+", Arity: ArityBinary, Description: "Sum of two operands", Apply: add},
	{Name: Subtract, Symbol: "−", Arity: ArityBinary, Description: "Difference of two operands", Apply: subtract},
	{Name: Multiply, Symbol: "×", Arity: ArityBinary, Description: "Product of two operands", Apply: multiply},
	{Name: Divide, Symbol: "÷", Arity: ArityBinary, Description: "Quotient of two operands", Apply: divide},
	{Name: Power, Symbol: "xʸ", Arity: ArityBinary, Description: "First operand raised to the power of the second", Apply: power},
	{Name: Modulo, Symbol: "mod", Arity: ArityBinary, Description: "Remainder of the division, sign of the dividend", Apply: modulo},
	{Name: Negate, Symbol: "±", Arity: ArityUnary, Description: "Additive inverse of the operand", Apply: negate},
	{Name: Sqrt, Symbol: "√", Arity: ArityUnary, Description: "Principal square root", Apply: sqrt},
	{Name: Square, Symbol: "x²", Arity: ArityUnary, Description: "Operand multiplied by itself", Apply: square},
	{Name: Percent, Symbol: "%", Arity: ArityUnary, Description: "Operand divided by one hundred", Apply: percent},
}

// index allows constant-time lookups without exposing the registry slice.
var index = func() map[Operation]Spec {
	m := make(map[Operation]Spec, len(registry))
	for _, spec := range registry {
		m[spec.Name] = spec
	}
	return m
}()

// Registry returns every supported operation in keypad order. The returned
// slice is a copy: callers cannot mutate the registry.
func Registry() []Spec {
	specs := make([]Spec, len(registry))
	copy(specs, registry)
	return specs
}

// Lookup returns the Spec of an operation, or false when it is unknown.
func Lookup(name Operation) (Spec, bool) {
	spec, ok := index[name]
	return spec, ok
}
