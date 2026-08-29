# JWT Guide

The API uses **JSON Web Tokens** signed with **HMAC-SHA256 (HS256)** for
authentication. All `/api/*` routes require a valid token; `/auth/*` and
`/healthz` are public.

## 1. The secret

Tokens are signed and verified with a single shared secret from the
`JWT_SECRET` environment variable. The process refuses to start if it is unset.

```bash
export JWT_SECRET="a-long-random-string"
```

Use a strong, random value in production (e.g. `openssl rand -base64 48`) and
keep it out of version control.

## 2. Getting a token

### Step 1 — register a user (once)

```bash
curl -s -X POST localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Alice","email":"alice@example.com","password":"password123"}'
```

### Step 2 — log in to receive a token

```bash
curl -s -X POST localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"password123"}'
```

```json
{ "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI..." }
```

## 3. Using the token

Send it as a Bearer token in the `Authorization` header:

```bash
TOKEN="<paste token>"
curl -s localhost:8080/api/users -H "Authorization: Bearer $TOKEN"
```

Missing/malformed header or an invalid/expired token → `401 Unauthorized`:

```json
{ "error": { "code": "unauthorized", "message": "invalid or expired token" } }
```

## 4. Token contents

The payload (claims):

| Claim | Meaning |
|---|---|
| `sub` | user ID (Mongo ObjectID hex) |
| `email` | user email |
| `iss` | issuer, from `JWT_ISSUER` (default `user-management-api`) |
| `iat` | issued-at (Unix seconds) |
| `exp` | expiry — `iat + JWT_TTL` (default 24h) |

Decode it for inspection (never trust an unverified token in code):

```bash
echo "$TOKEN" | cut -d. -f2 | base64 -d 2>/dev/null | jq
```

## 5. Validation rules (server side)

`internal/adapter/auth/jwt.go`:

- Signing method **must** be HMAC; the parser is pinned to `HS256` via
  `jwt.WithValidMethods` — an `alg: none` or `RS256` token is rejected.
- Signature must verify against `JWT_SECRET`.
- `exp` must be in the future (checked with an injectable clock so it is testable).
- `sub` must be non-empty.

## 6. gRPC

When `GRPC_AUTH=true`, the same token must be supplied as gRPC metadata:

```
authorization: Bearer <token>
```

A unary interceptor (`internal/adapter/grpc/interceptor.go`) validates it with
the same `TokenValidator`. With `GRPC_AUTH=false` (default) the gRPC methods are
open, which keeps local experimentation simple.
