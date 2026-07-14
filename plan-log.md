# Logging Audit — events-rest-api

Review document: where logs can be added and which log level to use.

**Date:** 2026-07-14  
**Scope:** Full codebase audit (no code changes in this document)

---

## Current state

There is **no structured logging**. What exists today:

| Location | Current behavior |
|---|---|
| `main.go` | `gin.Default()` — Gin’s built-in HTTP access logger + recovery middleware |
| `routes/events.go` | `fmt.Print` userId; `fmt.Println("Creating event ....")` |
| `routes/users.go` | `fmt.Println("Creating user ....")` / `"Logging in user ...."` |
| `db/events.go` | `fmt.Print(event)` after insert |
| `db/db.go` | `panic(err)` on open/schema failure (no prior log) |
| `secrets/client.go` | Errors returned only; panics via `MustGetSecretValue` |
| `utils/jwt.go` | Errors returned only (Vault load, verify, sign) |
| `middlewares/authentication.go` | Silent 401 responses |

**Problems with current `fmt` prints:**

- No levels (cannot filter noise in production)
- No structure (hard to search/aggregate)
- Can print whole structs (e.g. event payload)
- Inconsistent and easy to leave behind as debug noise

**Recommendation:** Use stdlib `log/slog` (Go 1.21+; project is on Go 1.26.3). No new dependencies.

---

## Log level guide

| Level | When to use in this API |
|---|---|
| **DEBUG** | High-volume diagnostics: handler entered, list counts, auth success with userId. Off by default in production. |
| **INFO** | Normal lifecycle and successful mutations: server start, DB ready, signup/login OK, event CRUD OK, registration OK/cancel. |
| **WARN** | Expected client/security issues: bad JSON, invalid id, missing/invalid JWT, bad credentials, ownership denial, not found, already registered. |
| **ERROR** | Unexpected server failures: SQL errors, bcrypt failure, Vault/JWT infra failures, HTTP server crash. |
| **FATAL** | Unrecoverable startup (DB open, schema create). Log ERROR then `panic` / `os.Exit(1)`. |

### What never to log

- Passwords or password hashes  
- Full `Authorization` header or JWT string  
- Vault token  
- Secret values from Vault  
- Prefer IDs + outcome over dumping full request/response bodies  

---

## Placement map by file

### 1. `main.go`

| Point | Level | Suggested message / fields |
|---|---|---|
| Logger configured | INFO | `logger initialized`, `level` |
| After `db.InitDB()` | INFO | `database ready` |
| Before listen | INFO | `starting HTTP server`, `addr=":8080"` |
| `server.Run` error | ERROR | `HTTP server failed`, `err` |

Notes:

- Keep Gin’s access logger for HTTP request lines, **or** later replace with slog middleware.
- Configure global slog once at process start (`LOG_LEVEL` env recommended).

---

### 2. `db/db.go` — startup only

| Point | Level | Suggested message / fields |
|---|---|---|
| `sql.Open` OK | INFO | `opened sqlite database`, `path="./events.db"` |
| `sql.Open` fail | ERROR → fatal | `failed to open database`, `err` |
| Table create fail | ERROR → fatal | `failed to create table`, `table`, `err` |
| All tables OK | INFO | `database schema ready` |

**Do not** log every query success in the db package (noise + double-logging with handlers).

---

### 3. `db/events.go`, `db/users.go`, `db/register.go`

| Point | Level | Action |
|---|---|---|
| `InsertEvent` `fmt.Print(event)` | — | **Remove** (or replace with DEBUG at handler, not here) |
| SQL / domain errors | — | Return to handler; **log at route layer** |
| Optional exception | ERROR | Only if a helper swallows an error (none do today) |

---

### 4. `secrets/client.go`

| Point | Level | Suggested message / fields |
|---|---|---|
| `NewClient` success | INFO | `vault client created`, `address` only (**never** token) |
| Missing token / client create fail | ERROR | Message in returned error; log at JWT load site |
| Secret/key missing | ERROR | path + key name only (no secret value) |

