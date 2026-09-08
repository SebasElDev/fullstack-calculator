package fiberadapter_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/SebasElDev/fullstack-calculator/backend/internal/adapter/http/dto"
	fiberadapter "github.com/SebasElDev/fullstack-calculator/backend/internal/adapter/http/fiber"
	"github.com/SebasElDev/fullstack-calculator/backend/internal/domain"
	"github.com/SebasElDev/fullstack-calculator/backend/internal/usecase/calculator"
)

const testVersion = "test-version"

// fakeCalculator is a stand-in for the use case. The handlers must be happy
// with whatever it returns — including deliberately wrong arithmetic — which is
// what proves the HTTP layer computes nothing itself.
type fakeCalculator struct {
	output     calculator.Output
	err        error
	operations []calculator.OperationInfo
	panic      bool

	calls     int
	lastInput calculator.Input
}

func (f *fakeCalculator) Calculate(_ context.Context, in calculator.Input) (calculator.Output, error) {
	f.calls++
	f.lastInput = in
	if f.panic {
		panic("the use case exploded")
	}
	return f.output, f.err
}

func (f *fakeCalculator) Operations() []calculator.OperationInfo { return f.operations }

// newTestApp builds an app with a discarding logger so test output stays clean.
func newTestApp(t *testing.T, opts fiberadapter.Options) *fiber.App {
	t.Helper()
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if opts.Version == "" {
		opts.Version = testVersion
	}
	return fiberadapter.New(opts)
}

func do(t *testing.T, app *fiber.App, method, target, body string) (*http.Response, []byte) {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != "" {
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	}

	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: app.Test returned unexpected error: %v", method, target, err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })

	payload, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("%s %s: reading the body failed: %v", method, target, err)
	}
	return res, payload
}

func decode[T any](t *testing.T, payload []byte) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatalf("decoding %s failed: %v", payload, err)
	}
	return value
}

func TestHealth(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, fiberadapter.Options{Calculator: &fakeCalculator{}})

	for _, path := range []string{"/health", "/api/v1/health"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			res, payload := do(t, app, http.MethodGet, path, "")
			if res.StatusCode != http.StatusOK {
				t.Fatalf("GET %s = %d; want 200", path, res.StatusCode)
			}
			got := decode[dto.HealthResponse](t, payload)
			want := dto.HealthResponse{Status: dto.HealthStatusOK, Version: testVersion}
			if got != want {
				t.Errorf("GET %s body = %+v; want %+v", path, got, want)
			}
		})
	}
}

