# Architecture

`fullstack-calculator` is a deliberately small system built to demonstrate a few
patterns end to end: **Clean Architecture** on the Go side, **Compound
Components + Context Providers** on the React side, a **framework-agnostic
contract** in the middle (`api/openapi.yaml`), and a **single container** as the
deployment unit.

The one non-negotiable rule: **no arithmetic in the frontend.** The browser
builds requests, renders responses and formats strings. Every `+ − × ÷` and every
unary operation is a round trip to the Go API.

```
┌──────────────────────────────┐        POST /api/v1/calculate        ┌──────────────────────────────┐
│  React 19 + Vite + TS + TW   │ ───────────────────────────────────▶ │  Go + Fiber (Clean Arch)     │
│  Compound <Calculator/>      │ ◀─────────────────────────────────── │  domain → usecase → adapter  │
│  CalculatorProvider (state)  │        { result: 5 }                 │  (Fiber is a plug-in)        │
└──────────────────────────────┘                                      └──────────────────────────────┘
          served as static files from the same Go binary in production (one Docker image)
```

---

## 1. Repository layout

```
.
├── api/openapi.yaml          # THE contract. Backend implements it, frontend types mirror it.
├── backend/                  # Go module: github.com/SebasElDev/fullstack-calculator/backend
├── frontend/                 # Vite app
├── deploy/aws/               # CloudFormation templates + scripts (ECR + App Runner)
├── docs/ARCHITECTURE.md      # this file
├── Dockerfile                # multi-stage: build SPA → build Go → distroless runtime
├── compose.yaml              # run the production image locally
├── Makefile                  # dev / test / lint / build / deploy entry points
└── .github/workflows/ci.yml  # test + build on every push, deploy on main
```

---

## 2. Backend (Go) — Clean Architecture

### 2.1 Dependency rule

Source code dependencies point **inward only**. Inner layers know nothing about
outer layers. This is enforced by a test (`internal/architecture_test.go`) that
inspects package imports with `go/build` and fails the build if a layer imports
something it must not.

```
   ┌──────────────────────────────────────────────────────────────────┐
   │ Frameworks & Drivers        cmd/api/main.go, internal/infrastructure/config │
   │  ┌────────────────────────────────────────────────────────────┐  │
   │  │ Interface Adapters        internal/adapter/http/{dto,fiber} │  │
   │  │  ┌──────────────────────────────────────────────────────┐  │  │
   │  │  │ Use Cases               internal/usecase/calculator   │  │  │
   │  │  │  ┌────────────────────────────────────────────────┐  │  │  │
   │  │  │  │ Entities               internal/domain          │  │  │  │
   │  │  │  └────────────────────────────────────────────────┘  │  │  │
   │  │  └──────────────────────────────────────────────────────┘  │  │
   │  └────────────────────────────────────────────────────────────┘  │
   └──────────────────────────────────────────────────────────────────┘
```

| Layer | Package(s) | May import | Must NOT import |
|---|---|---|---|
| Entities | `internal/domain` | stdlib only | anything under `internal/`, any third-party module |
| Use cases | `internal/usecase/calculator` | stdlib, `domain` | `adapter/*`, `infrastructure/*`, Fiber |
| Interface adapters | `internal/adapter/http/dto`, `internal/adapter/http/fiber` | stdlib, `domain`, `usecase`, (`fiber` only inside `adapter/http/fiber`) | `infrastructure/*`, `cmd/*` |
| Frameworks & drivers | `internal/infrastructure/config`, `cmd/api` | everything | — |

### 2.2 Package responsibilities

