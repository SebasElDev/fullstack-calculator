package domain

import "errors"

// Sentinel errors returned by the entities layer. Callers must compare with
// errors.Is; the adapter layer translates them into HTTP status codes and
// machine-readable error codes (see internal/adapter/http/dto).
//
// Every error returned by this package wraps exactly one of these values, so
// the translation table stays exhaustive.
var (
	// ErrUnsupportedOperation is returned when an operation name is not in the registry.
	ErrUnsupportedOperation = errors.New("unsupported operation")
	// ErrInvalidArity is returned when the number of operands does not match the operation's arity.
	ErrInvalidArity = errors.New("invalid number of operands")
	// ErrDivisionByZero is returned by divide and modulo when the divisor is zero.
	ErrDivisionByZero = errors.New("division by zero is undefined")
	// ErrUndefinedResult is returned when the mathematics has no real answer (NaN).
	ErrUndefinedResult = errors.New("result is undefined")
	// ErrResultOutOfRange is returned when a result overflows the float64 range (±Inf).
	ErrResultOutOfRange = errors.New("result is out of range")
)