func TestOperationsIsServedFromTheUseCase(t *testing.T) {
	t.Parallel()

	fake := &fakeCalculator{operations: []calculator.OperationInfo{
		{Name: "hypotenuse", Symbol: "⊿", Arity: 2, Description: "Only this fake knows it"},
	}}
	app := newTestApp(t, fiberadapter.Options{Calculator: fake})

	res, payload := do(t, app, http.MethodGet, "/api/v1/operations", "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/operations = %d; want 200", res.StatusCode)
	}

	got := decode[dto.OperationsResponse](t, payload)
	want := dto.OperationsResponse{Operations: []dto.OperationInfo{
		{Name: "hypotenuse", Symbol: "⊿", Arity: 2, Description: "Only this fake knows it"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GET /api/v1/operations body = %+v; want %+v", got, want)
	}
}

// TestCalculateReturnsWhateverTheUseCaseSays is the decoupling proof: the fake
// answers 2 + 3 with 42 and the HTTP layer reports 42.
func TestCalculateReturnsWhateverTheUseCaseSays(t *testing.T) {
	t.Parallel()

	fake := &fakeCalculator{output: calculator.Output{
		Operation: domain.Add,
		Operands:  []float64{2, 3},
		Result:    42,
	}}
	app := newTestApp(t, fiberadapter.Options{Calculator: fake})

	res, payload := do(t, app, http.MethodPost, "/api/v1/calculate", `{"operation":"add","operands":[2,3]}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/v1/calculate = %d; want 200. body: %s", res.StatusCode, payload)
	}
	if ct := res.Header.Get(fiber.HeaderContentType); !strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
		t.Errorf("content type = %q; want JSON", ct)
	}

	got := decode[dto.CalculateResponse](t, payload)
	want := dto.CalculateResponse{Operation: "add", Operands: []float64{2, 3}, Result: 42}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("body = %+v; want %+v", got, want)
	}

	if fake.calls != 1 {
		t.Fatalf("the use case was called %d times; want exactly 1", fake.calls)
	}
	wantInput := calculator.Input{Operation: domain.Add, Operands: []float64{2, 3}}
	if !reflect.DeepEqual(fake.lastInput, wantInput) {
		t.Errorf("the handler passed %+v; want %+v", fake.lastInput, wantInput)
	}
}

func TestCalculateMapsUseCaseErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   dto.ErrorCode
	}{
		{name: "unsupported operation", err: fmt.Errorf("%w: %q", domain.ErrUnsupportedOperation, "cube"), wantStatus: http.StatusBadRequest, wantCode: dto.CodeUnsupportedOperation},
		{name: "invalid arity", err: domain.ErrInvalidArity, wantStatus: http.StatusBadRequest, wantCode: dto.CodeInvalidOperands},
		{name: "division by zero", err: domain.ErrDivisionByZero, wantStatus: http.StatusUnprocessableEntity, wantCode: dto.CodeDivisionByZero},
		{name: "undefined result", err: domain.ErrUndefinedResult, wantStatus: http.StatusUnprocessableEntity, wantCode: dto.CodeUndefinedResult},
		{name: "result out of range", err: domain.ErrResultOutOfRange, wantStatus: http.StatusUnprocessableEntity, wantCode: dto.CodeResultOutOfRange},
		{name: "unexpected failure", err: errors.New("something internal"), wantStatus: http.StatusInternalServerError, wantCode: dto.CodeInternalError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			app := newTestApp(t, fiberadapter.Options{Calculator: &fakeCalculator{err: tc.err}})
			res, payload := do(t, app, http.MethodPost, "/api/v1/calculate", `{"operation":"divide","operands":[1,0]}`)

			if res.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d; want %d. body: %s", res.StatusCode, tc.wantStatus, payload)
			}
			got := decode[dto.ErrorResponse](t, payload)
			if got.Error.Code != tc.wantCode {
				t.Errorf("code = %q; want %q", got.Error.Code, tc.wantCode)
			}
			if got.Error.Message == "" {
				t.Error("message is empty")
			}
			if tc.wantCode == dto.CodeInternalError && got.Error.Message != dto.InternalErrorMessage {
				t.Errorf("message = %q; want the generic %q", got.Error.Message, dto.InternalErrorMessage)
			}
		})
	}
}

func TestCalculateRejectsInvalidBodies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "malformed json", body: `{"operation":"add",`},
		{name: "empty body", body: ""},
		{name: "wrong type for operands", body: `{"operation":"add","operands":"2,3"}`},
		{name: "wrong type for operand", body: `{"operation":"add","operands":[true,3]}`},
		{name: "unknown field", body: `{"operation":"add","operands":[2,3],"round":true}`},
		{name: "missing operands", body: `{"operation":"add"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fake := &fakeCalculator{}
			app := newTestApp(t, fiberadapter.Options{Calculator: fake})
			res, payload := do(t, app, http.MethodPost, "/api/v1/calculate", tc.body)

			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d; want 400. body: %s", res.StatusCode, payload)
			}
			if got := decode[dto.ErrorResponse](t, payload); got.Error.Code != dto.CodeInvalidRequest {
				t.Errorf("code = %q; want %q", got.Error.Code, dto.CodeInvalidRequest)
			}
			if fake.calls != 0 {
				t.Errorf("the use case was called %d times for an invalid body; want 0", fake.calls)
			}
		})
	}
}

// TestCalculateRejectsOversizedBodies covers the application-level limit: the
// rejection carries the JSON envelope and never reaches the use case.
func TestCalculateRejectsOversizedBodies(t *testing.T) {
	t.Parallel()

	fake := &fakeCalculator{}
	app := newTestApp(t, fiberadapter.Options{Calculator: fake})

	operands := strings.Repeat("1234567890,", 200)
	body := `{"operation":"add","operands":[` + strings.TrimSuffix(operands, ",") + `]}`

	res, payload := do(t, app, http.MethodPost, "/api/v1/calculate", body)
	if res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d; want 413. body: %s", res.StatusCode, payload)
	}
	got := decode[dto.ErrorResponse](t, payload)
	if got.Error.Code != dto.CodeInvalidRequest {
		t.Errorf("code = %q; want %q", got.Error.Code, dto.CodeInvalidRequest)
	}
	if !strings.Contains(got.Error.Message, "1024 bytes") {
		t.Errorf("message = %q; want it to name the limit", got.Error.Message)
	}
	if fake.calls != 0 {
		t.Errorf("the use case was called %d times for an oversized body; want 0", fake.calls)
	}
}

