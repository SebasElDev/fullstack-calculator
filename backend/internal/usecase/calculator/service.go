package calculator

import (
	"context"
	"fmt"

	"github.com/SebasElDev/fullstack-calculator/backend/internal/domain"
)

// Service is the default implementation of Calculator. It is stateless and
// safe for concurrent use.
type Service struct{}

// compile-time proof that Service satisfies the input port.
var _ Calculator = (*Service)(nil)

// NewService returns a ready-to-use Calculator.
func NewService() *Service { return &Service{} }

// Calculate resolves the operation in the domain registry, validates the
// operand count, applies the pure function and normalises the result.
//
// Errors wrap the domain sentinels, so callers can classify them with
// errors.Is without knowing this package.
func (s *Service) Calculate(ctx context.Context, in Input) (Output, error) {
	if err := ctx.Err(); err != nil {
		return Output{}, fmt.Errorf("calculate %q: %w", in.Operation, err)
	}

	spec, ok := domain.Lookup(in.Operation)
	if !ok {
		return Output{}, fmt.Errorf("%w: %q", domain.ErrUnsupportedOperation, in.Operation)
	}
	if err := spec.Validate(in.Operands); err != nil {
		return Output{}, err
	}

	raw, err := spec.Apply(in.Operands...)
	if err != nil {
		return Output{}, err
	}

	result, err := domain.NewResult(raw)
	if err != nil {
		return Output{}, err
	}

	return Output{
		Operation: spec.Name,
		Operands:  in.Operands,
		Result:    result.Float64(),
	}, nil
}

// Operations exposes the domain registry in its canonical order.
func (s *Service) Operations() []OperationInfo {
	specs := domain.Registry()
	infos := make([]OperationInfo, 0, len(specs))
	for _, spec := range specs {
		infos = append(infos, OperationInfo{
			Name:        spec.Name,
			Symbol:      spec.Symbol,
			Arity:       spec.Arity,
			Description: spec.Description,
		})
	}
	return infos
}
