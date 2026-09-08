package domain

import (
	"errors"
	"testing"
)

func TestRegistryMatchesTheContract(t *testing.T) {
	t.Parallel()

	// Order and metadata mirror the OperationName enum of api/openapi.yaml.
	want := []struct {
		name   Operation
		symbol string
		arity  int
	}{
		{Add, "+", ArityBinary},
		{Subtract, "−", ArityBinary},
		{Multiply, "×", ArityBinary},
		{Divide, "÷", ArityBinary},
		{Power, "xʸ", ArityBinary},
		{Modulo, "mod", ArityBinary},
		{Negate, "±", ArityUnary},
		{Sqrt, "√", ArityUnary},
		{Square, "x²", ArityUnary},
		{Percent, "%", ArityUnary},
	}

	specs := Registry()
	if len(specs) != len(want) {
		t.Fatalf("Registry() returned %d operations; want %d", len(specs), len(want))
	}

	for i, w := range want {
		spec := specs[i]
		switch {
		case spec.Name != w.name:
			t.Errorf("Registry()[%d].Name = %q; want %q", i, spec.Name, w.name)
		case spec.Symbol != w.symbol:
			t.Errorf("%s: symbol = %q; want %q", spec.Name, spec.Symbol, w.symbol)
		case spec.Arity != w.arity:
			t.Errorf("%s: arity = %d; want %d", spec.Name, spec.Arity, w.arity)
		case spec.Description == "":
			t.Errorf("%s: description is empty", spec.Name)
		case spec.Apply == nil:
			t.Errorf("%s: Apply is nil", spec.Name)
		}
	}
}

func TestRegistryReturnsACopy(t *testing.T) {
	t.Parallel()

	specs := Registry()
	specs[0] = Spec{Name: "tampered"}

	if again := Registry(); again[0].Name != Add {
		t.Fatalf("Registry()[0].Name = %q after mutating a previous copy; want %q", again[0].Name, Add)
	}
}

func TestLookup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		op   Operation
		want bool
	}{
		{name: "known operation", op: Divide, want: true},
		{name: "unknown operation", op: "cube", want: false},
		{name: "empty operation", op: "", want: false},
		{name: "wrong case", op: "ADD", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec, ok := Lookup(tc.op)
			if ok != tc.want {
				t.Fatalf("Lookup(%q) = _, %t; want %t", tc.op, ok, tc.want)
			}
			if ok && spec.Name != tc.op {
				t.Errorf("Lookup(%q).Name = %q; want %q", tc.op, spec.Name, tc.op)
			}
		})
	}
}

func TestLookupCoversEveryRegisteredOperation(t *testing.T) {
	t.Parallel()

	for _, spec := range Registry() {
		if _, ok := Lookup(spec.Name); !ok {
			t.Errorf("Lookup(%q) reported the registered operation as unknown", spec.Name)
		}
	}
}

func TestSpecValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		spec     Spec
		operands []float64
		wantErr  bool
	}{
		{name: "binary with two operands", spec: mustLookup(t, Add), operands: []float64{1, 2}},
		{name: "unary with one operand", spec: mustLookup(t, Sqrt), operands: []float64{1}},
		{name: "binary with one operand", spec: mustLookup(t, Add), operands: []float64{1}, wantErr: true},
		{name: "unary with two operands", spec: mustLookup(t, Sqrt), operands: []float64{1, 2}, wantErr: true},
		{name: "no operands at all", spec: mustLookup(t, Add), operands: nil, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.spec.Validate(tc.operands)
			if tc.wantErr && !errors.Is(err, ErrInvalidArity) {
				t.Fatalf("Validate(%v) = %v; want ErrInvalidArity", tc.operands, err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("Validate(%v) = %v; want nil", tc.operands, err)
			}
		})
	}
}

func TestSpecValidateMessageNamesTheOperation(t *testing.T) {
	t.Parallel()

	err := mustLookup(t, Add).Validate([]float64{1})
	const want = `invalid number of operands: operation "add" expects 2 operands, got 1`
	if err == nil || err.Error() != want {
		t.Fatalf("Validate([1]) = %v; want %q", err, want)
	}
}

func mustLookup(t *testing.T, name Operation) Spec {
	t.Helper()
	spec, ok := Lookup(name)
	if !ok {
		t.Fatalf("Lookup(%q) = _, false; want the operation to be registered", name)
	}
	return spec
}
