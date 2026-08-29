# Design Decisions & Assumptions

## Architecture: hexagonal / ports & adapters

- **`domain`** — the `User` entity, domain errors, and ports (interfaces). Zero
  third-party imports. This is the stable core.
- **`app`** — use-cases (`Service`). Orchestrates ports, owns business rules
  (email normalisation, uniqueness pre-check, credential verification). Depends
  only on `domain`.
- **`adapter`** — everything that talks to the outside world: HTTP (Gin), gRPC,
  MongoDB, bcrypt, JWT. Each implements or consumes a domain port.
- **`platform`** — process-level plumbing: config, Mongo client, logger.

The HTTP and gRPC adapters both call the same `app.Service`. Swapping MongoDB for
Postgres, or Gin for another router, touches one adapter and nothing else.

## Key decisions

| Decision | Rationale |
|---|---|
| **Gin** for HTTP | Fast, ubiquitous, built-in request binding + validation (`go-playground/validator`). Kept at the edge — handlers are thin and the core has no Gin import. |
| **`Repository` interface** in the domain | Satisfies the "abstract MongoDB" bonus and makes the app layer unit-testable with a generated mock instead of a live DB. |
| **mockery** for mocks | Declarative config (`.mockery.yaml`), typed `EXPECT()` API, regenerated via `make mocks`. |
| **Unique index on `email`** + app-level pre-check | The index is the real guard against the check-then-insert race; the pre-check just yields a friendlier `409` on the common path. Duplicate-key errors from Mongo are also mapped to `ErrEmailAlreadyExists`. |
| **bcrypt** for passwords | Standard, salted, adjustable cost (`BCRYPT_COST`; tests use cost 4 for speed). |
| **JWT HS256** | Required by the brief. Parser pinned to HMAC/`HS256` to prevent algorithm-confusion attacks. Injectable clock for testing expiry. |
| **`slog` JSON logs** | Structured, stdlib, no dependency. Logging middleware records method, path, status, duration, client IP. |
| **`errgroup` + `signal.NotifyContext`** | One cancellation signal fans out to the HTTP server, gRPC server, and background ticker; a dedicated goroutine runs `httpServer.Shutdown` / `grpcServer.GracefulStop` within `SHUTDOWN_TIMEOUT`. |
| **Background counter as a plain goroutine** | `time.Ticker` + `select` on `ctx.Done()`; wrapped in `recover` so a transient DB error can't kill it. Uses the `Service.Count` use-case, not the repo directly. |
| **gRPC on a separate port** with reflection | Reuses `app.Service`; reflection lets `grpcurl` work out of the box. Auth is opt-in via `GRPC_AUTH`. |
| **Config from env only** | 12-factor; `.env.example` documents every key. Only `JWT_SECRET` has no default and hard-fails. |
| **distroless final image**, non-root, static binary | Small attack surface, no shell, `CGO_ENABLED=0`. |

## Assumptions

1. **`/auth/register` is public**; `POST /api/users` is the authenticated
   "admin creates a user" path. Business rules are identical today but the two
   entry points are kept separate so they can diverge (e.g. roles, email
   verification) without breaking callers.
2. **No roles / RBAC.** Any authenticated user may call any `/api/users` route.
   The brief doesn't mention authorization tiers; adding one is a middleware +
   claim change.
3. **Hard delete.** `DELETE` removes the document. No soft-delete/audit trail was
   requested.
4. **Email is the identity.** Stored lower-cased and trimmed; comparisons and the
   unique index use the normalised form.
5. **`id` is the Mongo ObjectID hex string.** Exposed as-is in the API. A
   malformed id is treated as `not_found` rather than `400`.
6. **List is unpaginated.** Fine for the test's scale; a real deployment would
   add cursor/skip-limit pagination.
7. **Single shared JWT secret**, symmetric (HS256). No key rotation / JWKS.
8. **Password minimum is 8 characters.** No composition rules; bcrypt handles the
   rest.
9. **Timestamps are UTC**, set by the application (not the DB), so behaviour is
   deterministic in tests.
10. **The integration test drops its database** (`usermgmt_it`) on cleanup, so it
    needs a disposable MongoDB, not a shared one.

## Testing strategy

- **Unit** (default `go test ./...`): every use-case and adapter with the
  `Repository` / `Service` ports mocked. Covers happy paths and each error
  mapping. Race detector on.
- **Assertion library:** tests use `testify` (`assert` / `require` / `mock`) on
  top of the standard `testing` package — every test is still a plain
  `func TestX(t *testing.T)`. `testify` is the de-facto standard in the Go
  ecosystem and `mockery` generates the `Repository` / `Service` mocks from it.
- **Integration** (`-tags=integration`): the real Mongo adapter against a live
  MongoDB — CRUD round-trip, duplicate-key handling, not-found.
- **End-to-end**: `docker compose up` + the curl sequence in `docs/API.md`
  (also used to capture the sample responses).

## Trade-offs / what a production version would add

- Pagination + filtering on `List`.
- Refresh tokens / short-lived access tokens; secret rotation.
- Rate limiting on `/auth/*`.
- Request IDs propagated through logs and gRPC metadata.
- Metrics (Prometheus) and tracing (OTel) — the logger seam is already there.
- `testcontainers` to make the integration suite self-provisioning in CI.
