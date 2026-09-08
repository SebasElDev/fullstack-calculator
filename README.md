# fullstack-calculator

A small full-stack calculator built to demonstrate a few patterns end to end,
cleanly: **Clean Architecture** on a Go backend, **Compound Components +
Context Providers** on a React 19 frontend, a hand-maintained **OpenAPI
contract** between them, and a **single container** as the deployment unit.
**No arithmetic in the frontend.**

[![CI](https://github.com/SebasElDev/fullstack-calculator/actions/workflows/ci.yml/badge.svg)](https://github.com/SebasElDev/fullstack-calculator/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/go-1.27-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-blue)


## Live demo

<https://vegmwcswz2.us-east-1.awsapprunner.com> — deployed on AWS App Runner from the `main` branch.

```sh
curl https://vegmwcswz2.us-east-1.awsapprunner.com/health
# {"status":"ok","version":"<git-sha>"}
```

## Stack

| Layer | Technology | Version |
|---|---|---|
| Backend language | Go | 1.27 |
| Backend web framework | [Fiber](https://gofiber.io) v3 | v3.5.0 |
| Frontend framework | React | 19.2.8 |
| Frontend build tool | Vite | 8.2.2 |
| Frontend language | TypeScript | ~6.0.2 |
| CSS | Tailwind CSS | 4.3.3 |
| Frontend testing | Vitest + Testing Library + jsdom | 4.1.11 |
| Lint / format (frontend) | [Biome](https://biomejs.dev) | 2.5.12 |
| Container runtime | `gcr.io/distroless/static-debian12:nonroot` | — |
| Deployment | AWS App Runner + CloudFormation | — |

### Backend: Clean Architecture

| Layer | Package(s) |
|---|---|
| Entities | `internal/domain` |
| Use cases | `internal/usecase/calculator` |
| Interface adapters | `internal/adapter/http/dto`, `internal/adapter/http/fiber` |
| Frameworks & drivers | `internal/infrastructure/config`, `cmd/api` |

### Frontend: Compound Component + two Context Providers

`Calculator` is a compound component:

```tsx
<ApiClientProvider client={apiClient}>
  <Calculator>
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
├── api/openapi.yaml
├── backend/ 
│   ├── cmd/api/
│   └── internal/
│       ├── domain/  
│       ├── usecase/calculator/
│       ├── adapter/http/ 
│       └── infrastructure/config/
├── frontend/ 
│   └── src/
│       ├── lib/api/   
│       ├── lib/operations.ts   
│       └── features/calculator/
├── deploy/aws/
├── Dockerfile 
├── compose.yaml
├── Makefile
└── .github/workflows/ci.yml
```

## How to run

### Prerequisites

- Go 1.27
- Node 22+ and npm
- Docker (only to build/run the production image)
- AWS CLI and the `gh` CLI (only for deployment — see [Deployment](#deployment))

Run `make help` for a list of available targets

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
is optional):

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | TCP port the HTTP server listens on. |
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

## API usage

Full schema, including request/response examples for every status code, is
in [`api/openapi.yaml`](api/openapi.yaml).

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

## Design specifications

- **Result normalisation.** `domain.NewResult` rounds every result to 15
  significant digits.

- **Keyboard support.** Digits, `.`, `+ - * /`, `Enter`/`=`, `Backspace`,
  `Escape` all map onto the same `KeyAction` the on-screen keys dispatch
  (`state/keyboard.ts`).

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

## Deployment

### Single-image strategy

The production artifact is one Docker image built by a three-stage
`Dockerfile`:

1. `node:22-alpine` — `npm ci && npm run build` → `frontend/dist`.
2. `golang:1.27-alpine` — cross-compiled on the build platform
3. `gcr.io/distroless/static-debian12:nonroot` — just the Go binary and
   `frontend/dist` copied in at `/app/public`, running as `nonroot`, with a
   `HEALTHCHECK` that calls `/app/api -healthcheck`.

### AWS layout

Two CloudFormation stacks, deployed from `deploy/aws/`:

- **`foundation.yaml`** — the account-level, rarely-changing pieces: the
  ECR repository, the IAM role App Runner assumes to pull from it, and — optionally, since an account
  can only have one — a GitHub OIDC provider plus a deploy role scoped to the `main` branch, with permissions only to
  push to this one ECR repo and manage this one App Runner service.
- **`service.yaml`** — the App Runner service itself,
  listening on port 8080, health check on `GET /health`. A deploy is a CloudFormation update of
  this stack with a new `ImageUri` (the commit SHA tag).

Every resource in both stacks is tagged `Project=fullstack-calculator`, so
the whole deployment is trivially identifiable and isolated from any other
workload in the same AWS account — and torn down by exactly one command.

### First deploy

```sh
make bootstrap
```

Runs `deploy/aws/bootstrap.sh`, which: checks whether the AWS account
already has an OIDC provider and reuses it if so; deploys the foundation
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

### Cost

An App Runner instance provisioned at 0.25 vCPU / 0.5 GB, running
continuously, costs on the order of a few US dollars per month; There is no other billed
resource.

### CI/CD

`.github/workflows/ci.yml` runs `backend`, `frontend`, and `docker` on every push and pull request. The `deploy` job runs only on
a push to `main`, after the other three succeed, and authenticates to AWS
with GitHub OIDC. The
deploy role trusts only this repository's immutable OIDC subject on
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

## License

[MIT](LICENSE) © SebasElDev
