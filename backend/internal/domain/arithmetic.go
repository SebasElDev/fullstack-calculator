package domain

import (
	"fmt"
	"math"
)

// The arithmetic functions are unexported because the exported names of this
// package are already taken by the Operation constants (domain.Add is the
// operation identifier "add"). The registry is the public entry point: callers
// obtain a Spec through Lookup and invoke Spec.Apply.
//
// Every function is pure and never panics: the arity is re-checked here so the
// functions are safe to call directly. Only semantic errors are raised
// (division by zero, undefined result); NaN and infinities produced by
// overflow are caught uniformly by NewResult.

// percentDivisor converts a value into a percentage of itself.
const percentDivisor = 100

func add(operands ...float64) (float64, error) {
	if err := requireArity(Add, operands, ArityBinary); err != nil {
		return 0, err
	}
	return operands[0] + operands[1], nil
}

func subtract(operands ...float64) (float64, error) {
	if err := requireArity(Subtract, operands, ArityBinary); err != nil {
		return 0, err
	}
	return operands[0] - operands[1], nil
}

func multiply(operands ...float64) (float64, error) {
	if err := requireArity(Multiply, operands, ArityBinary); err != nil {
		return 0, err
	}
	return operands[0] * operands[1], nil
}

func divide(operands ...float64) (float64, error) {
	if err := requireArity(Divide, operands, ArityBinary); err != nil {
		return 0, err
	}
	if operands[1] == 0 {
		return 0, fmt.Errorf("%w: cannot divide %v by zero", ErrDivisionByZero, operands[0])
	}
	return operands[0] / operands[1], nil
}

func power(operands ...float64) (float64, error) {
	if err := requireArity(Power, operands, ArityBinary); err != nil {
		return 0, err
	}
	return math.Pow(operands[0], operands[1]), nil
}

func modulo(operands ...float64) (float64, error) {
	if err := requireArity(Modulo, operands, ArityBinary); err != nil {
		return 0, err
	}
	if operands[1] == 0 {
		return 0, fmt.Errorf("%w: cannot take %v modulo zero", ErrDivisionByZero, operands[0])
	}
	// math.Mod takes the sign of the dividend, like C's fmod.
	return math.Mod(operands[0], operands[1]), nil
}

func negate(operands ...float64) (float64, error) {
	if err := requireArity(Negate, operands, ArityUnary); err != nil {
		return 0, err
	}
	return -operands[0], nil
}

func sqrt(operands ...float64) (float64, error) {
	if err := requireArity(Sqrt, operands, ArityUnary); err != nil {
		return 0, err
	}
	if operands[0] < 0 {
		return 0, fmt.Errorf("%w: square root of the negative number %v", ErrUndefinedResult, operands[0])
	}
	return math.Sqrt(operands[0]), nil
}

func square(operands ...float64) (float64, error) {
	if err := requireArity(Square, operands, ArityUnary); err != nil {
		return 0, err
	}
	return operands[0] * operands[0], nil
}

func percent(operands ...float64) (float64, error) {
	if err := requireArity(Percent, operands, ArityUnary); err != nil {
		return 0, err
	}
	return operands[0] / percentDivisor, nil
}