Primary logging site for Vault failures is `utils/jwt.go` `loadJWTSigningKey` (first use / cache).

---

### 5. `utils/jwt.go`

| Point | Level | Suggested message / fields |
|---|---|---|
| First successful key load | INFO | `JWT signing key loaded from Vault`, path `events-api/jwt` |
| Vault client / secret fail | ERROR | `failed to load JWT signing key`, `err` |
| `GenerateToken` sign fail | ERROR | `failed to sign JWT`, `userId` |
| `VerifyToken` invalid/expired | DEBUG (or none) | Common client issue — middleware should WARN |
| Unexpected signing method | WARN | `unexpected JWT signing method`, `alg` |

---

### 6. `utils/hash.go`

| Point | Level | Suggested message / fields |
|---|---|---|
| `GenerateHash` bcrypt error | ERROR | `failed to hash password` (no password) |
| `CompareHash` mismatch | — | Handler logs WARN for failed login |

---

### 7. `middlewares/authentication.go`

| Point | Level | Suggested message / fields |
|---|---|---|
| Empty `Authorization` | WARN | `missing authorization header`, `method`, `path` |
| `VerifyToken` fails | WARN | `invalid or expired token`, `method`, `path` (not the token) |
| Success | DEBUG | `authenticated`, `userId` |

---

### 8. `routes/users.go`

| Handler | Point | Level | Safe fields |
|---|---|---|---|
| `createUser` | `ShouldBindJSON` fail | WARN | `err` |
| `createUser` | insert / hash fail | ERROR | `email`, `err` |
| `createUser` | success | INFO | `userId` and/or `email` |
| `login` | bind fail | WARN | `err` |
| `login` | user not found or bad password | WARN | `email` only (same outcome message; never password) |
| `login` | JWT generate fail | ERROR | `userId`, `err` |
| `login` | success | INFO | `userId`, `email` |

**Remove:** `fmt.Println("Creating user ....")`, `fmt.Println("Logging in user ....")`.

---

### 9. `routes/events.go`

| Handler | Point | Level | Safe fields |
|---|---|---|---|
| Any | invalid `:id` parse | WARN | `err` |
| create/update | bind JSON fail | WARN | `err` |
| `createEvent` | insert fail | ERROR | `userId`, `err` |
| `createEvent` | success | INFO | `eventId`, `userId` |
| `getEvents` | query fail | ERROR | `err` |
| `getEvents` | success | DEBUG | `count` |
| `getEventById` | not found | WARN | `eventId`, `err` |
| `getEventById` | unexpected SQL | ERROR | `eventId`, `err` |
| `updateEvent` / `deleteEvent` | not found / not owner | WARN | `eventId`, `userId`, `err` |
| `updateEvent` / `deleteEvent` | unexpected SQL | ERROR | `eventId`, `userId`, `err` |
| `updateEvent` / `deleteEvent` | success | INFO | `eventId`, `userId` |

**Remove:** `fmt.Print("User ID: ...")`, `fmt.Println("Creating event ....")`.

**Note:** Ownership and not-found currently return HTTP 500 from handlers. Still log those as **WARN** (expected business/authz outcomes), not ERROR. Fixing status codes (403/404) is separate from logging.

---

### 10. `routes/register.go`

| Handler | Point | Level | Safe fields |
|---|---|---|---|
| register/cancel | invalid `:id` | WARN | `err` |
| `registerForEvent` | event not found / already registered | WARN | `eventId`, `userId`, `err` |
| `registerForEvent` | unexpected DB error | ERROR | `eventId`, `userId`, `err` |
| `registerForEvent` | success | INFO | `eventId`, `userId` |
| `cancelEventRegistration` | not found / fail | WARN or ERROR | same rule: domain → WARN, SQL → ERROR |
| `cancelEventRegistration` | success | INFO | `eventId`, `userId` |
| `getAllRegistrations` | fail | ERROR | `userId`, `err` |
| `getAllRegistrations` | success | DEBUG | `userId`, `count` |

