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
  metrics/                   ← Prometheus metrics definitions and DB pool collector
  middleware/               ← Auth, CORS, rate limit, logging, metrics, idempotency
  repository/               ← database access (pgx)
  sanitize/                 ← free-text input sanitization (XSS defense-in-depth)
  service/                  ← business logic
  validation/               ← email and tax ID validators (CPF, CNPJ, SSN, EIN, NI, UTR)
  webhook/                   ← status-change notifications (HMAC-signed, async, retried)
  acceptance/                ← end-to-end tests (httptest + testcontainers)

db/migrations/              ← SQL migrations (golang-migrate)
docs/                       ← Swagger generated files
keys/                       ← RSA key pair (not committed)

Dockerfile                  ← multi-stage build for the API image
docker-compose.yml          ← API + PostgreSQL + Prometheus + Grafana, for local development
monitoring/                  ← Prometheus scrape config + Grafana provisioning/dashboard

infra/                       ← Terraform: AWS deployment (ECS, RDS, ALB, CloudFront, ECR)
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

### Webhook
When `WEBHOOK_URL` is configured, every status transition (`PATCH /v1/customers/:id/status`) sends a `POST` notification to that URL with a `customer.<newStatus>` event (e.g. `customer.approved`, `customer.blocked`):

```json
{
  "event": "customer.approved",
  "customerId": "5f2c1e3a-...",
  "oldStatus": "pending",
  "newStatus": "approved",
  "changedBy": "admin@fintech.com",
  "occurredAt": "2026-06-19T22:14:03Z"
}
```

If `WEBHOOK_SECRET` is also set, the request body is signed with HMAC-SHA256 in the `X-Webhook-Signature` header (hex-encoded) — the same pattern GitHub and Stripe use — so the receiver can verify the notification actually came from this API.

Delivery is fire-and-forget and runs in a background goroutine, detached from the original request: up to 3 attempts with a short backoff, and a failing or unreachable destination is logged but never affects the outcome of the status-update request itself (the transition has already succeeded in the database by the time delivery is attempted). Leaving `WEBHOOK_URL` empty disables webhooks entirely — no notifier is constructed, no overhead is added.

### Metrics
`GET /metrics` exposes Prometheus-format metrics, scraped by the `prometheus` service in Docker Compose and visualized in the pre-provisioned Grafana dashboard (`http://localhost:3000`, no login needed locally). Three groups are tracked:

- **HTTP** — `http_requests_total{method,path,status}` and `http_request_duration_seconds{method,path}`, recorded for every request by `middleware.Metrics`. The `path` label uses the route *pattern* (e.g. `/v1/customers/:id`), not the literal URL, so cardinality stays bounded regardless of how many distinct customer IDs are ever requested.
- **Business** — `customers_created_total`, `customer_status_transitions_total{from,to}`, `webhook_deliveries_total{outcome}` (success/failure, after all retries), and `idempotent_replays_total`. These are incremented directly in the service/middleware layer at the point each event actually happens, not inferred from HTTP status codes.
- **Database connection pool** — `db_connections_open`, `db_connections_in_use`, `db_connections_idle`, and `db_wait_count_total`, sourced from `sql.DB.Stats()` and polled every 5 seconds (see `metrics.StartDBStatsCollector`).

Go runtime metrics (`go_goroutines`, `go_memstats_*`, GC pauses) come for free from the Prometheus client library's default registry — no custom code needed for those.

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

Spins up the full stack — API + PostgreSQL + Prometheus + Grafana — and runs all migrations automatically. No local Go or PostgreSQL installation required, only Docker.

```bash
git clone https://github.com/louisealberti/onboarding-api
cd onboarding-api
make docker-up
```

`make docker-up` generates the RSA key pair (`keys/`) if it doesn't exist yet, then builds the API image and starts everything in the background:

- API: `http://localhost:8080`, Swagger UI at `http://localhost:8080/swagger/index.html`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000` (no login required locally — anonymous Viewer access; see [Metrics](#metrics)). The "Customer Onboarding API" dashboard is provisioned automatically.

```bash
make docker-logs   # tail logs from all services
make docker-down   # stop everything (Postgres/Prometheus/Grafana data persists in named volumes)
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
| `WEBHOOK_URL` | — | Destination URL for status-change notifications; empty disables webhooks |
| `WEBHOOK_SECRET` | — | HMAC-SHA256 signing secret for `X-Webhook-Signature`; optional even when `WEBHOOK_URL` is set |

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

## Deploy (AWS)

The infrastructure for a production deployment — ECS Fargate behind an Application Load Balancer, RDS PostgreSQL, and CloudFront for HTTPS termination — is fully defined in Terraform under [`infra/`](infra/), but **not yet applied**. Development is currently focused on application features; the AWS deployment is the last item on the roadmap, by design — see [`infra/README.md`](infra/README.md) for setup and deploy instructions when that time comes.

