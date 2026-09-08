package domain

import (
	"errors"
	"math"
	"testing"
)

func TestNewResultNormalises(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value float64
		want  float64
	}{
		{name: "whole number", value: 5, want: 5},
		{name: "zero", value: 0, want: 0},
		{name: "negative zero becomes zero", value: math.Copysign(0, -1), want: 0},
		{name: "binary noise from 0.1+0.2", value: 0.1 + 0.2, want: 0.3},
		{name: "binary noise from 0.3-0.1", value: 0.3 - 0.1, want: 0.2},
		{name: "recurring decimal", value: 1.0 / 3.0, want: 0.333333333333333},
		{name: "already short", value: 1.25, want: 1.25},
		{name: "negative value", value: -2.675, want: -2.675},
		{name: "smallest positive value survives", value: math.SmallestNonzeroFloat64, want: math.SmallestNonzeroFloat64},
		{name: "largest value survives rounding overflow", value: math.MaxFloat64, want: math.MaxFloat64},
		{name: "large exponent", value: 1e300, want: 1e300},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewResult(tc.value)
			if err != nil {
				t.Fatalf("NewResult(%v) returned unexpected error: %v", tc.value, err)
			}
			if got.Float64() != tc.want {
				t.Errorf("NewResult(%v) = %v; want %v", tc.value, got.Float64(), tc.want)
			}
			if math.Signbit(got.Float64()) && got.Float64() == 0 {
				t.Errorf("NewResult(%v) returned negative zero; want 0", tc.value)
			}
		})
	}
}

func TestNewResultRejectsNonFiniteValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value float64
		want  error
	}{
		{name: "NaN", value: math.NaN(), want: ErrUndefinedResult},
		{name: "positive infinity", value: math.Inf(1), want: ErrResultOutOfRange},
		{name: "negative infinity", value: math.Inf(-1), want: ErrResultOutOfRange},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewResult(tc.value)
			if !errors.Is(err, tc.want) {
				t.Fatalf("NewResult(%v) = %v, %v; want error %v", tc.value, got, err, tc.want)
			}
			if got != 0 {
				t.Errorf("NewResult(%v) = %v alongside an error; want the zero value", tc.value, got)
			}
		})
	}
}

func TestResultRoundTripsThroughFifteenSignificantDigits(t *testing.T) {
	t.Parallel()

	// 15 digits are kept, the 16th is dropped as binary noise.
	got, err := NewResult(1.234567890123456)
	if err != nil {
		t.Fatalf("NewResult returned unexpected error: %v", err)
	}
	if want := 1.23456789012346; got.Float64() != want {
		t.Fatalf("NewResult(1.234567890123456) = %v; want %v", got.Float64(), want)
	}
}
