package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/SebasElDev/fullstack-calculator/backend/internal/adapter/http/dto"
	"github.com/SebasElDev/fullstack-calculator/backend/internal/infrastructure/config"
)

// The signal-driven tests share the process, so this file does not use
// t.Parallel: a SIGTERM raised by one test would reach the others.

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// freePort reserves a port and releases it again, so the server under test can
// bind it.
func freePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserving a port failed: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("releasing the reserved port failed: %v", err)
	}
	return port
}

func waitForHealth(t *testing.T, port int) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := probeHealth(port); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("the server on port %d never became healthy", port)
}

// TestNewAppWiresTheRealCalculator proves the composition root connects the use
// case to the HTTP adapter: the arithmetic in the response comes from the
// domain, not from a fake.
func TestNewAppWiresTheRealCalculator(t *testing.T) {
	app := newApp(config.Config{Port: 8080}, discardLogger())

	res, err := app.Test(newCalculateRequest(`{"operation":"add","operands":[0.1,0.2]}`))
	if err != nil {
		t.Fatalf("app.Test returned unexpected error: %v", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d; want 200", res.StatusCode)
	}

	var body dto.CalculateResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decoding the response failed: %v", err)
	}
	if body.Result != 0.3 {
		t.Errorf("0.1 + 0.2 = %v; want the normalised 0.3", body.Result)
	}
}

func TestNewAppReportsTheBuildVersion(t *testing.T) {
	app := newApp(config.Config{Port: 8080}, discardLogger())

	res, err := app.Test(httptest.NewRequest(http.MethodGet, "/health", nil))
	if err != nil {
		t.Fatalf("app.Test returned unexpected error: %v", err)
	}
	defer func() { _ = res.Body.Close() }()

	var body dto.HealthResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decoding the response failed: %v", err)
	}
	if body.Version != version {
		t.Fatalf("version = %q; want %q", body.Version, version)
	}
	if body.Status != dto.HealthStatusOK {
		t.Fatalf("status = %q; want %q", body.Status, dto.HealthStatusOK)
	}
}

func TestProbeHealth(t *testing.T) {
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthy.Close()

	unhealthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer unhealthy.Close()

	if err := probeHealth(portOf(t, healthy.URL)); err != nil {
		t.Errorf("probeHealth against a healthy server = %v; want nil", err)
	}
	if err := probeHealth(portOf(t, unhealthy.URL)); err == nil {
		t.Error("probeHealth against an unhealthy server = nil; want an error")
	}
	if err := probeHealth(freePort(t)); err == nil {
		t.Error("probeHealth against a closed port = nil; want an error")
	}
}

func TestRunServesAndShutsDownOnSIGTERM(t *testing.T) {
	port := freePort(t)
	cfg := config.Config{Port: port, ShutdownTimeout: 5 * time.Second}

	done := make(chan error, 1)
	go func() { done <- run(cfg, discardLogger()) }()

	waitForHealth(t, port)

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("raising SIGTERM failed: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned %v; want a clean shutdown", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run did not return after SIGTERM")
	}

	if err := probeHealth(port); err == nil {
		t.Error("the server still answers after the shutdown; want a closed port")
	}
}

func TestRunReportsAListenFailure(t *testing.T) {
	// config.Load can never produce this port, but run must still report a
	// listen failure instead of blocking forever.
	cfg := config.Config{Port: -1, ShutdownTimeout: time.Second}

	done := make(chan error, 1)
	go func() { done <- run(cfg, discardLogger()) }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("run on an unusable address = nil; want an error")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run did not report the listen failure")
	}
}

func portOf(t *testing.T, rawURL string) int {
	t.Helper()

	_, portText, err := net.SplitHostPort(rawURL[len("http://"):])
	if err != nil {
		t.Fatalf("parsing %s failed: %v", rawURL, err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("parsing the port of %s failed: %v", rawURL, err)
	}
	return port
}

// newCalculateRequest builds a JSON POST for the calculate endpoint.
func newCalculateRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calculate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}
