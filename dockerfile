# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache dependency downloads separately from source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO disabled: pgx's stdlib driver is pure Go, no libpq/cgo dependency.
# Static binary, smaller and portable across the final alpine stage.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.version=docker -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /app/bin/api ./cmd/api/

# ---- Run stage ----
FROM alpine:3.20

# ca-certificates: needed for any outbound TLS calls (none today, but cheap
# insurance — e.g. once webhooks or external integrations are added).
# wget: used by the docker-compose healthcheck to probe GET /health.
RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /app/bin/api ./api
COPY --from=builder /app/docs ./docs

EXPOSE 8080

ENTRYPOINT ["./api"]