```
Internet
   │  HTTPS (*.cloudfront.net)
   ▼
CloudFront  ──── HTTP (inside AWS network) ────►  ALB  ────►  ECS Fargate (API)  ────►  RDS PostgreSQL
```

Key choices, expanded in [Technical Decisions](#technical-decisions):
- **No NAT Gateway** — ECS and RDS both run in public subnets with no internet route other than the Internet Gateway, avoiding the ~US$32/month fixed cost of a NAT Gateway. Network isolation is enforced via security groups instead (RDS only accepts connections from ECS's security group).
- **CloudFront instead of a custom domain** — gets real HTTPS (a valid certificate for `*.cloudfront.net`) without buying and managing a domain.
- **Secrets via SSM Parameter Store** — DB password, JWT keys, and the webhook secret are stored as `SecureString` parameters and injected into the ECS task as environment variables at startup; they never appear in the Terraform state in plaintext at the root module (sensitive variables) nor in the task definition shown by `aws ecs describe-task-definition`.
- **GitHub Actions deploys via OIDC** — no AWS access keys stored as GitHub secrets; the deploy workflow assumes a narrowly-scoped IAM role (push to this project's ECR repo, update this project's ECS service — nothing else) using short-lived, per-run credentials.

CI/CD: once deployed, every merge to `main` re-runs the test suite, then builds the image, pushes it to ECR tagged with the commit SHA, and forces a new ECS deployment (`.github/workflows/deploy.yml`).

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

**Why fire-and-forget HTTP for webhooks instead of a message queue?**
A handful of retries over plain HTTP is appropriate for *notifying* a single external system about a status change — it's simple, requires no extra infrastructure, and the worst case (destination down, all retries exhausted) is an acceptable trade-off for a non-critical side effect. A message broker (Kafka, listed under Next Steps) would be the right tool if delivery needed to be guaranteed, replayed, or fanned out to multiple independent consumers — none of which is the case yet. Reaching for Kafka before a second consumer exists would be solving a problem the project doesn't have.

**Why does RDS have a public IP instead of sitting in a private subnet?**
This is the one infrastructure trade-off in this project that would not pass a real production security review, and it's worth being upfront about why it's here anyway: a private subnet for RDS needs a NAT Gateway (or VPC endpoints for every AWS service ECS talks to) for ECS to reach the internet — for pulling images from ECR — which adds a fixed ~US$32/month, disproportionate for a low-traffic portfolio API. The mitigation actually doing the work here is the security group, not network placement: RDS only accepts inbound PostgreSQL connections from ECS's security group, never from `0.0.0.0/0` — a public IP is assigned, but the port is not reachable from the internet. In a real fintech production environment, the private-subnet-plus-NAT (or VPC endpoints) approach would be the correct call regardless of cost.

**Why CloudFront instead of a custom domain with ACM?**
ACM requires a domain you control to issue a certificate — there's no way to get a trusted HTTPS certificate for the ALB's own `*.elb.amazonaws.com` hostname. Buying a domain is cheap (a few dollars a year) and is the more standard setup, but CloudFront was chosen here to get real HTTPS at zero additional cost, accepting an unmemorable `*.cloudfront.net` URL as the trade-off. This is admittedly not the textbook use of CloudFront (a CDN for cacheable content, not an HTTPS shim for a dynamic API) — caching is explicitly disabled in the distribution config for exactly this reason, so it doesn't introduce stale-response bugs.

**Why label HTTP metrics by route pattern instead of the literal path?**
`c.FullPath()` returns `/v1/customers/:id`, not `/v1/customers/550e8400-...` — using the pattern keeps the number of distinct label combinations (cardinality) bounded by the number of routes, not by the number of customer IDs ever requested. Prometheus performance degrades badly with high-cardinality labels; using the literal path would have been a correctness bug disguised as more detail.

**Why poll `sql.DB.Stats()` on a timer instead of updating the gauges on every query?**
The connection pool's state (open/in-use/idle connections) isn't tied to any single request — it's a property of the pool as a whole, sampled at a point in time. Updating it from inside every repository call would mean touching unrelated code paths just to refresh a number that changes on its own between requests anyway. A periodic poller (every 5s) is simpler, keeps the metrics code fully decoupled from the repository layer, and is precise enough for a dashboard meant for trend-spotting, not split-second alerting.

---

## Next Steps

- [ ] Redis — cache for frequent taxId/ID lookups
- [ ] gRPC integration with Ledger service
- [ ] Kafka — domain events (depends on a concrete use case being implemented first, not just the broker)