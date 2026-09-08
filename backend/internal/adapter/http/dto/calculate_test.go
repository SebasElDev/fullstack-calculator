package dto_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/SebasElDev/fullstack-calculator/backend/internal/adapter/http/dto"
	"github.com/SebasElDev/fullstack-calculator/backend/internal/domain"
	"github.com/SebasElDev/fullstack-calculator/backend/internal/usecase/calculator"
)

func TestParseCalculateRequestSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want dto.CalculateRequest
	}{
		{
			name: "binary operation",
			body: `{"operation":"add","operands":[2,3]}`,
			want: dto.CalculateRequest{Operation: "add", Operands: []float64{2, 3}},
		},
		{
			name: "unary operation",
			body: `{"operation":"sqrt","operands":[9]}`,
			want: dto.CalculateRequest{Operation: "sqrt", Operands: []float64{9}},
		},
		{
			name: "fields in any order with whitespace",
			body: "{\n  \"operands\": [1.5, -2],\n  \"operation\": \"multiply\"\n}",
			want: dto.CalculateRequest{Operation: "multiply", Operands: []float64{1.5, -2}},
		},
		{
			name: "exponent notation",
			body: `{"operation":"add","operands":[1e2,3E-1]}`,
			want: dto.CalculateRequest{Operation: "add", Operands: []float64{100, 0.3}},
		},
		{
			name: "empty operand list is left to the arity check",
			body: `{"operation":"add","operands":[]}`,
			want: dto.CalculateRequest{Operation: "add", Operands: []float64{}},
		},
		{
			name: "unknown operation name is left to the registry",
			body: `{"operation":"cube","operands":[2]}`,
			want: dto.CalculateRequest{Operation: "cube", Operands: []float64{2}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := dto.ParseCalculateRequest([]byte(tc.body))
			if err != nil {
				t.Fatalf("ParseCalculateRequest(%s) returned unexpected error: %v", tc.body, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParseCalculateRequest(%s) = %+v; want %+v", tc.body, got, tc.want)
			}
		})
	}
}

func TestParseCalculateRequestRejectsInvalidBodies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ""},
		{name: "not JSON", body: "not json at all"},
		{name: "truncated JSON", body: `{"operation":"add","operands":[2,`},
		{name: "JSON array instead of object", body: `[1,2]`},
		{name: "operation is a number", body: `{"operation":1,"operands":[2,3]}`},
		{name: "operands is not an array", body: `{"operation":"add","operands":2}`},
		{name: "operand is a string", body: `{"operation":"add","operands":["2",3]}`},
		{name: "operand is null", body: `{"operation":"add","operands":[null,3]}`},
		{name: "operand out of float64 range", body: `{"operation":"add","operands":[1e400,3]}`},
		{name: "unknown field", body: `{"operation":"add","operands":[2,3],"precision":4}`},
		{name: "missing operation", body: `{"operands":[2,3]}`},
		{name: "missing operands", body: `{"operation":"add"}`},
		{name: "null operation", body: `{"operation":null,"operands":[2,3]}`},
		{name: "null operands", body: `{"operation":"add","operands":null}`},
		{name: "trailing content", body: `{"operation":"add","operands":[2,3]}{"operation":"add"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := dto.ParseCalculateRequest([]byte(tc.body))
			if !errors.Is(err, dto.ErrInvalidRequest) {
				t.Fatalf("ParseCalculateRequest(%s) = %+v, %v; want ErrInvalidRequest", tc.body, got, err)
			}
			if !reflect.DeepEqual(got, dto.CalculateRequest{}) {
				t.Errorf("ParseCalculateRequest(%s) = %+v alongside an error; want the zero value", tc.body, got)
			}
		})
	}
}

func TestCalculateRequestToInput(t *testing.T) {
	t.Parallel()

	req := dto.CalculateRequest{Operation: "divide", Operands: []float64{1, 2}}
	want := calculator.Input{Operation: domain.Divide, Operands: []float64{1, 2}}

	if got := req.ToInput(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ToInput() = %+v; want %+v", got, want)
	}
}

func TestNewCalculateResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		out  calculator.Output
		want string
	}{
		{
			name: "binary result",
			out:  calculator.Output{Operation: domain.Add, Operands: []float64{2, 3}, Result: 5},
			want: `{"operation":"add","operands":[2,3],"result":5}`,
		},
		{
			name: "unary result",
			out:  calculator.Output{Operation: domain.Sqrt, Operands: []float64{9}, Result: 3},
			want: `{"operation":"sqrt","operands":[9],"result":3}`,
		},
		{
			name: "nil operands are encoded as an empty array",
			out:  calculator.Output{Operation: domain.Add, Result: 0},
			want: `{"operation":"add","operands":[],"result":0}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(dto.NewCalculateResponse(tc.out))
			if err != nil {
				t.Fatalf("json.Marshal returned unexpected error: %v", err)
			}
			if string(encoded) != tc.want {
				t.Errorf("NewCalculateResponse(%+v) encoded as %s; want %s", tc.out, encoded, tc.want)
			}
		})
	}
}

func TestNewOperationsResponse(t *testing.T) {
	t.Parallel()

	infos := calculator.NewService().Operations()
	got := dto.NewOperationsResponse(infos)

	if len(got.Operations) != len(infos) {
		t.Fatalf("NewOperationsResponse returned %d operations; want %d", len(got.Operations), len(infos))
	}
	for i, info := range infos {
		want := dto.OperationInfo{
			Name:        string(info.Name),
			Symbol:      info.Symbol,
			Arity:       info.Arity,
			Description: info.Description,
		}
		if got.Operations[i] != want {
			t.Errorf("operation %d = %+v; want %+v", i, got.Operations[i], want)
		}
	}

	encoded, err := json.Marshal(dto.NewOperationsResponse(nil))
	if err != nil {
		t.Fatalf("json.Marshal returned unexpected error: %v", err)
	}
	if want := `{"operations":[]}`; string(encoded) != want {
		t.Errorf("empty registry encoded as %s; want %s", encoded, want)
	}
}

func TestNewHealthResponse(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(dto.NewHealthResponse("1.2.3"))
	if err != nil {
		t.Fatalf("json.Marshal returned unexpected error: %v", err)
	}
	if want := `{"status":"ok","version":"1.2.3"}`; string(encoded) != want {
		t.Fatalf("NewHealthResponse encoded as %s; want %s", encoded, want)
	}
	if dto.NewHealthResponse("").Status != dto.HealthStatusOK {
		t.Fatalf("health status = %q; want %q", dto.NewHealthResponse("").Status, dto.HealthStatusOK)
	}
}
