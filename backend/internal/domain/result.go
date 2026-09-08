package domain

import (
	"fmt"
	"math"
	"strconv"
)

// significantDigits is float64's guaranteed decimal round-trip precision
// (DBL_DIG). Anything beyond it is binary representation noise, e.g.
// 0.1 + 0.2 == 0.30000000000000004.
const significantDigits = 15

// Result is a validated, normalised calculation result. It is always finite.
type Result float64

// Float64 returns the underlying value.
func (r Result) Float64() float64 { return float64(r) }

// NewResult validates and normalises a raw computation.
//
// NaN becomes ErrUndefinedResult and ±Inf becomes ErrResultOutOfRange, so the
// individual arithmetic functions only have to raise their semantic errors
// (division by zero, square root of a negative number). Finite values are
// rounded to 15 significant digits and negative zero is normalised to zero.
func NewResult(v float64) (Result, error) {
	switch {
	case math.IsNaN(v):
		return 0, fmt.Errorf("%w: not a number", ErrUndefinedResult)
	case math.IsInf(v, 1):
		return 0, fmt.Errorf("%w: positive infinity", ErrResultOutOfRange)
	case math.IsInf(v, -1):
		return 0, fmt.Errorf("%w: negative infinity", ErrResultOutOfRange)
	}
	return Result(normalise(v)), nil
}

// normalise rounds v to significantDigits decimal digits and maps -0 to 0.
func normalise(v float64) float64 {
	rounded, err := strconv.ParseFloat(strconv.FormatFloat(v, 'g', significantDigits, 64), 64)
	// Rounding the largest representable float64 up to 15 digits overflows on
	// the way back in; in that case the original (finite) value is the best
	// answer available.
	if err != nil || math.IsInf(rounded, 0) {
		rounded = v
	}
	if rounded == 0 {
		return 0
	}
	return rounded
}
