# syntax=docker/dockerfile:1

# ---- stage: web -------------------------------------------------------------
# Builds the Vite/React SPA into frontend/dist.
FROM node:22-alpine AS web
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci
COPY frontend/ ./
RUN npm run build

# ---- stage: api ---------------------------------------------------------
# Builds the Go binary. TARGETPLATFORM/TARGETOS/TARGETARCH are populated
# automatically by buildx (e.g. `docker buildx build --platform linux/amd64`).
FROM golang:1.27-alpine AS api
ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY backend/ ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/api ./cmd/api

# ---- stage: runtime -----------------------------------------------------
# Distroless, non-root, single process: the Go binary serves the API and the
# static SPA assets from the same port.
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app
COPY --from=api /out/api /app/api
COPY --from=web /src/frontend/dist /app/public
ENV PORT=8080 \
    STATIC_DIR=/app/public
EXPOSE 8080
USER nonroot
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/app/api", "-healthcheck"]
ENTRYPOINT ["/app/api"]
