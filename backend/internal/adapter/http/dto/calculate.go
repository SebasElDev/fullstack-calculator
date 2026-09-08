// Package dto holds the wire representation of the API — the request and
// response shapes of api/openapi.yaml and the translation of domain errors into
// HTTP status codes. It is deliberately free of any web framework: swapping
// Fiber for another router only rewrites the sibling package.
package dto

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/SebasElDev/fullstack-calculator/backend/internal/domain"
	"github.com/SebasElDev/fullstack-calculator/backend/internal/usecase/calculator"
)

// HealthStatusOK is the only value the health endpoint reports; the OpenAPI
// schema pins it with `const: ok`.
const HealthStatusOK = "ok"

// CalculateRequest is the body of POST /api/v1/calculate.
type CalculateRequest struct {
	Operation string    `json:"operation"`
	Operands  []float64 `json:"operands"`
}

// CalculateResponse is the 200 body of POST /api/v1/calculate.
type CalculateResponse struct {
	Operation string    `json:"operation"`
	Operands  []float64 `json:"operands"`
	Result    float64   `json:"result"`
}

// OperationInfo advertises one operation.
type OperationInfo struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	Arity       int    `json:"arity"`
	Description string `json:"description"`
}

// OperationsResponse is the body of GET /api/v1/operations.
type OperationsResponse struct {
	Operations []OperationInfo `json:"operations"`
}

// HealthResponse is the body of GET /health and GET /api/v1/health.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// ParseCalculateRequest decodes and validates a request body.
//
// Unknown JSON fields are REJECTED (the CalculateRequest schema declares
// `additionalProperties: false`), as are malformed JSON, wrong field types,
// trailing content after the JSON value and missing required fields. Every one
// of those failures wraps ErrInvalidRequest and is reported as 400
// INVALID_REQUEST.
func ParseCalculateRequest(body []byte) (CalculateRequest, error) {
	// Presence is checked with pointers, so that `{"operation":"add"}` is a
	// malformed request rather than an arity error, and so that a null operand
	// (which encoding/json would silently decode as 0) is rejected.
	var raw struct {
		Operation *string     `json:"operation"`
		Operands  *[]*float64 `json:"operands"`
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return CalculateRequest{}, fmt.Errorf("%w: %s", ErrInvalidRequest, err)
	}
	if decoder.More() {
		return CalculateRequest{}, fmt.Errorf("%w: unexpected content after the JSON body", ErrInvalidRequest)
	}
	if raw.Operation == nil {
		return CalculateRequest{}, fmt.Errorf(`%w: field "operation" is required`, ErrInvalidRequest)
	}
	if raw.Operands == nil {
		return CalculateRequest{}, fmt.Errorf(`%w: field "operands" is required`, ErrInvalidRequest)
	}

	operands := make([]float64, 0, len(*raw.Operands))
	for i, operand := range *raw.Operands {
		if operand == nil {
			return CalculateRequest{}, fmt.Errorf("%w: operand %d is null, expected a finite number", ErrInvalidRequest, i)
		}
		operands = append(operands, *operand)
	}

	return CalculateRequest{Operation: *raw.Operation, Operands: operands}, nil
}

// ToInput converts the wire request into the use case's input.
func (r CalculateRequest) ToInput() calculator.Input {
	return calculator.Input{
		Operation: domain.Operation(r.Operation),
		Operands:  r.Operands,
	}
}

// NewCalculateResponse converts a use case result into its wire form.
func NewCalculateResponse(out calculator.Output) CalculateResponse {
	operands := out.Operands
	if operands == nil {
		// The contract types `operands` as an array; never emit JSON null.
		operands = []float64{}
	}
	return CalculateResponse{
		Operation: string(out.Operation),
		Operands:  operands,
		Result:    out.Result,
	}
}

// NewOperationsResponse converts the advertised operations into their wire form,
// preserving registry order.
func NewOperationsResponse(infos []calculator.OperationInfo) OperationsResponse {
	operations := make([]OperationInfo, 0, len(infos))
	for _, info := range infos {
		operations = append(operations, OperationInfo{
			Name:        string(info.Name),
			Symbol:      info.Symbol,
			Arity:       info.Arity,
			Description: info.Description,
		})
	}
	return OperationsResponse{Operations: operations}
}

// NewHealthResponse builds the liveness payload for a build version.
func NewHealthResponse(version string) HealthResponse {
	return HealthResponse{Status: HealthStatusOK, Version: version}
}
