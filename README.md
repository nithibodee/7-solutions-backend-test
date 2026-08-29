# User Management API

A RESTful (and gRPC) user-management service in Go, backed by MongoDB, secured
with JWT (HS256), and structured with a hexagonal / ports-and-adapters layout.

Built for the **7 Solutions Backend Golang Coding Test**. The lottery search
design exercise lives in [`LOTTERY_DESIGN.md`](LOTTERY_DESIGN.md).

---

## Features

| Requirement | Where |
|---|---|
| User model (id, name, unique email, hashed password, timestamps) | `internal/domain/user` |
| Registration + login returning a JWT | `POST /auth/register`, `POST /auth/login` |
| JWT auth via middleware, HMAC-SHA256 | `internal/middleware/auth.go`, `internal/adapter/auth/jwt.go` |
| CRUD: create / get / list / update / delete | `/api/users*` (JWT-protected) |
| Official MongoDB Go driver, behind an interface | `internal/adapter/mongo`, port in `internal/domain/user/port.go` |
| Logging middleware (method, path, status, duration) | `internal/middleware/logging.go` |
| Background goroutine logging user count every 10s | `internal/app/user/counter.go` |
| Unit tests + mocked persistence | `*_test.go`, mocks in `test/mocks` (mockery) |
| **Bonus:** Docker + docker-compose | `deployments/Dockerfile`, `docker-compose.yml` |
| **Bonus:** interface abstraction over Mongo | `domain.Repository` port |
| **Bonus:** input validation | Gin binding tags in `internal/adapter/http/dto.go` |
| **Bonus:** graceful shutdown via `context.Context` | `cmd/server/main.go` (`signal.NotifyContext` + `errgroup`) |
| **Bonus:** gRPC `CreateUser` / `GetUser` (+ optional token metadata) | `api/proto/user/v1`, `internal/adapter/grpc` |
| **Bonus:** hexagonal architecture | see layout below |

---

## Architecture

```
cmd/server            entrypoint: config, wiring, start HTTP+gRPC+background job, graceful shutdown
internal/
  domain/user         entity + ports (Repository, PasswordHasher, TokenIssuer/Validator) — no deps
  app/user            use-cases (Service): Register, Authenticate, Create, Get, List, Update, Delete, Count
  adapter/
    http              Gin router, handlers, DTO validation, error mapping
    grpc              UserService server + optional auth interceptor
    mongo             Repository implementation on the official driver
    auth              bcrypt hasher, JWT (HS256) issuer/validator
  middleware          Gin logging + JWT auth
  platform            config (env), mongodb client, slog logger
api/proto/user/v1     .proto contract + generated code
test/mocks            generated mocks (mockery)
```

Dependencies point **inward**: adapters depend on the app/domain, never the
reverse. The HTTP and gRPC adapters call the exact same `app/user.Service`.

---

## Quick start (Docker)

```bash
docker compose up --build
```

This starts MongoDB and the API. HTTP on `:8080`, gRPC on `:9090`.
Override host ports if they clash: `API_HTTP_PORT=18080 API_GRPC_PORT=19090 docker compose up --build`.

Smoke test:

```bash
curl -s localhost:8080/healthz
curl -s -X POST localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Alice","email":"alice@example.com","password":"password123"}'
```

Full request/response reference: [`docs/API.md`](docs/API.md).
Ready-to-run **Bruno** collection: [`api/bruno/user-management`](api/bruno/README.md)
(`npx @usebruno/cli run api/bruno/user-management --env Local`).

---

## Run locally

Prerequisites: Go 1.25+, a running MongoDB (`docker run -p 27017:27017 mongo:7`).

```bash
cp .env.example .env          # then export the vars, or use a tool like direnv
export JWT_SECRET=dev-secret
go run ./cmd/server
```

### Configuration (environment variables)

| Var | Default | Notes |
|---|---|---|
| `HTTP_PORT` | `8080` | |
| `GRPC_PORT` | `9090` | |
| `MONGO_URI` | `mongodb://localhost:27017` | |
| `MONGO_DB` | `usermgmt` | |
| `JWT_SECRET` | — | **required**, no default |
| `JWT_ISSUER` | `user-management-api` | `iss` claim |
| `JWT_TTL` | `24h` | Go duration |
| `BCRYPT_COST` | `10` | |
| `USER_COUNT_INTERVAL` | `10s` | background count logger |
| `SHUTDOWN_TIMEOUT` | `10s` | graceful shutdown budget |
| `GRPC_AUTH` | `false` | require Bearer token in gRPC metadata |
| `LOG_LEVEL` | `info` | `debug`/`info`/`warn`/`error` |

---

## Tests

```bash
make test              # unit tests, race detector
make cover             # + coverage summary
make test-integration  # integration tests against a real MongoDB (build tag: integration)
```

Unit tests mock the `Repository` port (mockery). Core layers — `app`, `adapter/http`,
`adapter/auth`, `adapter/grpc`, `platform/config`, `middleware` — sit around 80%+
statement coverage. The Mongo adapter is covered by the tagged integration suite:

```bash
docker run -d -p 27017:27017 mongo:7
make test-integration
```

---

## Regenerating code

```bash
make mocks   # needs: go install github.com/vektra/mockery/v2@v2.53
make proto   # needs: protoc + protoc-gen-go + protoc-gen-go-grpc
```

---

## Documentation

- [`docs/JWT.md`](docs/JWT.md) — how tokens are generated, signed, and used
- [`docs/API.md`](docs/API.md) — every endpoint with sample requests & responses (HTTP + gRPC)
- [`docs/DESIGN.md`](docs/DESIGN.md) — assumptions and design decisions
- [`LOTTERY_DESIGN.md`](LOTTERY_DESIGN.md) — lottery ticket search design proposal