```
backend/
├── go.mod                              module github.com/SebasElDev/fullstack-calculator/backend
├── cmd/api/main.go                     composition root: config → usecase → fiber app → graceful shutdown
│                                       also: `api -healthcheck` (GET /health, exit 0/1) for distroless HEALTHCHECK
└── internal/
    ├── domain/                         ENTITIES — pure, stdlib-only, no I/O
    │   ├── operation.go                type Operation string + Spec{Name,Symbol,Arity,Description,Apply}
    │   │                               Registry(): ordered list of all Specs; Lookup(Operation) (Spec, bool)
    │   ├── arithmetic.go               Add, Subtract, Multiply, Divide, Power, Modulo, Negate, Sqrt, Square, Percent
    │   │                               each: func(operands ...float64) (float64, error) — returns domain errors only
    │   ├── result.go                   NewResult(v float64) (Result, error): rejects NaN/±Inf, normalises to 15 sig. digits
    │   ├── errors.go                   sentinel errors (see §2.4)
    │   └── *_test.go                   table-driven tests for every operation incl. edge cases
    ├── usecase/calculator/             USE CASES — application rules, still framework-free
    │   ├── ports.go                    type Calculator interface { Calculate(ctx, Input) (Output, error); Operations() []OperationInfo }
    │   │                               Input{Operation, Operands}, Output{Operation, Operands, Result}, OperationInfo{Name,Symbol,Arity,Description}
    │   ├── service.go                  Service implements Calculator: lookup → arity check → Apply → domain.NewResult
    │   └── service_test.go
    ├── adapter/http/
    │   ├── dto/                        INTERFACE ADAPTERS, framework-agnostic
    │   │   ├── calculate.go            CalculateRequest/Response, OperationsResponse, HealthResponse (json tags = OpenAPI names)
    │   │   ├── errors.go               ErrorResponse{Error{Code,Message}}; MapError(err) (httpStatus int, ErrorResponse)
    │   │   └── *_test.go               every domain error → exact (status, code) from the OpenAPI spec
    │   └── fiber/                      the ONLY package that imports gofiber
    │       ├── app.go                  New(Options{Calculator, Version, CORSOrigins, StaticDir}) *fiber.App
    │       ├── handlers.go             thin: parse → dto → usecase → dto → respond. No arithmetic, no business rules.
    │       ├── static.go               optional SPA hosting: /assets/*, index.html fallback for non-/api GETs
    │       └── handlers_test.go        app.Test() against a FAKE Calculator — proves handlers are decoupled from math
    └── infrastructure/config/          FRAMEWORKS & DRIVERS
        ├── config.go                   Load() from env: PORT(8080) STATIC_DIR("") CORS_ALLOWED_ORIGINS("") SHUTDOWN_TIMEOUT(10s)
        └── config_test.go
```

Why `dto` is separate from `fiber`: the request/response shapes and the
error→status mapping are part of the API contract, not of the web framework.
Swapping Fiber for `net/http` or Echo means rewriting `adapter/http/fiber`
only; `dto`, `usecase`, and `domain` are untouched. The handler tests use a fake
`Calculator` to demonstrate the seam.

### 2.3 Operation registry (extensibility)

Adding an operation = one `Spec` entry in `domain.Registry()` + one pure
function + tests. Nothing in `usecase`, `dto`, or `fiber` changes, and
`GET /api/v1/operations` starts advertising it automatically.

```go
// domain/operation.go
type Operation string

const (
    Add      Operation = "add"
    Subtract Operation = "subtract"
    Multiply Operation = "multiply"
    Divide   Operation = "divide"
    Power    Operation = "power"
    Modulo   Operation = "modulo"
    Negate   Operation = "negate"
    Sqrt     Operation = "sqrt"
    Square   Operation = "square"
    Percent  Operation = "percent"
)

type Spec struct {
    Name        Operation
    Symbol      string  // "+", "−", "×", "÷", "xʸ", "mod", "±", "√", "x²", "%"
    Arity       int     // 1 or 2
    Description string
    Apply       func(operands ...float64) (float64, error)
}
```

Semantics (must match `api/openapi.yaml`):

| op | arity | definition | error cases |
|---|---|---|---|
| add | 2 | a + b | overflow → `ErrResultOutOfRange` |
| subtract | 2 | a − b | overflow |
| multiply | 2 | a × b | overflow |
| divide | 2 | a ÷ b | b == 0 → `ErrDivisionByZero` |
| power | 2 | `math.Pow(a,b)` | NaN → `ErrUndefinedResult`, ±Inf → `ErrResultOutOfRange` |
| modulo | 2 | `math.Mod(a,b)` (sign of dividend) | b == 0 → `ErrDivisionByZero` |
| negate | 1 | −a | — |
| sqrt | 1 | `math.Sqrt(a)` | a < 0 → `ErrUndefinedResult` |
| square | 1 | a × a | overflow |
| percent | 1 | a ÷ 100 | — |

The finite check and 15-significant-digit normalisation live in
`domain.NewResult` and are applied by the use case to **every** result, so the
individual functions only need to raise their *semantic* errors
(division by zero, undefined). `NewResult` catches NaN/Inf uniformly.

Normalisation: `strconv.ParseFloat(strconv.FormatFloat(v, 'g', 15, 64), 64)`.
Rationale: `float64` guarantees 15 decimal digits round-trip (`DBL_DIG`);
anything beyond is binary noise (`0.1+0.2`). Also normalises `-0` to `0`.