---

### 11. `routes/routes.go`

No logging needed (route registration only). Optional DEBUG at boot: “routes registered” — low value.

---

### 12. Models (`models/*`)

No logging — pure data structs.

---

## Summary table (quick scan)

| Layer | DEBUG | INFO | WARN | ERROR |
|---|---|---|---|---|
| Startup (`main`, `db.InitDB`) | — | server/DB ready | — | open/schema/listen fail |
| Vault / JWT key | — | client + key loaded | bad alg | Vault/secret/sign fail |
| Auth middleware | auth OK | — | missing/invalid token | — |
| Users routes | — | signup/login OK | bind, bad creds | insert, JWT gen |
| Events routes | list count | create/update/delete OK | bind, bad id, not found, not owner | SQL failures |
| Register routes | list count | register/cancel OK | bad id, not found, duplicate | SQL failures |
| `db/*` queries | — | — | — | avoid (log in routes) |
| `hash` | — | — | — | bcrypt generate fail |

---

## Implementation sketch (if you implement later)

```go
// main.go — once at process start
level := slog.LevelInfo
switch os.Getenv("LOG_LEVEL") {
case "debug":
    level = slog.LevelDebug
case "warn":
    level = slog.LevelWarn
case "error":
    level = slog.LevelError
}
slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
```

Example handler logs:

```go
slog.Info("event created", "eventId", event.ID, "userId", userId)
slog.Warn("unauthorized event update", "eventId", id, "userId", userIdFromToken)
slog.Error("failed to insert event", "userId", userId, "err", err)
```

- Prefer logging **once** at the HTTP boundary for request-scoped failures.
- Replace all `fmt.Print*` with slog or delete them.
- No new module deps required for logging.

---

## Files that would change on implementation

| File | Change type |
|---|---|
| `main.go` | slog setup + lifecycle logs |
| `db/db.go` | startup INFO/ERROR |
| `db/events.go` | remove `fmt.Print` |
| `secrets/client.go` | optional connect INFO |
| `utils/jwt.go` | key load INFO/ERROR |
| `utils/hash.go` | bcrypt ERROR |
| `middlewares/authentication.go` | WARN/DEBUG |
| `routes/users.go` | full path logs; remove fmt |
| `routes/events.go` | full path logs; remove fmt |
| `routes/register.go` | full path logs |

---

## Verification checklist (after implementation)

1. `go build .` succeeds.  
2. Startup shows: logger init → DB ready → server starting.  
3. First login/protected call shows JWT key loaded (INFO) if not preloaded.  
4. Bad login → WARN (no password in output).  
5. Missing auth header → WARN.  
6. Create/update/delete event → INFO with ids.  
7. Ownership denial / not found → WARN.  
8. Forced SQL/Vault failure → ERROR.  
9. `LOG_LEVEL=debug` shows DEBUG; default `info` does not.  
10. Grep logs: no tokens, passwords, or Vault secrets.

---

## Out of scope (unless requested later)

- OpenTelemetry / log aggregation shipping  
- Request-ID correlation middleware  
- Replacing Gin access logs entirely  
- Changing 500 → 403/404 for ownership/not-found  
- Unit tests specifically for logging  

---

## Bottom line

| Priority | Action |
|---|---|
| P0 | Remove `fmt.Print*` debug dumps |
| P0 | ERROR on infra failures (DB, Vault, JWT sign, unexpected SQL) |
| P1 | WARN on auth failures and client validation errors |
| P1 | INFO on startup + successful domain mutations |
| P2 | DEBUG for auth OK and list counts |
| P2 | `LOG_LEVEL` env to control verbosity |