// syncBuffer is a goroutine-safe bytes.Buffer for capturing logs written by
// the server while the test goroutine reads them.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestTransportLimitIsLoggedWithItsRealStatus runs against a real listener: a
// body over the transport ceiling is rejected by the HTTP server before the
// router sees the request, which the in-memory app.Test transport cannot
// reproduce. The response must still be the JSON envelope, and the log must
// carry the status that was actually sent.
func TestTransportLimitIsLoggedWithItsRealStatus(t *testing.T) {
	t.Parallel()

	logs := &syncBuffer{}
	fake := &fakeCalculator{}
	app := newTestApp(t, fiberadapter.Options{
		Calculator: fake,
		Logger:     slog.New(slog.NewJSONHandler(logs, nil)),
	})

	body := `{"operation":"add","operands":[1,` + strings.Repeat("1", 66*1024) + `]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calculate", strings.NewReader(body))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	res, payload := serveOverMemory(t, app, req)

	if res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d; want 413. body: %s", res.StatusCode, payload)
	}
	if got := decode[dto.ErrorResponse](t, payload); got.Error.Code != dto.CodeInvalidRequest {
		t.Errorf("code = %q; want %q", got.Error.Code, dto.CodeInvalidRequest)
	}
	if fake.calls != 0 {
		t.Errorf("the use case was called %d times; want 0", fake.calls)
	}

	var rejected bool
	for _, line := range strings.Split(strings.TrimSpace(logs.String()), "\n") {
		var entry struct {
			Msg    string `json:"msg"`
			Status int    `json:"status"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("log line is not JSON: %q", line)
		}
		if entry.Msg == "request rejected" && entry.Status == http.StatusRequestEntityTooLarge {
			rejected = true
		}
	}
	if !rejected {
		t.Errorf("no \"request rejected\" log line with status 413. logs:\n%s", logs.String())
	}
}

// serveOverMemory drives the fasthttp server on an in-process connection and
// returns whatever response it wrote, even when serving ended with an error.
//
// It exists for requests the transport itself rejects. fasthttp reports an
// oversized body as an error from ServeConn after it has already written the
// 413, and app.Test surfaces that error instead of the response. A real TCP
// client is no better: the server resets the connection mid-upload, so the
// client can lose the response to that race.
func serveOverMemory(t *testing.T, app *fiber.App, req *http.Request) (*http.Response, []byte) {
	t.Helper()

	conn := &memoryConn{}
	if err := req.Write(&conn.in); err != nil {
		t.Fatalf("serialising the request failed: %v", err)
	}

	app.RebuildTree()
	if err := app.Server().ServeConn(conn); err != nil {
		t.Logf("ServeConn reported %q; the response written before it is what counts", err)
	}

	res, err := http.ReadResponse(bufio.NewReader(&conn.out), req)
	if err != nil {
		t.Fatalf("no parseable response was written: %v", err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })

	payload, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("reading the body failed: %v", err)
	}
	return res, payload
}

// memoryConn is a net.Conn whose peer is a pair of buffers: the server reads
// the raw request from in and writes its response into out.
type memoryConn struct {
	in  bytes.Buffer
	out bytes.Buffer
}

func (c *memoryConn) Read(b []byte) (int, error)  { return c.in.Read(b) }
func (c *memoryConn) Write(b []byte) (int, error) { return c.out.Write(b) }
func (*memoryConn) Close() error                  { return nil }
func (*memoryConn) LocalAddr() net.Addr           { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)} }
func (*memoryConn) RemoteAddr() net.Addr          { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)} }
func (*memoryConn) SetDeadline(time.Time) error   { return nil }
func (*memoryConn) SetReadDeadline(time.Time) error {
	return nil
}
func (*memoryConn) SetWriteDeadline(time.Time) error {
	return nil
}

func TestPanicsBecomeInternalErrors(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, fiberadapter.Options{Calculator: &fakeCalculator{panic: true}})
	res, payload := do(t, app, http.MethodPost, "/api/v1/calculate", `{"operation":"add","operands":[2,3]}`)

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d; want 500. body: %s", res.StatusCode, payload)
	}
	got := decode[dto.ErrorResponse](t, payload)
	if got.Error.Code != dto.CodeInternalError {
		t.Errorf("code = %q; want %q", got.Error.Code, dto.CodeInternalError)
	}
	if got.Error.Message != dto.InternalErrorMessage {
		t.Errorf("message = %q; want the generic %q", got.Error.Message, dto.InternalErrorMessage)
	}
}

func TestUnknownRoutesAnswerWithJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		target string
	}{
		{name: "unknown api path", method: http.MethodGet, target: "/api/v1/nope"},
		{name: "unknown api version", method: http.MethodPost, target: "/api/v2/calculate"},
		{name: "wrong method on calculate", method: http.MethodGet, target: "/api/v1/calculate"},
		{name: "unknown root path", method: http.MethodGet, target: "/nope"},
	}

	app := newTestApp(t, fiberadapter.Options{Calculator: &fakeCalculator{}})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, payload := do(t, app, tc.method, tc.target, "")
			if res.StatusCode != http.StatusNotFound {
				t.Fatalf("%s %s = %d; want 404. body: %s", tc.method, tc.target, res.StatusCode, payload)
			}
			got := decode[dto.ErrorResponse](t, payload)
			if got.Error.Code != dto.CodeNotFound {
				t.Errorf("code = %q; want %q", got.Error.Code, dto.CodeNotFound)
			}
			if got.Error.Message == "" {
				t.Error("message is empty")
			}
		})
	}
}

