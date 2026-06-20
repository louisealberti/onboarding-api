.PHONY: build run swagger tidy test test-unit test-integration test-acceptance keys docker-up docker-down docker-logs

VERSION := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

## build: compile the binary with version info
build:
	go build $(LDFLAGS) -o bin/api ./cmd/api/

## run: run the API locally
run:
	go run $(LDFLAGS) ./cmd/api/

## swagger: generate swagger docs (requires swag: go install github.com/swaggo/swag/cmd/swag@latest)
swagger:
	swag init -g cmd/api/main.go -o docs

## tidy: tidy and vendor dependencies
tidy:
	go mod tidy

## test: run all tests
test:
	go test ./... -v

## test-unit: run only unit tests (no containers)
test-unit:
	go test ./internal/service/... ./internal/validation/... -v

## test-integration: run repository integration tests
test-integration:
	go test ./internal/repository/... -v

## test-acceptance: run end-to-end acceptance tests
test-acceptance:
	go test ./internal/acceptance/... -v

## keys: generate the RSA key pair used for JWT signing (idempotent — skips if already present)
keys:
	@if [ -f keys/private.pem ] && [ -f keys/public.pem ]; then \
		echo "keys/ already populated, skipping"; \
	else \
		mkdir -p keys; \
		openssl genrsa -out keys/private.pem 2048; \
		openssl rsa -in keys/private.pem -pubout -out keys/public.pem; \
	fi

## docker-up: generate keys (if needed) and start the full local stack (API + Postgres) in the background
docker-up: keys
	docker compose up --build -d

## docker-down: stop the local stack and remove containers (keeps the Postgres volume)
docker-down:
	docker compose down

## docker-logs: tail logs from all services
docker-logs:
	docker compose logs -f