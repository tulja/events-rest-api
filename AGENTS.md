# AGENTS.md — events-rest-api

Project rules and context for the Events REST API (Go + Gin).

## Project Overview

A lightweight REST API for creating and managing events with user authentication and registrations.

- **Language**: Go 1.26+ (`go.mod` uses `go 1.26`; local toolchain may be 1.26.x)
- **Module**: `events-rest-api`
- **HTTP Framework**: Gin (`github.com/gin-gonic/gin`)
- **Database**: SQLite (via pure-Go `modernc.org/sqlite`, no CGO). Path: `DATABASE_PATH` / `SQLITE_PATH`, else `/tmp/events.db` on Vercel, else `./events.db`
- **Auth**: JWT (golang-jwt/jwt/v5) + bcrypt password hashing
- **Secrets**: JWT signing key required at startup via env — `JWT_SIGNING_KEY` (or legacy `JWT_SECRET`)
- **Port**: 8080

Core domain:
- Users (signup/login)
- Events (owned by users)
- Event registrations (many-to-many between users and events)

## Development Setup & Commands

### 1. Set the JWT signing key

```bash
export JWT_SIGNING_KEY="your-dev-signing-key"
# optional legacy alias:
# export JWT_SECRET="your-dev-signing-key"
```

### 2. Run the API

```bash
go run main.go
# or
./events-rest-api
```

Server starts on `http://localhost:8080`.

### 3. Useful Go commands

```bash
make build   # go build -o events-rest-api .
make run     # go run .  (JWT_SIGNING_KEY required)
make test    # go test ./... -count=1  (no CGO required; modernc.org/sqlite)
make test-v  # verbose tests
make fmt
make tidy
make clean
```

Unit tests cover `db`, `utils`, `middlewares`, `models`, and `routes`. See `plan-unit-tests.md`.

## Architecture & File Layout

```
main.go                 # Entry point: InitDB() + Gin server + routes
routes/
  routes.go             # Route registration + auth grouping
  events.go             # Event CRUD (create/update/delete enforce ownership)
  users.go              # Signup + Login (returns JWT)
  register.go           # Register/cancel + list my registrations
db/
  db.go                 # DB connection + table creation (IF NOT EXISTS)
  events.go             # Event queries + ownership checks
  users.go              # User insert + lookup by email
  register.go           # Registration logic (prevent duplicate)
models/
  events.go             # Event struct (with time.Time and binding tags)
  user.go               # User struct
  registrations.go      # Registration struct (lightly used)
middlewares/
  authentication.go     # JWT middleware — sets "userId" in gin context
utils/
  jwt.go                # GenerateToken / VerifyToken (loads key from env)
  hash.go               # bcrypt helpers
```

### Critical Implementation Details

- **Authentication**: `Authorization: Bearer <jwt>` or raw JWT both accepted (`Bearer ` prefix stripped case-insensitively).
- **Authorization**: `updateEvent` and `deleteEvent` fetch the event and compare `eventFromDb.UserID != userIdFromToken`.
- **Ownership**: Every created event stores the creator's `userId` from the JWT.
- **JWT**: Signing key loaded at startup via `utils.EnsureJWTSigningKey()` (fail-fast) from env `JWT_SIGNING_KEY` or `JWT_SECRET`. Successful loads are cached; failures are not sticky so retries can succeed after the env is set.
- **Database**: Raw SQL only. Positional `?` parameters. `PRAGMA foreign_keys = ON`. Rows closed with `defer`.
- **Error responses**: Shape `{ "error": "message" }` or `{ "message": "..." }` + optional data/token. Domain errors map to 404/403/409 where applicable.
- **Registrations**: Unique constraint on `(event_id, user_id)`. Event delete cascades registrations (FK pragma on).
- **Health**: `GET /health` pings SQLite.

## Coding Conventions

- Follow standard Go style (`gofmt` / `goimports`).
- Gin handlers receive `*gin.Context` (commonly named `ginContext` in this codebase).
- Always bind with `ShouldBindJSON` and return early on error.
- Prefer `gin.H{"message": "..."}` for simple success responses.
- Parse IDs with `strconv.ParseInt(..., 10, 64)`.
- Keep SQL close to the db/ package; do not scatter queries in routes.
- New features should follow existing patterns (models → db funcs → route handlers).

## When Working in This Codebase

- Always set `JWT_SIGNING_KEY` (or `JWT_SECRET`) before starting the API. On Vercel: set `JWT_SIGNING_KEY` in Project → Settings → Environment Variables.
- When adding protected routes, place them under the `authenticated` group.
- When modifying events, replicate the ownership check pattern.
- Do not commit `events.db` or real secrets (the current db file contains dev data).
- Pre-built binaries (`events-rest-api`, `events-api`) exist at root — treat as artifacts.

## Missing / Future

- No OpenAPI-driven contract tests / Swagger runtime
- No CI workflow yet
- No input validation beyond Gin's binding tags

## Verification

After changes:
- `go build .` must succeed with no errors
- Manually test critical flows: signup → login → create event → register → update (as owner)