### 2.4 Errors

```go
// domain/errors.go — sentinel values; wrap with fmt.Errorf("%w: ...") for detail
var (
    ErrUnsupportedOperation = errors.New("unsupported operation")
    ErrInvalidArity         = errors.New("invalid number of operands")
    ErrDivisionByZero       = errors.New("division by zero is undefined")
    ErrUndefinedResult      = errors.New("result is undefined")
    ErrResultOutOfRange     = errors.New("result is out of range")
)
```

`dto.MapError` (adapter layer) translates with `errors.Is`:

| domain error | HTTP | code |
|---|---|---|
| (body parse / type failure in handler) | 400 | `INVALID_REQUEST` |
| `ErrUnsupportedOperation` | 400 | `UNSUPPORTED_OPERATION` |
| `ErrInvalidArity` | 400 | `INVALID_OPERANDS` |
| `ErrDivisionByZero` | 422 | `DIVISION_BY_ZERO` |
| `ErrUndefinedResult` | 422 | `UNDEFINED_RESULT` |
| `ErrResultOutOfRange` | 422 | `RESULT_OUT_OF_RANGE` |
| anything else | 500 | `INTERNAL_ERROR` (message is generic; real error is logged) |

400 vs 422: 400 means "your request is malformed for this API"; 422 means "your
request is valid but the mathematics has no finite answer".

### 2.5 HTTP surface (Fiber adapter)

Routes (see `api/openapi.yaml` for schemas):

| Method | Path | Handler |
|---|---|---|
| GET | `/health` and `/api/v1/health` | `{status:"ok", version}` |
| GET | `/api/v1/operations` | `Calculator.Operations()` → `OperationsResponse` |
| POST | `/api/v1/calculate` | `CalculateRequest` → `Calculator.Calculate` → `CalculateResponse` / `ErrorResponse` |
| GET | `/*` (only when `STATIC_DIR` set) | SPA assets; `index.html` fallback for non-`/api` paths; `/api/*` unknown routes still 404 JSON |

Middleware: `recover` (panics → 500 `INTERNAL_ERROR`), `requestid`, structured
request logging (`log/slog`), `cors` only when `CORS_ALLOWED_ORIGINS` is set
(same-origin deployments need none). Body limit small (1 KiB is plenty).

Server lifecycle (`cmd/api/main.go`): listen on `:PORT`, trap
`SIGINT`/`SIGTERM`, `ShutdownWithTimeout(SHUTDOWN_TIMEOUT)`. Version string is
injected via `-ldflags "-X main.version=<sha>"`.

### 2.6 Testing strategy (Go)

* `domain`: table-driven tests per operation incl. 0.1+0.2, ÷0, √−1, overflow,
  `-0`, `math.Pow` edge cases, normalisation.
* `usecase`: arity validation, unknown op, error propagation, ordering of
  `Operations()` matching `domain.Registry()`.
* `dto`: exhaustive error mapping table.
* `fiber`: `app.Test()` with a fake `Calculator` — status codes, JSON shapes,
  malformed body, unknown route under `/api`, health, static fallback.
* `architecture_test.go`: dependency rule.
* `go test -race -cover ./...` in CI; target ≥ 90 % on `domain`, `usecase`, `dto`.

---

## 3. Frontend (React 19 + Vite + TypeScript + Tailwind CSS v4)

### 3.1 Patterns

**Context Provider pattern** — two providers, two concerns:

* `ApiClientProvider` — dependency injection of the typed API client. Tests
  inject a mock client; production injects `createApiClient({ baseUrl })`.
  Hook: `useApiClient()` (throws if no provider).
* `CalculatorProvider` — owns the calculator state machine (`useReducer`) and
  the async actions that talk to the API. Hook: `useCalculator()` (throws if no
  provider). Components never call `fetch` and never touch numbers.

**Compound Component pattern** — `Calculator` is a root that provides context
and exposes its parts as static properties. Consumers compose the UI
declaratively; parts communicate through context, not props:

```tsx
<ApiClientProvider client={client}>
  <Calculator>                    {/* renders CalculatorProvider + card shell */}
    <Calculator.Status />         {/* API online / offline / checking */}
    <Calculator.Display />        {/* expression line + current input / result / error */}
    <Calculator.Keypad />         {/* default grid built from <Calculator.Key/> */}
    <Calculator.History />        {/* session history of server results */}
  </Calculator>
</ApiClientProvider>

// Custom layouts are possible because Key is public:
<Calculator.Keypad>
  <Calculator.Key action={{ type: "digit", digit: "7" }} />
  <Calculator.Key action={{ type: "operator", operation: "add" }} />
</Calculator.Keypad>
```

