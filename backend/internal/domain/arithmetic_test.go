package domain

import (
	"errors"
	"math"
	"testing"
)

// applyThroughRegistry runs an operation exactly the way the use case does:
// registry lookup, arity validation, Apply, then NewResult.
func applyThroughRegistry(t *testing.T, name Operation, operands ...float64) (float64, error) {
	t.Helper()
	spec, ok := Lookup(name)
	if !ok {
		t.Fatalf("Lookup(%q) = _, false; want the operation to be registered", name)
	}
	if err := spec.Validate(operands); err != nil {
		return 0, err
	}
	raw, err := spec.Apply(operands...)
	if err != nil {
		return 0, err
	}
	result, err := NewResult(raw)
	return result.Float64(), err
}

func TestOperationsSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		op       Operation
		operands []float64
		want     float64
	}{
		{name: "add", op: Add, operands: []float64{2, 3}, want: 5},
		{name: "add negative", op: Add, operands: []float64{-2, -3}, want: -5},
		{name: "add binary noise is normalised", op: Add, operands: []float64{0.1, 0.2}, want: 0.3},
		{name: "subtract", op: Subtract, operands: []float64{10, 4}, want: 6},
		{name: "subtract to negative zero", op: Subtract, operands: []float64{0, 0}, want: 0},
		{name: "multiply", op: Multiply, operands: []float64{6, 7}, want: 42},
		{name: "multiply by zero", op: Multiply, operands: []float64{-3, 0}, want: 0},
		{name: "divide", op: Divide, operands: []float64{9, 3}, want: 3},
		{name: "divide fractional", op: Divide, operands: []float64{1, 8}, want: 0.125},
		{name: "power", op: Power, operands: []float64{2, 10}, want: 1024},
		{name: "power zero exponent", op: Power, operands: []float64{0, 0}, want: 1},
		{name: "power negative exponent", op: Power, operands: []float64{2, -2}, want: 0.25},
		{name: "power negative base integer exponent", op: Power, operands: []float64{-2, 3}, want: -8},
		{name: "modulo positive dividend", op: Modulo, operands: []float64{7, 3}, want: 1},
		{name: "modulo negative dividend keeps its sign", op: Modulo, operands: []float64{-7, 3}, want: -1},
		{name: "modulo negative divisor follows dividend", op: Modulo, operands: []float64{7, -3}, want: 1},
		{name: "modulo both negative", op: Modulo, operands: []float64{-7, -3}, want: -1},
		{name: "modulo fractional", op: Modulo, operands: []float64{5.5, 2}, want: 1.5},
		{name: "negate positive", op: Negate, operands: []float64{5}, want: -5},
		{name: "negate negative", op: Negate, operands: []float64{-5}, want: 5},
		{name: "negate zero normalises minus zero", op: Negate, operands: []float64{0}, want: 0},
		{name: "sqrt", op: Sqrt, operands: []float64{9}, want: 3},
		{name: "sqrt zero", op: Sqrt, operands: []float64{0}, want: 0},
		{name: "sqrt of two", op: Sqrt, operands: []float64{2}, want: 1.4142135623731},
		{name: "square", op: Square, operands: []float64{7}, want: 49},
		{name: "square negative", op: Square, operands: []float64{-7}, want: 49},
		{name: "percent", op: Percent, operands: []float64{50}, want: 0.5},
		{name: "percent negative", op: Percent, operands: []float64{-25}, want: -0.25},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := applyThroughRegistry(t, tc.op, tc.operands...)
			if err != nil {
				t.Fatalf("%s(%v) returned unexpected error: %v", tc.op, tc.operands, err)
			}
			if got != tc.want {
				t.Errorf("%s(%v) = %v; want %v", tc.op, tc.operands, got, tc.want)
			}
			if math.Signbit(got) && got == 0 {
				t.Errorf("%s(%v) returned negative zero; want it normalised to 0", tc.op, tc.operands)
			}
		})
	}
}

func TestOperationsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		op       Operation
		operands []float64
		want     error
	}{
		{name: "divide by zero", op: Divide, operands: []float64{1, 0}, want: ErrDivisionByZero},
		{name: "divide zero by zero", op: Divide, operands: []float64{0, 0}, want: ErrDivisionByZero},
		{name: "modulo by zero", op: Modulo, operands: []float64{1, 0}, want: ErrDivisionByZero},
		{name: "sqrt of a negative number", op: Sqrt, operands: []float64{-1}, want: ErrUndefinedResult},
		{name: "power of a negative base with fractional exponent", op: Power, operands: []float64{-8, 0.5}, want: ErrUndefinedResult},
		{name: "power overflow", op: Power, operands: []float64{10, 400}, want: ErrResultOutOfRange},
		{name: "power of zero to a negative exponent", op: Power, operands: []float64{0, -1}, want: ErrResultOutOfRange},
		{name: "add overflow", op: Add, operands: []float64{math.MaxFloat64, math.MaxFloat64}, want: ErrResultOutOfRange},
		{name: "subtract overflow", op: Subtract, operands: []float64{-math.MaxFloat64, math.MaxFloat64}, want: ErrResultOutOfRange},
		{name: "multiply overflow", op: Multiply, operands: []float64{math.MaxFloat64, 2}, want: ErrResultOutOfRange},
		{name: "square overflow", op: Square, operands: []float64{math.MaxFloat64}, want: ErrResultOutOfRange},
		{name: "binary operation with one operand", op: Add, operands: []float64{1}, want: ErrInvalidArity},
		{name: "binary operation with three operands", op: Multiply, operands: []float64{1, 2, 3}, want: ErrInvalidArity},
		{name: "unary operation with two operands", op: Sqrt, operands: []float64{1, 2}, want: ErrInvalidArity},
		{name: "unary operation without operands", op: Negate, operands: nil, want: ErrInvalidArity},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := applyThroughRegistry(t, tc.op, tc.operands...)
			if !errors.Is(err, tc.want) {
				t.Fatalf("%s(%v) = %v, %v; want error %v", tc.op, tc.operands, got, err, tc.want)
			}
			if got != 0 {
				t.Errorf("%s(%v) returned %v alongside an error; want the zero value", tc.op, tc.operands, got)
			}
		})
	}
}

// TestArithmeticFunctionsGuardTheirArity proves the pure functions are safe to
// call directly, without the use case's validation step.
func TestArithmeticFunctionsGuardTheirArity(t *testing.T) {
	t.Parallel()

	for _, spec := range Registry() {
		t.Run(string(spec.Name), func(t *testing.T) {
			t.Parallel()

			tooMany := make([]float64, spec.Arity+1)
			if _, err := spec.Apply(tooMany...); !errors.Is(err, ErrInvalidArity) {
				t.Errorf("%s.Apply with %d operands returned %v; want ErrInvalidArity", spec.Name, len(tooMany), err)
			}
			if _, err := spec.Apply(); spec.Arity > 0 && !errors.Is(err, ErrInvalidArity) {
				t.Errorf("%s.Apply with no operands returned %v; want ErrInvalidArity", spec.Name, err)
			}
		})
	}
}
