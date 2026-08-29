# API Reference

Base URL (local): `http://localhost:8080`
All bodies are JSON. Responses below are **real captured output** from the
running service.

## Error envelope

Every error uses one shape:

```json
{ "error": { "code": "invalid_request", "message": "..." } }
```

| HTTP | `code` | When |
|---|---|---|
| 400 | `invalid_request` | body fails validation |
| 400 | `empty_update` | PATCH with no updatable fields |
| 401 | `unauthorized` | missing / invalid / expired token |
| 401 | `invalid_credentials` | wrong email or password on login |
| 404 | `not_found` | user does not exist |
| 409 | `email_taken` | email already registered |
| 500 | `internal_error` | unexpected failure |

---

## Public endpoints

### `GET /healthz`

```bash
curl -s localhost:8080/healthz
```
```json
{"status":"ok"}
```

### `POST /auth/register`

Creates a user via public sign-up.

| Field | Rules |
|---|---|
| `name` | required |
| `email` | required, valid email, stored lower-cased, unique |
| `password` | required, min 8 chars |

```bash
curl -s -X POST localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Alice","email":"alice@example.com","password":"password123"}'
```
`201 Created`
```json
{
  "id": "6a90627de2cb5fc59ab1e13a",
  "name": "Alice",
  "email": "alice@example.com",
  "created_at": "2026-08-27T16:14:53.146064045Z",
  "updated_at": "2026-08-27T16:14:53.146064045Z"
}
```

The password is **never** returned.

Duplicate email → `409`:
```json
{"error":{"code":"email_taken","message":"email already exists"}}
```

Invalid body → `400`:
```json
{"error":{"code":"invalid_request","message":"Key: 'registerRequest.Email' Error:Field validation for 'Email' failed on the 'email' tag"}}
```

### `POST /auth/login`

```bash
curl -s -X POST localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"password123"}'
```
`200 OK`
```json
{"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."}
```

Wrong credentials → `401`:
```json
{"error":{"code":"invalid_credentials","message":"invalid email or password"}}
```

---

## Protected endpoints (`Authorization: Bearer <token>`)

Without a token every route below returns `401`:
```json
{"error":{"code":"unauthorized","message":"missing or malformed Authorization header"}}
```

### `POST /api/users`

Same body and rules as `/auth/register` (admin-style create). `201 Created`,
same user object.

### `GET /api/users`

```bash
curl -s localhost:8080/api/users -H "Authorization: Bearer $TOKEN"
```
`200 OK`
```json
{
  "users": [
    {
      "id": "6a90627de2cb5fc59ab1e13a",
      "name": "Alice",
      "email": "alice@example.com",
      "created_at": "2026-08-27T16:14:53.146Z",
      "updated_at": "2026-08-27T16:14:53.146Z"
    }
  ]
}
```

### `GET /api/users/{id}`

`200 OK` with the user object, or `404`:
```json
{"error":{"code":"not_found","message":"user not found"}}
```

### `PATCH /api/users/{id}`

Partial update. Provide `name`, `email`, or both. Omitted fields are unchanged.

| Field | Rules |
|---|---|
| `name` | optional, min 1 char |
| `email` | optional, valid email, must not belong to another user |

```bash
curl -s -X PATCH localhost:8080/api/users/6a90627de2cb5fc59ab1e13a \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"Alice Cooper"}'
```
`200 OK`
```json
{
  "id": "6a90627de2cb5fc59ab1e13a",
  "name": "Alice Cooper",
  "email": "alice@example.com",
  "created_at": "2026-08-27T16:14:53.146Z",
  "updated_at": "2026-08-27T16:14:53.292Z"
}
```

Empty body `{}` → `400 empty_update`.
Email owned by another user → `409 email_taken`.

### `DELETE /api/users/{id}`

```bash
curl -s -o /dev/null -w '%{http_code}\n' -X DELETE \
  localhost:8080/api/users/6a90627de2cb5fc59ab1e13a \
  -H "Authorization: Bearer $TOKEN"
```
`204 No Content` (empty body). Unknown id → `404`.

---

## gRPC

Proto: [`api/proto/user/v1/user.proto`](../api/proto/user/v1/user.proto).
Server reflection is enabled, so `grpcurl` works without local `.proto` files.

```bash
# CreateUser
grpcurl -plaintext -d '{"name":"Bob","email":"bob@example.com","password":"password123"}' \
  localhost:9090 user.v1.UserService/CreateUser

# GetUser
grpcurl -plaintext -d '{"id":"<id>"}' localhost:9090 user.v1.UserService/GetUser
```

Response:
```json
{
  "id": "6a9063...",
  "name": "Bob",
  "email": "bob@example.com",
  "createdAt": "2026-08-27T16:20:00Z",
  "updatedAt": "2026-08-27T16:20:00Z"
}
```

With `GRPC_AUTH=true`, add `-H "authorization: Bearer $TOKEN"`. Domain errors map
to gRPC codes: `NotFound`, `AlreadyExists`, `Unauthenticated`, `Internal`.

---

## Background job

Every `USER_COUNT_INTERVAL` (default 10s) the server logs:
```json
{"time":"2026-08-27T16:15:19Z","level":"INFO","msg":"user count","total":1}
```

## Request logging

Every HTTP request is logged by middleware:
```json
{"time":"...","level":"INFO","msg":"http request","method":"POST","path":"/auth/register","status":201,"duration":83771667,"client_ip":"192.168.65.1"}
```
`duration` is nanoseconds.