Every subcomponent calls `useCalculator()`; using one outside `<Calculator>`
throws a descriptive error (tested).

### 3.2 Layout

```
frontend/
├── index.html · vite.config.ts · tsconfig*.json · biome.json · package.json
├── src/
│   ├── main.tsx                         mounts <App/>
│   ├── App.tsx                          wires ApiClientProvider + Calculator composition
│   ├── index.css                        @import "tailwindcss"; theme tokens
│   ├── lib/
│   │   ├── api/
│   │   │   ├── types.ts                 mirrors api/openapi.yaml: OperationName, CalculateRequest/Response,
│   │   │   │                            OperationInfo, HealthResponse, ErrorCode, ErrorResponse
│   │   │   ├── client.ts                createApiClient({ baseUrl, fetch? }) → ApiClient { calculate, listOperations, health }
│   │   │   │                            ApiError extends Error { status, code, message }
│   │   │   └── client.test.ts
│   │   └── operations.ts                OPERATIONS: Record<OperationName, OperationMeta{ symbol, label, arity, kind:"binary"|"unary" }>
│   │                                    single source of truth for keypad glyphs and history formatting
│   ├── features/calculator/
│   │   ├── context/
│   │   │   ├── ApiClientContext.tsx     ApiClientProvider, useApiClient
│   │   │   └── CalculatorContext.tsx    CalculatorProvider, useCalculator (state + actions)
│   │   ├── state/
│   │   │   ├── types.ts                 CalculatorState, CalculatorAction, KeyAction, HistoryEntry
│   │   │   ├── reducer.ts               pure transitions — string editing only, NO arithmetic
│   │   │   ├── formatting.ts            expression strings for history ("2 + 3", "√(9)") — string work only
│   │   │   └── *.test.ts
│   │   ├── components/
│   │   │   ├── Calculator.tsx           root + static assignment of parts
│   │   │   ├── CalculatorDisplay.tsx · CalculatorKeypad.tsx · CalculatorKey.tsx
│   │   │   ├── CalculatorHistory.tsx · CalculatorStatus.tsx
│   │   │   └── *.test.tsx
│   │   └── index.ts                     public surface of the feature
│   └── test/
│       ├── setup.ts                     @testing-library/jest-dom
│       └── utils.tsx                    renderWithProviders(ui, { client }), createMockApiClient()
```

### 3.3 State machine (no arithmetic)

```ts
interface CalculatorState {
  input: string;                        // what the user is typing, e.g. "12.5" (always a valid numeric prefix; starts "0")
  accumulator: number | null;           // left operand; ONLY ever set from a server result or Number(input)
  pendingOperation: BinaryOperationName | null;
  overwrite: boolean;                   // next digit replaces `input` (true after a result or an operator)
  expression: string | null;            // secondary line, e.g. "2 +" or "2 + 3 ="
  status: "idle" | "calculating" | "error";
  error: { code: ErrorCode | "NETWORK_ERROR"; message: string } | null;
  history: HistoryEntry[];              // { id, expression: "2 + 3", result: 5 } — newest first, capped (e.g. 20)
}
```

Reducer actions (pure): `DIGIT`, `DECIMAL`, `BACKSPACE`, `CLEAR`,
`OPERATOR_SELECTED`, `CALCULATION_STARTED`, `CALCULATION_SUCCEEDED`,
`CALCULATION_FAILED`.

Provider actions (async, orchestrate reducer + API):

| user action | behaviour |
|---|---|
| digit / decimal / backspace / clear | reducer only |
| operator `op` (binary) | if a `pendingOperation` exists and the user has typed a right operand → `calculate(pending, [accumulator, Number(input)])`, result becomes `accumulator`, `pendingOperation = op`. Otherwise `accumulator = Number(input)`, `pendingOperation = op`. Pressing two operators in a row just replaces the pending one. |
| unary `op` | `calculate(op, [Number(input)])` → result becomes `input`; history entry `√(9) = 3` |
| equals | requires `pendingOperation` → `calculate(pending, [accumulator, Number(input)])` → result becomes `input`, accumulator/pending cleared, history entry `2 + 3 = 5` |
| any key while `status === "calculating"` | ignored (keys disabled) |
| API error | `status = "error"`, message shown in display; next digit/clear resets |

