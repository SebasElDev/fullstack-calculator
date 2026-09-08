# fullstack-calculator — root Makefile
# Entry points for local dev, testing, linting, building and AWS deployment.
SHELL := /bin/sh

IMAGE_NAME    ?= fullstack-calculator
VERSION       := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
AWS_REGION    ?= us-east-1
PROJECT_NAME  ?= fullstack-calculator

.PHONY: help dev dev-api dev-web install test test-backend test-frontend lint build docker-build docker-run bootstrap deploy destroy

help: ## Show this help
	@echo "fullstack-calculator — available targets:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_-]+:.*##/ { printf "  %-14s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

dev: ## Run backend and frontend dev servers concurrently
	$(MAKE) -j2 dev-api dev-web

dev-api: ## Run the Go API in dev mode
	cd backend && go run ./cmd/api

dev-web: ## Run the Vite dev server
	cd frontend && npm run dev

install: ## Install all dependencies (frontend + backend)
	cd frontend && npm ci
	cd backend && go mod download

test: test-backend test-frontend ## Run backend and frontend test suites

test-backend: ## Run Go tests with race detector and coverage
	cd backend && go test -race -cover ./...

test-frontend: ## Run frontend tests
	cd frontend && npm run test

lint: ## Lint and typecheck both backend and frontend
	cd backend && gofmt -l . && go vet ./...
	cd frontend && npx biome check . && npx tsc --noEmit

build: ## Build production frontend assets and the Go binary
	cd frontend && npm run build
	cd backend && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/api ./cmd/api

docker-build: ## Build the production Docker image
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE_NAME):$(VERSION) -t $(IMAGE_NAME):latest .

docker-run: ## Run the production image locally on :8080
	docker run --rm -p 8080:8080 --name $(IMAGE_NAME) $(IMAGE_NAME):latest

bootstrap: ## First-time AWS provisioning (foundation + service stacks)
	AWS_REGION=$(AWS_REGION) PROJECT_NAME=$(PROJECT_NAME) ./deploy/aws/bootstrap.sh

deploy: ## Build, push and deploy the current commit to AWS
	AWS_REGION=$(AWS_REGION) PROJECT_NAME=$(PROJECT_NAME) ./deploy/aws/deploy.sh

destroy: ## Tear down all AWS resources for this project
	AWS_REGION=$(AWS_REGION) PROJECT_NAME=$(PROJECT_NAME) ./deploy/aws/destroy.sh
