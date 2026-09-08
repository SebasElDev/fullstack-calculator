# fullstack-calculator

A small full-stack calculator built to demonstrate a few patterns end to end,
cleanly: **Clean Architecture** on a Go backend, **Compound Components +
Context Providers** on a React 19 frontend, a hand-maintained **OpenAPI
contract** between them, and a **single container** as the deployment unit.

[![CI](https://github.com/SebasElDev/fullstack-calculator/actions/workflows/ci.yml/badge.svg)](https://github.com/SebasElDev/fullstack-calculator/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/go-1.27-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-blue)

## The one non-negotiable rule

**No arithmetic in the frontend.** In practice this means:

- The browser parses whatever the user types (`Number(input)`), sends it to
  the Go API, and renders exactly what comes back (`String(result)`). Those
  two conversions are the only numeric operations anywhere in the React code.
- Every operation is a network request — including `negate` (±) and
  `percent` (%), which are unary but still round trips to `POST
  /api/v1/calculate`. There is no "cheap" client-side shortcut for them.
- A Vitest test proves it directly: it mocks the API to answer `2 + 3` with
  `42`, and asserts the display shows `42`. If the frontend were computing
  anything itself, that test would fail.

## Live demo

<https://vegmwcswz2.us-east-1.awsapprunner.com> — deployed on AWS App Runner from the `main` branch.

```sh
curl https://vegmwcswz2.us-east-1.awsapprunner.com/health
# {"status":"ok","version":"<git-sha>"}
```

## Stack

| Layer | Technology | Version | Why it's there |
|---|---|---|---|
| Backend language | Go | 1.27 | Static binary, no runtime, fast startup — a good fit for a container that does one small job. |
| Backend web framework | [Fiber](https://gofiber.io) v3 | v3.5.0 | Low-overhead router with built-in `recover`, `requestid` and `cors` middleware; confined to a single package (see [Architecture](#architecture)) so it stays swappable. |
| Frontend framework | React | 19.2.8 | `use()`, the `ref`-as-prop change, and first-class Suspense/transitions primitives. |
| Frontend build tool | Vite | 8.2.2 | Fast dev server with a built-in proxy for `/api` and `/health`, and the production bundler. |
| Frontend language | TypeScript | ~6.0.2 | Static types for the DTOs shared with the OpenAPI contract. |
| CSS | Tailwind CSS | 4.3.3 | CSS-first configuration (`@import "tailwindcss"` + theme tokens), no separate config file. |
| Frontend testing | Vitest + Testing Library + jsdom | 4.1.11 | Fast, Vite-native test runner; Testing Library keeps tests behavior-focused. |
| Lint / format (frontend) | [Biome](https://biomejs.dev) | 2.5.12 | One tool for both lint and format, replacing ESLint + Prettier. |
| Container runtime | `gcr.io/distroless/static-debian12:nonroot` | — | No shell, no package manager, non-root by default — the smallest attack surface for a single static binary. |
| Deployment | AWS App Runner + CloudFormation | — | Fully managed containers with no cluster to operate, provisioned as code. |

## Architecture

The system has two halves that only ever talk over HTTP, in JSON, against a
contract neither side owns unilaterally:

```
┌──────────────────────────────┐        POST /api/v1/calculate        ┌──────────────────────────────┐
│  React 19 + Vite + TS + TW   │ ───────────────────────────────────▶ │  Go + Fiber (Clean Arch)     │
│  Compound <Calculator/>      │ ◀─────────────────────────────────── │  domain → usecase → adapter  │
│  CalculatorProvider (state)  │        { result: 5 }                 │  (Fiber is a plug-in)        │
└──────────────────────────────┘                                      └──────────────────────────────┘
          served as static files from the same Go binary in production (one Docker image)
```

The frontend never imports a math function; the backend never imports a UI
concern. `api/openapi.yaml` is the arbiter of what either side is allowed to
assume about the other. Full design rationale lives in
[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — this section is a summary.

### Backend: Clean Architecture

Dependencies point inward only, and a test enforces it by walking the import
graph with `go/build` (`backend/internal/architecture_test.go`,
`TestDependencyRule` / `TestTheFrameworkStaysInItsPackage`):

| Layer | Package(s) | May import | Must NOT import |
|---|---|---|---|
| Entities | `internal/domain` | stdlib only | anything under `internal/`, any third-party module |
| Use cases | `internal/usecase/calculator` | stdlib, `domain` | `adapter/*`, `infrastructure/*`, Fiber |
| Interface adapters | `internal/adapter/http/dto`, `internal/adapter/http/fiber` | stdlib, `domain`, `usecase`, (Fiber only inside `adapter/http/fiber`) | `infrastructure/*`, `cmd/*` |
| Frameworks & drivers | `internal/infrastructure/config`, `cmd/api` | everything | — |

`TestTheFrameworkStaysInItsPackage` is the headline guarantee: Fiber is
imported in exactly two places (`internal/adapter/http/fiber` and
`cmd/api`), so swapping it for `net/http` or Echo means rewriting that one
adapter package — `dto`, `usecase`, and `domain` are untouched.

### Frontend: Compound Component + two Context Providers

`Calculator` is a compound component: a root that owns context and exposes
its parts as static properties, composed declaratively by the caller
(`frontend/src/App.tsx`):

```tsx
<ApiClientProvider client={apiClient}>
  <Calculator>
    <Calculator.Status />
    <Calculator.Display />
    <Calculator.Keypad />
    <Calculator.History />
  </Calculator>
</ApiClientProvider>
```

Two providers, two separate concerns:

- **`ApiClientProvider`** — dependency injection of the typed API client.
  Production injects `createApiClient({ baseUrl })`; tests inject a mock.
- **`CalculatorProvider`** — owns the state machine (`useReducer`) and the
  async actions that call the API. Every subcomponent reads it through
  `useCalculator()`, which throws a descriptive error when used outside
  `<Calculator>`.

### Repository layout

```
.
├── api/openapi.yaml          # THE contract. Backend implements it, frontend types mirror it.
├── backend/                  # Go module: github.com/SebasElDev/fullstack-calculator/backend
│   ├── cmd/api/               # composition root + -healthcheck flag
│   └── internal/
│       ├── domain/            # entities: operations, arithmetic, result normalisation
│       ├── usecase/calculator/# application rules
│       ├── adapter/http/      # dto (framework-agnostic) + fiber (the only package importing Fiber)
│       └── infrastructure/config/
├── frontend/                  # Vite app
│   └── src/
│       ├── lib/api/            # hand-written mirror of the OpenAPI contract + typed client
│       ├── lib/operations.ts   # single source of truth for keypad glyphs/labels
│       └── features/calculator/# context, state (reducer/formatting/keyboard), components
├── deploy/aws/                # CloudFormation templates + scripts (ECR + App Runner)
├── docs/ARCHITECTURE.md       # full design document
├── Dockerfile                 # multi-stage: build SPA → build Go → distroless runtime
├── compose.yaml               # run the production image locally
├── Makefile                   # dev / test / lint / build / deploy entry points
└── .github/workflows/ci.yml   # test + build on every push, deploy on main
```

## How to run

### Prerequisites

- Go 1.27
- Node 22+ and npm
- Docker (only to build/run the production image)
- AWS CLI and the `gh` CLI (only for deployment — see [Deployment](#deployment))

### `make help`

```
fullstack-calculator — available targets:
  help            Show this help
  dev             Run backend and frontend dev servers concurrently
  dev-api         Run the Go API in dev mode
  dev-web         Run the Vite dev server
  install         Install all dependencies (frontend + backend)
  test            Run backend and frontend test suites
  test-backend    Run Go tests with race detector and coverage
  test-frontend   Run Vitest with the coverage thresholds enforced
  lint            Lint and typecheck both backend and frontend
  lint-backend    gofmt (fails on unformatted files) and go vet
  lint-frontend   Biome lint/format check and TypeScript typecheck
  build           Build production frontend assets and the Go binary
  build-frontend  Build the SPA into frontend/dist
  build-backend   Build the Go binary into backend/bin/api
  docker-build    Build the production Docker image
  docker-run      Run the production image locally on :8080
  bootstrap       First-time AWS provisioning (foundation + service stacks)
  deploy          Build, push and deploy the current commit to AWS
  destroy         Tear down all AWS resources for this project
```

### Local development

```sh
make install   # frontend: npm ci · backend: go mod download
make dev       # runs dev-api and dev-web concurrently
```

`make dev` starts the Vite dev server on `:5173`, which proxies `/api` and
`/health` to the Go API on `:8080` (configured in `frontend/vite.config.ts`).
Open `http://localhost:5173`.

Two-terminal alternative, if you want the logs separated:

```sh
# terminal 1
cd backend && go run ./cmd/api

# terminal 2
cd frontend && npm run dev
```

### Environment variables

Backend (`backend/internal/infrastructure/config/config.go` — every variable
is optional; an invalid value is a startup error, not a silent fallback):

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | TCP port the HTTP server listens on. |
| `STATIC_DIR` | `""` (disabled) | Directory holding the built SPA. When set, the server also hosts the frontend and falls back to `index.html` for client-side routes. |
| `CORS_ALLOWED_ORIGINS` | `""` (disabled) | Comma-separated list of allowed browser origins. Same-origin deployments (the binary serving its own SPA) need none. |
| `SHUTDOWN_TIMEOUT` | `10s` | How long graceful shutdown waits for in-flight requests to finish before forcing an exit. |

Frontend (`frontend/.env.example`):

| Variable | Default | Purpose |
|---|---|---|
| `VITE_API_BASE_URL` | `/api/v1` | Base URL of the calculator API. The default is same-origin, which is correct for both `npm run dev` (proxied) and the production container (served together). Point it elsewhere only when the API lives on a different origin. |

### Running the production image locally

```sh
make docker-build && make docker-run   # builds, then runs on :8080
```

or, equivalently:

```sh
docker compose up --build
```

`compose.yaml` builds the same multi-stage `Dockerfile`, publishes `:8080`,
and wires up the same container `HEALTHCHECK` the image ships with.

## API usage

Full schema, including request/response examples for every status code, is
in [`api/openapi.yaml`](api/openapi.yaml) — treat it as the source of truth.
If the contract grows large enough that hand-mirroring types stops being
comfortable, generate the frontend types from it instead of maintaining them
by hand:

```sh
npx openapi-typescript api/openapi.yaml -o frontend/src/lib/api/schema.d.ts
```

### Endpoints

| Method | Path | Purpose |
|---|---|---|
| GET | `/health` | Liveness probe, also mounted without the API prefix for container platforms. |
| GET | `/api/v1/health` | Same liveness probe, under the versioned API prefix. |
| GET | `/api/v1/operations` | Canonical list of supported operations, in keypad order. |
| POST | `/api/v1/calculate` | Perform one operation. |

### Examples

Captured against `PORT=18090 go run ./cmd/api` from `backend/`.

**Health:**

```sh
$ curl -s http://localhost:18090/health
{"status":"ok","version":"dev"}
```

**Operations** (abbreviated — ten total, see the full list via the same call):

```sh
$ curl -s http://localhost:18090/api/v1/operations | python3 -m json.tool
{
    "operations": [
        {"name": "add", "symbol": "+", "arity": 2, "description": "Sum of two operands"},
        {"name": "subtract", "symbol": "−", "arity": 2, "description": "Difference of two operands"},
        {"name": "sqrt", "symbol": "√", "arity": 1, "description": "Principal square root"},
        ...
    ]
}
```

**A binary calculation — result normalisation:**

```sh
$ curl -s -X POST http://localhost:18090/api/v1/calculate \
    -H "Content-Type: application/json" \
    -d '{"operation":"add","operands":[0.1,0.2]}'
{"operation":"add","operands":[0.1,0.2],"result":0.3}
```

`0.1 + 0.2` is `0.30000000000000004` in raw `float64` binary arithmetic; the
API normalises to 15 significant digits before it ever leaves the server, so
clients never see that artefact.

**A unary calculation:**

```sh
$ curl -s -X POST http://localhost:18090/api/v1/calculate \
    -H "Content-Type: application/json" \
    -d '{"operation":"sqrt","operands":[9]}'
{"operation":"sqrt","operands":[9],"result":3}
```

**400 — arity mismatch:**

```sh
$ curl -s -w "\nHTTP %{http_code}\n" -X POST http://localhost:18090/api/v1/calculate \
    -H "Content-Type: application/json" \
    -d '{"operation":"add","operands":[2]}'
{"error":{"code":"INVALID_OPERANDS","message":"invalid number of operands: operation \"add\" expects 2 operands, got 1"}}
HTTP 400
```

**422 — division by zero:**

```sh
$ curl -s -w "\nHTTP %{http_code}\n" -X POST http://localhost:18090/api/v1/calculate \
    -H "Content-Type: application/json" \
    -d '{"operation":"divide","operands":[5,0]}'
{"error":{"code":"DIVISION_BY_ZERO","message":"division by zero is undefined: cannot divide 5 by zero"}}
HTTP 422
```

**404 — unknown route under `/api`:**

```sh
$ curl -s -w "\nHTTP %{http_code}\n" http://localhost:18090/api/v1/nope
{"error":{"code":"NOT_FOUND","message":"no route for GET /api/v1/nope"}}
HTTP 404
```

### Error codes

| Code | HTTP status | When |
|---|---|---|
| `INVALID_REQUEST` | 400 | Malformed body, wrong field types, or a body over 1 KiB. |
| `UNSUPPORTED_OPERATION` | 400 | The `operation` field is not one of the ten known names. |
| `INVALID_OPERANDS` | 400 | Wrong number of operands for the operation's arity. |
| `DIVISION_BY_ZERO` | 422 | `divide` or `modulo` with a zero divisor. |
| `UNDEFINED_RESULT` | 422 | The mathematics has no real answer, e.g. `sqrt(-1)` (`NaN`). |
| `RESULT_OUT_OF_RANGE` | 422 | The result overflows `float64` (`±Inf`). |
| `NOT_FOUND` | 404 | No route matches the method and path under `/api`. |
| `INTERNAL_ERROR` | 500 | Unexpected failure; the real cause is logged, never leaked in the response. |

**400 vs 422:** 400 means *your request is malformed for this API* — bad
JSON, an unknown operation, the wrong number of operands. 422 means *your
request is perfectly valid, but the mathematics has no finite answer* — the
server understood exactly what you asked for and computed that it cannot be
answered.

## Design specifications

Things a reviewer should notice, beyond what's already covered above:

- **Clean Architecture + the dependency test.** The layer table in
  [Architecture](#architecture) is enforced, not just documented —
  `TestDependencyRule` fails the build the moment any package imports
  something the table forbids, and `TestEveryPackageHasARule` fails if a new
  package is added without a rule (so the table can't silently go stale).

- **Fiber confined to one package.** All Fiber types and imports live in
  `internal/adapter/http/fiber`. To swap Fiber for `net/http` or Echo:
  rewrite `app.go`, `handlers.go`, `static.go`, and `errors.go` in that
  package against the same `calculator.Calculator` input port; `domain`,
  `usecase`, and `dto` need no changes. The handler tests
  (`handlers_test.go`) already prove the seam by exercising a fake
  `Calculator`.

- **Operation registry.** Adding an operation is: (1) a pure function in
  `backend/internal/domain/arithmetic.go` (`func(operands ...float64)
  (float64, error)`), (2) one `Spec{Name, Symbol, Arity, Description, Apply}`
  entry in the `registry` slice in `backend/internal/domain/operation.go`,
  (3) table-driven tests for it, and (4) a matching entry in
  `frontend/src/lib/operations.ts`. Nothing in `usecase`, `dto`, or `fiber`
  changes — `GET /api/v1/operations` advertises the new operation
  automatically because it reads `domain.Registry()` directly.

- **Result normalisation.** `domain.NewResult` rounds every result to 15
  significant digits (`strconv.ParseFloat(strconv.FormatFloat(v, 'g', 15,
  64), 64)`) and maps `-0` to `0`. 15 is `float64`'s guaranteed decimal
  round-trip precision (`DBL_DIG`); anything beyond it is binary
  representation noise — the reason `0.1 + 0.2` prints as
  `0.30000000000000004` in naive floating point. The same function rejects
  non-finite values (`NaN` → `ErrUndefinedResult`, `±Inf` →
  `ErrResultOutOfRange`), so individual arithmetic functions only raise
  their own semantic errors.

- **Error taxonomy.** Five domain sentinel errors
  (`ErrUnsupportedOperation`, `ErrInvalidArity`, `ErrDivisionByZero`,
  `ErrUndefinedResult`, `ErrResultOutOfRange`) plus one adapter-level
  `ErrInvalidRequest` for bodies that never made it to a valid
  `CalculateRequest`. `dto.MapError` is the single, exhaustively tested
  translation table from those sentinels to (HTTP status, `ErrorCode`)
  pairs; anything unrecognised becomes a generic 500 whose message never
  leaks internals.

- **Layered body limits.** The `calculate` handler rejects any body over 1
  KiB itself, with the full JSON error envelope and a 413 status — that's
  the size any legal request could plausibly need (an operation name plus
  two numbers). Independently, Fiber's `BodyLimit` is configured at 64 KiB
  as a transport-level ceiling: it protects the process against abusive
  payloads before any handler runs, and is never expected to trigger for a
  legitimate client.

- **HTTP hardening.** Every response carries a same-origin
  Content-Security-Policy (`frame-ancestors 'none'`, no plugins, no external
  scripts), `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY` and a
  strict `Referrer-Policy`, set once by Fiber's `helmet` middleware. Vite's
  content-hashed bundles under `/assets/` are served with
  `Cache-Control: public, max-age=31536000, immutable`, a missing bundle is
  an honest JSON 404 rather than the SPA shell, and the shell itself is
  `no-cache` so a redeploy is picked up on the next navigation.

- **Graceful shutdown + `-healthcheck`.** `cmd/api/main.go` traps
  `SIGINT`/`SIGTERM`, stops accepting new connections, and waits up to
  `SHUTDOWN_TIMEOUT` for in-flight requests to finish. The same binary also
  accepts a `-healthcheck` flag that performs a single GET against its own
  `/health` and exits 0/1 — used by the Dockerfile's `HEALTHCHECK`, because
  a `FROM scratch`-style distroless image has no shell and no `curl` to run
  a probe with.

- **Frontend state machine.** `CalculatorState` holds: `input` (what the
  user is typing — always a valid numeric prefix), `accumulator` (the left
  operand — **set only from a server result or `Number(input)`, never from
  arithmetic**), `pendingOperation`, `overwrite` (next digit replaces
  `input`), `expression` (the secondary display line), `status`
  (`idle`/`calculating`/`error`), `error`, and `history`. The reducer
  (`state/reducer.ts`) performs pure string transitions only; the provider
  (`context/CalculatorContext.tsx`) is the only place that calls the API and
  folds a response back into state.

- **Keyboard support.** Digits, `.`, `+ - * /`, `Enter`/`=`, `Backspace`,
  `Escape` all map onto the same `KeyAction` the on-screen keys dispatch
  (`state/keyboard.ts`). Keystrokes are ignored while focus is on an
  editable element (`isEditableTarget`), and a focused key's own `Enter`
  handling is left to the browser rather than double-fired.

- **Accessibility choices.** The display uses `aria-live="polite"` +
  `aria-atomic` so screen readers announce new results and errors. Keys use
  `aria-disabled` rather than the native `disabled` attribute while a
  calculation is in flight — a genuinely disabled button falls out of the
  accessibility tree and drops focus to `<body>`, forcing a keyboard user to
  tab back in after every calculation; `aria-disabled` just ignores the
  press. The currently pending binary operator key carries `aria-pressed`.

- **Contract discipline.** `frontend/src/lib/operations.ts` and
  `backend/internal/domain.Registry()` must list the same ten operations
  with the same symbols and arities. A frontend test
  (`operations.test.ts`) pins `OPERATIONS` against an independent copy of
  the registry table; a Go test asserts `Operations()` matches
  `domain.Registry()` in order. The cross-wire check —
  `curl localhost:8080/api/v1/operations` compared by eye against
  `OPERATIONS` — is manual and documented in `frontend/README.md`.

## Testing

```sh
make test            # backend + frontend
make test-backend    # go test -race -cover ./...
make test-frontend   # npm run test:coverage (vitest, 85% thresholds enforced)
```

Coverage:

```sh
cd backend && go test -race -cover ./...
cd frontend && npm run test:coverage
```

### What each suite covers, and the real numbers

**Backend** — 53 top-level Go tests across 7 packages, all passing with
`-race`:

| Package | Tests | Coverage |
|---|---|---|
| `internal/domain` | 12 | 100.0% |
| `internal/usecase/calculator` | 5 | 100.0% |
| `internal/adapter/http/dto` | 9 | 100.0% |
| `internal/adapter/http/fiber` | 15 | 100.0% |
| `internal/infrastructure/config` | 4 | 100.0% |
| `cmd/api` | 5 | 62.5% (the `run`/signal-handling loop is exercised only partially by unit tests) |
| `internal` (architecture tests) | 3 | — (no statements; the tests inspect other packages) |

`domain` covers every operation table-driven, including `0.1 + 0.2`,
division by zero, `sqrt(-1)`, overflow, and `-0` normalisation. `usecase`
covers arity validation, unknown operations, and that `Operations()` matches
`domain.Registry()` in order. `dto` exhaustively covers the error-mapping
table. `fiber` drives the app with `app.Test()` against a fake `Calculator`,
covering status codes, JSON shapes, malformed bodies, the 413/404 paths, and
static-file fallback.

**Frontend** — 14 test files, 183 tests, 100% coverage (statements
260/260, branches 167/167, functions 71/71, lines 259/259) on the
Vitest v8 provider, well above the 85% threshold configured in
`vite.config.ts`. Covers the reducer's pure transitions, the API client's
URL/method/body construction and error parsing, every compound component
(including "throws outside provider"), and keyboard bindings.

**The two signature tests:**

- `frontend/src/features/calculator/context/CalculatorContext.test.tsx` —
  *"displays what the server returned, even when the answer is wrong"*: the
  mock API is told to answer `2 + 3` with `42`, and the test asserts the
  display shows `42` and the history reads `2 + 3 = 42`. This is the direct
  proof that no arithmetic happens client-side — a real calculation could
  never produce that answer.
- `backend/internal/architecture_test.go` — `TestDependencyRule` walks the
  import graph of every package with `go/build` and fails the build if any
  layer imports something the dependency table forbids;
  `TestTheFrameworkStaysInItsPackage` specifically asserts Fiber is
  imported nowhere outside the adapter and the composition root.

**Formatting and static analysis**, all currently passing:

- `gofmt -l .` (backend) — no unformatted files.
- `go vet ./...` (backend) — clean.
- `npx biome check .` (frontend) — lint + format, clean across 43 files.
- `npx tsc --noEmit -p tsconfig.app.json` (frontend) — no type errors.

CI (`.github/workflows/ci.yml`) runs the same Makefile targets on every push
and pull request, so there is exactly one definition of each gate: the
`backend` job runs `make lint-backend` and `make test-backend`, the `frontend`
job runs `make lint-frontend`, `make test-frontend` and `make build-frontend`,
and a `docker` job builds the image for `linux/amd64` without pushing it. The
workflow token is read-only; only a push to `main` additionally runs the
`deploy` job, which builds and pushes the image and then executes
`deploy/aws/deploy.sh` — the same script an operator runs locally.

## Deployment

### Single-image strategy

The production artifact is one Docker image built by a three-stage
`Dockerfile`:

1. `node:22-alpine` — `npm ci && npm run build` → `frontend/dist`.
2. `golang:1.27-alpine` — cross-compiled on the build platform
   (`CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X
   main.version=$VERSION"`), so building `--platform linux/amd64` on Apple
   Silicon never runs under emulation.
3. `gcr.io/distroless/static-debian12:nonroot` — just the Go binary and
   `frontend/dist` copied in at `/app/public`, running as `nonroot`, with a
   `HEALTHCHECK` that calls `/app/api -healthcheck`. The resulting image is
   roughly 22 MB — a static binary plus a handful of SPA assets, no OS
   packages, no shell.

### AWS layout

Two CloudFormation stacks, deployed from `deploy/aws/`:

- **`foundation.yaml`** — the account-level, rarely-changing pieces: the
  ECR repository (lifecycle policy: keep the last 10 images), the IAM role
  App Runner assumes to pull from it, and — optionally, since an account
  can only have one — a GitHub OIDC provider plus a deploy role scoped to
  `SebasElDev/fullstack-calculator`'s `main` branch, permissioned only to
  push to this one ECR repo and manage this one App Runner service.
- **`service.yaml`** — the App Runner service itself: `0.25 vCPU / 0.5 GB`,
  listening on port 8080, health check on `GET /health` (10s interval, 1
  healthy / 5 unhealthy threshold). A deploy is a CloudFormation update of
  this stack with a new `ImageUri` (the commit SHA tag); App Runner rolls
  out the new image, and the stack stays the single source of truth for
  what's running.

Every resource in both stacks is tagged `Project=fullstack-calculator`, so
the whole deployment is trivially identifiable and isolated from any other
workload in the same AWS account — and torn down by exactly one command.

### First deploy

```sh
make bootstrap
```

Runs `deploy/aws/bootstrap.sh`, which: checks whether the AWS account
already has a `token.actions.githubusercontent.com` OIDC provider (an
account may only have one) and reuses it if so; deploys the foundation
stack; calls `deploy.sh` to build, push, and stand up the service; then
prints the live service URL and the exact `gh variable set ...` commands
needed to wire up CI/CD.

### Subsequent deploys

```sh
make deploy
```

Runs `deploy/aws/deploy.sh`: builds the image for `linux/amd64` with
`docker buildx`, pushes it to ECR tagged with the current commit SHA,
updates the `service.yaml` stack with the new `ImageUri`, polls
`apprunner describe-service` until the status is `RUNNING` (or fails fast on
a `*_FAILED` status), and curls `/health` as a final sanity check.

### Teardown

```sh
make destroy
```

Runs `deploy/aws/destroy.sh`: deletes the service stack, empties and then
lets the foundation stack delete the ECR repository, and deletes the
foundation stack — prompting for a typed `yes` confirmation unless
`FORCE=1` is set.

### Cost

An App Runner instance provisioned at 0.25 vCPU / 0.5 GB, running
continuously, costs on the order of a few US dollars per month; ECR storage
for a handful of ~22 MB images is negligible. There is no other billed
resource — no load balancer, no NAT gateway, no database.

### CI/CD

`.github/workflows/ci.yml` runs `backend`, `frontend`, and `docker` (build
only, no push) on every push and pull request. The `deploy` job runs only on
a push to `main`, after the other three succeed, and authenticates to AWS
with GitHub OIDC — no long-lived AWS credentials are stored in the repo. The
deploy role trusts only this repository's immutable OIDC subject (owner and
repository IDs, the default for repositories created after July 2026) on
`main`; `bootstrap.sh` resolves the two IDs with `gh`.
It needs five repository variables, all printed by `bootstrap.sh` on first
deploy:

| Variable | What it's for |
|---|---|
| `AWS_ROLE_ARN` | The GitHub deploy role from the foundation stack; assumed via OIDC to authenticate the workflow to AWS. |
| `AWS_REGION` | The AWS region everything is deployed to. |
| `ECR_REPOSITORY` | Name of the ECR repository the image is pushed to. |
| `SERVICE_STACK_NAME` | Name of the `service.yaml` CloudFormation stack to update. |
| `APP_RUNNER_ACCESS_ROLE_ARN` | The role App Runner assumes to pull the image from ECR, passed as a stack parameter. |

## Conventions

- **Commits:** [Conventional Commits](https://www.conventionalcommits.org)
  — `type(scope): description`, lowercase, e.g. `feat(backend): add the
  operation registry`, `fix(frontend): keep keypad focus during a
  calculation`.
- **Branches:** `type/short-description`, e.g. `feat/auth-flow`,
  `fix/dynamo-ttl` (types: `feat/`, `fix/`, `refactor/`, `chore/`, `docs/`,
  `test/`, `ci/`).
- **Formatting and linting:** `gofmt` + `go vet` on the backend, Biome +
  `tsc --noEmit` on the frontend — both run locally via `make lint` and in
  CI on every push.
- **Types live in exactly two places, both hand-mirrored from the
  contract:** `backend/internal/adapter/http/dto` on the Go side,
  `frontend/src/lib/api/types.ts` on the React side.
- **`api/openapi.yaml` changes first.** It is the contract both sides
  implement, not a byproduct of either implementation. A new operation, a
  new field, or a new error code starts as an edit to the OpenAPI spec;
  the Go DTOs and the frontend types are then updated to match it, not the
  other way around.

## License

[MIT](LICENSE) © SebasElDev