`Number("12.5")` is *parsing*, `String(result)` is *rendering* — neither is
arithmetic. The display shows exactly what the server returned.

### 3.4 Keypad

All ten operations, digits `0-9`, `.`, `AC`, `⌫`, `=` must be reachable. Suggested
4-column grid (the frontend agent may adjust for aesthetics, not for scope):

```
AC   ⌫    mod   ÷
√    x²   xʸ    ×
7    8    9     −
4    5    6     +
1    2    3     =   (= spans two rows)
±    0    .     %
```

Keyboard support: digits, `.`, `+ - * /`, `Enter`/`=`, `Backspace`, `Escape`.

### 3.5 Testing strategy (Vitest + Testing Library + jsdom)

* `reducer.test.ts` — every transition, incl. leading zero, single decimal
  point, backspace to empty → "0", overwrite semantics, error reset.
* `CalculatorContext.test.tsx` — with a **mock client that returns a wrong
  number on purpose** (e.g. `2 + 3 → 42`): the display must show `42`. This is
  the proof that no arithmetic happens client-side. Also: request payload
  shapes, chained operators, unary ops, error handling, keys disabled while
  calculating.
* `client.test.ts` — URL/method/body, success parsing, `ErrorResponse` →
  `ApiError`, network failure → `NETWORK_ERROR`.
* Component tests — each compound part; "throws outside provider"; keyboard
  events.
* Coverage via `@vitest/coverage-v8`, target ≥ 85 % on `src/features` and
  `src/lib`.

Tooling: Biome for lint + format, `tsc --noEmit` for types.

---

## 4. Contract discipline

* `api/openapi.yaml` is the source of truth. Backend DTO json tags and frontend
  `lib/api/types.ts` mirror it by hand (the project is small enough that codegen
  would be more machinery than value; the README says how to regenerate types
  with `openapi-typescript` if it grows).
* `frontend/src/lib/operations.ts` and `backend/internal/domain.Registry()` must
  list the same ten operations with the same symbols. A frontend test asserts
  the `OperationName` union matches `OPERATIONS` keys; the README documents the
  manual cross-check against `GET /api/v1/operations`.

---

## 5. Build & deployment

**Single image, single process.** The multi-stage `Dockerfile`:

1. `node:22-alpine` — `npm ci && npm run build` → `frontend/dist`
2. `golang:1.27-alpine` — `CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$VERSION" ./cmd/api`
3. `gcr.io/distroless/static-debian12:nonroot` — binary + `dist/` at
   `/app/public`; `ENV PORT=8080 STATIC_DIR=/app/public`; `EXPOSE 8080`;
   `HEALTHCHECK CMD ["/app/api","-healthcheck"]`

**AWS (isolated from any other stack):** ECR repository + **AWS App Runner**
service (0.25 vCPU / 0.5 GB, port 8080, health check `/health`). Provisioned
via two CloudFormation stacks in `deploy/aws/`:

* `foundation.yaml` — ECR repo (lifecycle: keep last 10), App Runner ECR
  access role, GitHub OIDC provider (optional, conditional) + a deploy role
  scoped to this repo's `main` branch that may push to this ECR repo and start
  App Runner deployments.
* `service.yaml` — the App Runner service (params: `ImageUri`, `AccessRoleArn`).

Scripts: `bootstrap.sh` (first deploy), `deploy.sh` (build → push → start
deployment), `destroy.sh` (delete service, empty & delete ECR, delete
foundation). Everything is tagged `Project=fullstack-calculator`.

**CI/CD (`.github/workflows/ci.yml`):** `backend` (gofmt, vet, test -race),
`frontend` (biome, tsc, vitest), `docker` (build, no push) on every push/PR;
`deploy` job on `main` only, authenticating with OIDC (`vars.AWS_ROLE_ARN`),
tags the image with the commit SHA, pushes, and triggers
`aws apprunner start-deployment`.

---

## 6. Non-goals (kept out on purpose)

* Persistence / shared history — the API is stateless (12-factor, horizontally
  scalable); history lives in the browser session. A `HistoryRepository`
  output port would slot into `usecase/calculator` if this ever changed.
* Expression parsing (`2+3*4` with precedence) — a natural next step for the
  domain layer; the UI follows immediate-execution semantics like a desk
  calculator.
* Authentication, rate limiting beyond App Runner defaults.
