[![CI](https://github.com/louisealberti/onboarding-api/actions/workflows/ci.yml/badge.svg)](https://github.com/louisealberti/onboarding-api/actions/workflows/ci.yml)

# Customer Onboarding API

A production-grade REST API for customer onboarding in an international fintech. Built with Go, PostgreSQL, and JWT authentication — designed to demonstrate real-world backend engineering practices including layered architecture, state machines, audit logging, idempotency, and multi-level testing.

---

## Business Context

This API handles the full lifecycle of a customer in a payment/digital wallet product. A customer goes through a structured onboarding process governed by a state machine, from initial registration to activation — with compliance-oriented features like audit logging, soft delete, and tax ID validation for multiple jurisdictions (BR, US, GB).

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.21+ |
| Web framework | Gin |
| Database | PostgreSQL 16 (via `pgx/v5`) |
| Authentication | JWT RS256 (`golang-jwt/jwt/v5`) |
| Rate limiting | Token bucket (`golang.org/x/time/rate`) |
| Migrations | `golang-migrate` |
| Testing | `testify`, `testcontainers-go` |
| Documentation | Swagger/OpenAPI (`swaggo/swag`) |
| CI | GitHub Actions |

---

## Architecture

The project follows a layered architecture with clear separation of concerns:

```
cmd/api/
  main.go                   ← entry point, wires dependencies

internal/
  config/                   ← environment configuration
  database/                 ← PostgreSQL connection
  domain/                   ← entities, interfaces, state machine
  handler/                  ← HTTP handlers (Gin)
  middleware/               ← Auth, CORS, rate limit, logging, idempotency, HSTS
  repository/               ← database access (pgx)
  sanitize/                 ← free-text input sanitization (XSS defense-in-depth)
  service/                  ← business logic
  validation/               ← email and tax ID validators (CPF, CNPJ, SSN, EIN, NI, UTR)
  acceptance/                ← end-to-end tests (httptest + testcontainers)

db/migrations/              ← SQL migrations (golang-migrate)
docs/                       ← Swagger generated files
keys/                       ← RSA key pair (not committed)

Dockerfile                  ← multi-stage build for the API image
docker-compose.yml          ← API + PostgreSQL + migrations, for local development
```

**Request flow:** `Handler → Service → Repository → PostgreSQL`

---

## Customer State Machine

A customer transitions through the following statuses:

```
                    ┌─────────────────┐
                    │     pending      │
                    └────────┬────────┘
                             │ approved
                    ┌────────▼────────┐
                    │    approved      │
                    └────────┬────────┘
                             │ active
                    ┌────────▼────────┐
              ┌────►│     active       │◄────┐
              │     └────────┬────────┘     │
          active             │ suspended    │ active
              │     ┌────────▼────────┐     │
              └─────│   suspended     │─────┘
                    └────────┬────────┘
                             │
              blocked ◄──────┼──────► terminated
```

Any status can transition to `blocked` or `terminated`. Blocked customers cannot be deleted.

---

## API Endpoints

All `/v1/*` endpoints require a Bearer JWT token. Swagger UI is available at `http://localhost:8080/swagger/index.html`.

### Authentication
| Method | Path | Role | Description |
|---|---|---|---|
| POST | `/auth/token` | public | Generate JWT token |

### Health
| Method | Path | Role | Description |
|---|---|---|---|
| GET | `/health` | public | API and database status + build version |

### Customers
| Method | Path | Role | Description |
|---|---|---|---|
| POST | `/v1/customers` | admin | Create customer |
| GET | `/v1/customers` | admin, operator | List customers (paginated) or search by taxId |
| GET | `/v1/customers/:id` | admin, operator | Get customer by ID |
| PUT | `/v1/customers/:id` | admin | Update customer data |
| PATCH | `/v1/customers/:id/status` | admin, operator | Transition customer status |
| DELETE | `/v1/customers/:id` | admin | Soft delete customer |
| GET | `/v1/customers/:id/audit` | admin, operator | Get audit log |

### Query parameters for `GET /v1/customers`
- `?taxId=<value>` — search by tax ID (bypasses pagination)
- `?page=1&limit=20` — paginated listing
- `?status=approved` — filter by status (must be one of the known state-machine statuses; unrecognized values return `400`)

---

## Security

### JWT RS256
- Tokens signed with 2048-bit RSA private key
- API validates with the public key only — private key never leaves the auth service
- Claims: `sub` (subject/email), `role` (`admin` or `operator`), `exp` (24h)
- `sub` is injected into the request context and used by the audit log

### Role-based authorization
- `admin` — full access (CRUD + status transitions + audit)
- `operator` — read-only + status transitions (cannot create, update, or delete)

### Rate limiting (token bucket)
- Public routes (`/health`, `/auth/token`) — 10 req/s per IP, burst 20
- Protected routes (`/v1/*`) — 20 req/s per JWT subject, burst 40

### CORS
Configurable via `CORS_ORIGINS` environment variable. Defaults to `*` in development.

### Input sanitization
Free-text fields (`firstName`, `lastName`, address fields) are sanitized in the service layer before being persisted: trimmed, stripped of control characters, HTML-escaped against XSS, and length-capped. SQL injection is mitigated structurally — every repository query uses parameterized placeholders (`$1`, `$2`, ...) via `database/sql`/`pgx`, never string concatenation. The `status` filter on `GET /v1/customers` is validated against the known state-machine enum rather than passed through as an arbitrary string.

---

## Key Features

### Idempotency
`POST /v1/customers` supports the `Idempotency-Key` header. Duplicate requests with the same key within 24h return the original response without re-processing. Stored in PostgreSQL with automatic expiry.

### Audit Log
Every customer creation, data update, and status change is recorded in `audit_logs` with the previous state (`old_value`), new state (`new_value`), timestamp, and the actor (`changed_by` from the JWT `sub` claim). Queryable via `GET /v1/customers/:id/audit`.

### Tax ID Validation
- **BR**: CPF (individuals) and CNPJ (companies) — full checksum validation
- **US**: SSN and EIN — format and range validation
- **GB**: NI and UTR — format and check digit validation

### Soft Delete
Customers are never hard-deleted. `deleted_at` is set on deletion and all queries filter `WHERE deleted_at IS NULL`. A soft-deleted customer's email and tax ID become available for re-registration.

### Optimistic Concurrency
The `version` field is incremented on every update. This prevents lost updates in concurrent modification scenarios.

### Structured Logging
All requests are logged as JSON with `slog` (stdlib): method, path, status, latency, request ID, and client IP.

---

## Database Migrations

| # | Migration |
|---|---|
| 001 | Create `customers`, `addresses`, `phones` tables |
| 002 | Add partial index on `customers.status` for filtered listing |
| 003 | Create `idempotency_keys` table |
| 004 | Create `audit_logs` table |

---

## Running Locally

### Option 1: Docker Compose (recommended)

Spins up the full stack — API + PostgreSQL — and runs all migrations automatically. No local Go or PostgreSQL installation required, only Docker.

```bash
git clone https://github.com/louisealberti/onboarding-api
cd onboarding-api
make docker-up
```

`make docker-up` generates the RSA key pair (`keys/`) if it doesn't exist yet, then builds the API image and starts everything in the background. The API will be available at `http://localhost:8080`, with Swagger UI at `http://localhost:8080/swagger/index.html`.

```bash
make docker-logs   # tail logs from API + Postgres
make docker-down   # stop everything (Postgres data persists in a named volume)
```

Environment variables for the containers (DB credentials, CORS origins) can be overridden via a `.env` file in the project root — `docker compose` reads it automatically. See [Environment Variables](#environment-variables) below; the same variable names apply.

#### Troubleshooting

**`database files are incompatible with server` / Postgres container exits on startup.**
The named volume `postgres_data` already has data from a different PostgreSQL major version (e.g. a previous local setup running v15). Reset it:
```bash
docker compose down -v
make docker-up
```

**`port is already allocated` on `5432`.**
Another PostgreSQL instance (local install, another project's container) is already using that port. By default this compose file does **not** publish `5432` to the host — only `api` and `migrate` reach it internally — so this should only come up if you uncommented the `ports:` block under `postgres` in `docker-compose.yml`. Either stop the other instance or keep that port unpublished.

### Option 2: Run Go directly

### Prerequisites
- Go 1.21+
- Docker (for PostgreSQL and tests)
- `golang-migrate` CLI
- `swag` CLI (for Swagger regeneration)

### Setup

```bash
# 1. Clone and install dependencies
git clone https://github.com/louisealberti/onboarding-api
cd onboarding-api
go mod tidy

# 2. Generate RSA key pair
mkdir -p keys
openssl genrsa -out keys/private.pem 2048
openssl rsa -in keys/private.pem -pubout -out keys/public.pem
echo "keys/" >> .gitignore

# 3. Configure environment
cp .env.example .env
# Edit .env with your database credentials

# 4. Run migrations
migrate -path db/migrations -database "postgres://user:pass@localhost:5432/onboarding?sslmode=disable" up

# 5. Start the server
make run
```

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `DB_HOST` | — | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | — | Database user |
| `DB_PASSWORD` | — | Database password |
| `DB_NAME` | — | Database name |
| `DB_SSLMODE` | `disable` | SSL mode |
| `SERVER_PORT` | `8080` | HTTP port |
| `JWT_PRIVATE_KEY_PATH` | `keys/private.pem` | RSA private key path |
| `JWT_PUBLIC_KEY_PATH` | `keys/public.pem` | RSA public key path |
| `CORS_ORIGINS` | `*` | Allowed CORS origins |

---

## Testing

```bash
# All tests
make test

# Unit tests only (no Docker required)
make test-unit

# Repository integration tests
make test-integration

# Acceptance tests (end-to-end)
make test-acceptance

# With coverage
go test ./internal/service/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

### Test strategy
- **Unit tests** — service layer with mocks, no I/O
- **Integration tests** — repository layer against real PostgreSQL via `testcontainers-go`
- **Acceptance tests** — full stack via `httptest.Server` + real PostgreSQL, covering happy paths, error cases, state machine flows, idempotency, auth, and rate limiting

### CI
Every push and pull request against `main`/`develop` runs two GitHub Actions jobs: `vet` + `build` + unit tests (no Docker), and integration + acceptance tests (Docker-in-Docker, required by `testcontainers-go`). See `.github/workflows/ci.yml`.

---

## Build with version info

```bash
make build
# binary at bin/api

# Or manually:
go build \
  -ldflags "-X main.version=$(git rev-parse --short HEAD) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o bin/api ./cmd/api/
```

The `GET /health` endpoint returns the injected version and build time.

---

## Generating a token (example)

```bash
# Get a token
curl -s -X POST http://localhost:8080/auth/token \
  -H "Content-Type: application/json" \
  -d '{"sub": "admin@fintech.com", "role": "admin"}' | jq .

# Use the token
curl -s http://localhost:8080/v1/customers \
  -H "Authorization: Bearer <token>" | jq .
```

---

## Technical Decisions

**Why layered architecture instead of hexagonal?**
The project is a focused REST API with a single external dependency (PostgreSQL). Hexagonal architecture would add abstraction overhead without meaningful benefit at this scope. The layered approach keeps the code readable and the dependency flow clear: handlers depend on services, services depend on repository interfaces, repositories depend on the database.

**Why soft delete?**
In a regulated fintech environment, customer data must be retained for compliance purposes (LGPD, anti-money laundering regulations). Hard deletion would violate audit requirements. Soft delete via `deleted_at` preserves the record while making it invisible to business logic queries.

**Why RS256 over HS256?**
With asymmetric signing, only the authentication service needs access to the private key. The API validates tokens with the public key alone — meaning even if the API server is compromised, the attacker cannot forge tokens. In a microservices environment this is the standard approach.

**Why idempotency keys in PostgreSQL instead of Redis?**
Keeping idempotency keys in the same database as customers ensures consistency within the same transaction boundary. Redis would introduce a second dependency and a potential consistency gap. At this scale, PostgreSQL with a TTL index on `expires_at` is sufficient.

**Why `log/slog` over zerolog/zap?**
`slog` is part of the Go standard library since 1.21. Zero external dependencies, JSON output out of the box, and the API surface is intentionally simple. For this project's observability needs it is more than sufficient, and it avoids adding a dependency that would require explanation to reviewers.

**Why does `docker-compose.yml` only include the API and PostgreSQL?**
Redis and Kafka are listed under Next Steps but have no corresponding code yet — no cache reads/writes, no event publishing or consumption. Adding those containers to the compose file now would stand up infrastructure with nothing using it, which is misleading about the project's actual state. Each will be added to the compose file in the same change that introduces its first real usage in the code.

---

## Next Steps

- [ ] Webhook — notify external systems on status changes
- [ ] Deploy on AWS (ECS + RDS), with HTTPS/TLS terminated at the ALB (ACM certificate + HTTPS listener)
- [ ] Redis — cache for frequent taxId/ID lookups
- [ ] Prometheus metrics + Grafana dashboard
- [ ] gRPC integration with Ledger service
- [ ] Kafka — domain events (depends on a concrete use case being implemented first, not just the broker)