func TestRequestIDHeaderIsSet(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, fiberadapter.Options{Calculator: &fakeCalculator{}})
	res, _ := do(t, app, http.MethodGet, "/health", "")

	if id := res.Header.Get(fiber.HeaderXRequestID); id == "" {
		t.Fatal("the X-Request-ID response header is empty")
	}
}

func TestCORSIsOptional(t *testing.T) {
	t.Parallel()

	const origin = "https://calculator.example"

	tests := []struct {
		name    string
		origins []string
		want    string
	}{
		{name: "disabled by default", origins: nil, want: ""},
		{name: "enabled when configured", origins: []string{origin}, want: origin},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			app := newTestApp(t, fiberadapter.Options{Calculator: &fakeCalculator{}, CORSOrigins: tc.origins})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
			req.Header.Set(fiber.HeaderOrigin, origin)
			res, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test returned unexpected error: %v", err)
			}
			t.Cleanup(func() { _ = res.Body.Close() })

			if got := res.Header.Get(fiber.HeaderAccessControlAllowOrigin); got != tc.want {
				t.Fatalf("Access-Control-Allow-Origin = %q; want %q", got, tc.want)
			}
		})
	}
}

func TestStaticHostingServesTheSPA(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	const shell = "<!doctype html><title>calculator</title>"
	const asset = "console.log('calculator');"

	writeFile(t, filepath.Join(dir, "index.html"), shell)
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o750); err != nil {
		t.Fatalf("creating the assets directory failed: %v", err)
	}
	writeFile(t, filepath.Join(dir, "assets", "app.js"), asset)

	app := newTestApp(t, fiberadapter.Options{Calculator: &fakeCalculator{}, StaticDir: dir})

	tests := []struct {
		name   string
		target string
		want   string
	}{
		{name: "root serves the shell", target: "/", want: shell},
		{name: "asset is served as it is", target: "/assets/app.js", want: asset},
		{name: "client-side route falls back to the shell", target: "/history/42", want: shell},
		{name: "unknown file falls back to the shell", target: "/favicon.svg", want: shell},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, payload := do(t, app, http.MethodGet, tc.target, "")
			if res.StatusCode != http.StatusOK {
				t.Fatalf("GET %s = %d; want 200", tc.target, res.StatusCode)
			}
			if string(payload) != tc.want {
				t.Errorf("GET %s body = %q; want %q", tc.target, payload, tc.want)
			}
		})
	}

	t.Run("api routes are never shadowed by the shell", func(t *testing.T) {
		t.Parallel()

		res, payload := do(t, app, http.MethodGet, "/api/v1/nope", "")
		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("GET /api/v1/nope = %d; want 404", res.StatusCode)
		}
		if got := decode[dto.ErrorResponse](t, payload); got.Error.Code != dto.CodeNotFound {
			t.Errorf("code = %q; want %q", got.Error.Code, dto.CodeNotFound)
		}
	})

	t.Run("the api still answers", func(t *testing.T) {
		t.Parallel()

		res, _ := do(t, app, http.MethodGet, "/api/v1/health", "")
		if res.StatusCode != http.StatusOK {
			t.Fatalf("GET /api/v1/health = %d; want 200", res.StatusCode)
		}
	})
}

func TestStaticHostingWithoutAShell(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "robots.txt"), "User-agent: *\n")

	app := newTestApp(t, fiberadapter.Options{Calculator: &fakeCalculator{}, StaticDir: dir})

	res, payload := do(t, app, http.MethodGet, "/history/42", "")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("GET /history/42 = %d; want 404. body: %s", res.StatusCode, payload)
	}
	if got := decode[dto.ErrorResponse](t, payload); got.Error.Code != dto.CodeNotFound {
		t.Errorf("code = %q; want %q", got.Error.Code, dto.CodeNotFound)
	}

	res, payload = do(t, app, http.MethodGet, "/robots.txt", "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /robots.txt = %d; want 200", res.StatusCode)
	}
	if want := "User-agent: *\n"; string(payload) != want {
		t.Errorf("GET /robots.txt body = %q; want %q", payload, want)
	}
}

// TestNewUsesTheDefaultLogger covers the Options.Logger fallback.
func TestNewUsesTheDefaultLogger(t *testing.T) {
	t.Parallel()

	app := fiberadapter.New(fiberadapter.Options{Calculator: &fakeCalculator{}, Version: testVersion})
	res, _ := do(t, app, http.MethodGet, "/health", "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /health = %d; want 200", res.StatusCode)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s failed: %v", path, err)
	}
}
