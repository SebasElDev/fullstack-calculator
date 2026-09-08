package calculator_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/SebasElDev/fullstack-calculator/backend/internal/domain"
	"github.com/SebasElDev/fullstack-calculator/backend/internal/usecase/calculator"
)

func TestServiceCalculateSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   calculator.Input
		want float64
	}{
		{name: "binary", in: calculator.Input{Operation: domain.Add, Operands: []float64{2, 3}}, want: 5},
		{name: "unary", in: calculator.Input{Operation: domain.Sqrt, Operands: []float64{9}}, want: 3},
		{name: "result is normalised", in: calculator.Input{Operation: domain.Add, Operands: []float64{0.1, 0.2}}, want: 0.3},
		{name: "negative zero is normalised", in: calculator.Input{Operation: domain.Negate, Operands: []float64{0}}, want: 0},
		{name: "modulo keeps the sign of the dividend", in: calculator.Input{Operation: domain.Modulo, Operands: []float64{-7, 3}}, want: -1},
	}

	service := calculator.NewService()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			out, err := service.Calculate(t.Context(), tc.in)
			if err != nil {
				t.Fatalf("Calculate(%+v) returned unexpected error: %v", tc.in, err)
			}
			if out.Result != tc.want {
				t.Errorf("Calculate(%+v).Result = %v; want %v", tc.in, out.Result, tc.want)
			}
			if out.Operation != tc.in.Operation {
				t.Errorf("Calculate echoed operation %q; want %q", out.Operation, tc.in.Operation)
			}
			if !reflect.DeepEqual(out.Operands, tc.in.Operands) {
				t.Errorf("Calculate echoed operands %v; want %v", out.Operands, tc.in.Operands)
			}
		})
	}
}

func TestServiceCalculateError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   calculator.Input
		want error
	}{
		{name: "unknown operation", in: calculator.Input{Operation: "cube", Operands: []float64{2}}, want: domain.ErrUnsupportedOperation},
		{name: "empty operation", in: calculator.Input{Operation: "", Operands: []float64{2}}, want: domain.ErrUnsupportedOperation},
		{name: "too few operands", in: calculator.Input{Operation: domain.Add, Operands: []float64{2}}, want: domain.ErrInvalidArity},
		{name: "too many operands", in: calculator.Input{Operation: domain.Sqrt, Operands: []float64{2, 3}}, want: domain.ErrInvalidArity},
		{name: "no operands", in: calculator.Input{Operation: domain.Add}, want: domain.ErrInvalidArity},
		{name: "division by zero propagates", in: calculator.Input{Operation: domain.Divide, Operands: []float64{1, 0}}, want: domain.ErrDivisionByZero},
		{name: "undefined result propagates", in: calculator.Input{Operation: domain.Sqrt, Operands: []float64{-1}}, want: domain.ErrUndefinedResult},
		{name: "overflow propagates", in: calculator.Input{Operation: domain.Multiply, Operands: []float64{1e308, 10}}, want: domain.ErrResultOutOfRange},
	}

	service := calculator.NewService()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			out, err := service.Calculate(t.Context(), tc.in)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Calculate(%+v) = %+v, %v; want error %v", tc.in, out, err, tc.want)
			}
			if !reflect.DeepEqual(out, calculator.Output{}) {
				t.Errorf("Calculate returned %+v alongside an error; want the zero value", out)
			}
		})
	}
}

func TestServiceCalculateHonoursContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := calculator.NewService().Calculate(ctx, calculator.Input{Operation: domain.Add, Operands: []float64{1, 2}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Calculate with a cancelled context = %v; want context.Canceled", err)
	}
}

func TestServiceOperationsMirrorsTheRegistry(t *testing.T) {
	t.Parallel()

	specs := domain.Registry()
	infos := calculator.NewService().Operations()

	if len(infos) != len(specs) {
		t.Fatalf("Operations() returned %d entries; want %d", len(infos), len(specs))
	}
	for i, spec := range specs {
		want := calculator.OperationInfo{
			Name:        spec.Name,
			Symbol:      spec.Symbol,
			Arity:       spec.Arity,
			Description: spec.Description,
		}
		if infos[i] != want {
			t.Errorf("Operations()[%d] = %+v; want %+v", i, infos[i], want)
		}
	}
}

func TestServiceOperationsAreCallable(t *testing.T) {
	t.Parallel()

	service := calculator.NewService()
	for _, info := range service.Operations() {
		operands := make([]float64, info.Arity)
		for i := range operands {
			operands[i] = 2
		}
		if _, err := service.Calculate(t.Context(), calculator.Input{Operation: info.Name, Operands: operands}); err != nil {
			t.Errorf("advertised operation %q failed on %v: %v", info.Name, operands, err)
		}
	}
}
