# Bruno API collection

A ready-to-run [Bruno](https://www.usebruno.com/) collection for the User
Management API. Bruno stores requests as plain `.bru` files in git — no cloud
account, no sign-in.

## Open in the Bruno app

1. Install Bruno (`brew install --cask bruno`, or from usebruno.com).
2. **Open Collection** → pick `api/bruno/user-management`.
3. Top-right environment selector:
   - **Local** — `http://localhost:8080` (local run or `docker compose up`)
   - **Docker (alt ports)** — `http://localhost:18080` (compose started with
     `API_HTTP_PORT=18080 API_GRPC_PORT=19090`)
4. Run the requests top to bottom. `03 Login` captures the JWT into a runtime
   variable that every protected request reuses automatically.

## Run headless (CLI)

```bash
npx @usebruno/cli run api/bruno/user-management --env Local
```

Expected: `11 (11 Passed)`, `Assertions 17/17`, `Status ✓ PASS` against a fresh
database (`docker compose down -v && docker compose up`).

## Requests

| # | Request | Notes |
|---|---|---|
| 01 | Health | `GET /healthz` |
| 02 | Register | `201` fresh, `409` if already registered — both accepted |
| 03 | Login | captures `{{token}}` (runtime var, not written to disk) |
| 04 | List Users (no token → 401) | negative check |
| 05 | Create User | authenticated create |
| 06 | List Users | captures `{{userId}}` = first user |
| 07 | Get User by ID | |
| 08 | Update User | partial update (`name`) |
| 09 | Delete User | `204` |
| 10 | Register — validation error | `400 invalid_request` |
| 11 | Login — bad credentials | `401 invalid_credentials` |
