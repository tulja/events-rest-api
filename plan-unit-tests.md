# Unit test plan — events-rest-api

**Document:** project-wide unit test plan (expanded from original `db/` plan).

## Status overview

| Package | Status | Run |
|---|---|---|
| `db` | **Implemented** | `go test ./db/ -count=1 -v` |
| `utils` | **Implemented** | `go test ./utils/ -count=1 -v` |
| `middlewares` | **Implemented** | `go test ./middlewares/ -count=1 -v` |
| `models` | **Implemented** | `go test ./models/ -count=1 -v` |
| `secrets` | **Implemented** | `go test ./secrets/ -count=1 -v` |
| `routes` | **Implemented** | `go test ./routes/ -count=1 -v` |

```bash
go test ./... -count=1
```

Requires **CGO** for `go-sqlite3` (db + routes).

---

## Shared conventions

| Principle | Choice |
|---|---|
| Prefer stdlib + gin test utils | `testing`, `net/http/httptest`, `gin` test mode |
| Prefer real logic over mocks | Real bcrypt; real JWT crypto; real in-memory SQLite |
| Vault | Mock with `httptest.Server` (KV v2 JSON), not a real Vault process |
| Production hooks | JWT inject/reset; `db.InitInMemory()` for cross-package tests |
| Never use `./events.db` | `:memory:` only |
| Parallelism | Avoid on packages with package-global state (`db`, JWT key) |

---

## Production test hooks

### `utils` JWT

- `SetJWTSigningKeyForTest(key []byte)` — inject key without Vault
- `ResetJWTSigningKeyForTest()` — clear cached key between tests
- Loader uses mutex + loaded flag (not sticky failed `sync.Once`)

### `db`

- `InitInMemory() error` — open `:memory:` SQLite, `MaxOpenConns(1)`, create tables  
  Used by `routes` tests (other packages cannot access unexported `db.db`).

---

## `db/` (implemented)

### Files

| File | Purpose |
|---|---|
| `db/test_helpers_test.go` | `setupTestDB`, `seedUser`, `seedEvent` |
| `db/users_test.go` | User insert / lookup |
| `db/events_test.go` | Event CRUD + ownership |
| `db/register_test.go` | Register / list / cancel |
| `db/db_test.go` | Schema idempotency |

### Approach

White-box tests (`package db`) against in-memory SQLite. No sqlmock.

### Key cases

- Users: success, duplicate email, not found, hash stored
- Events: CRUD, owner vs non-owner, not found
- Registrations: success, duplicate, list only own, cancel, cancel with no prior reg (nil)
- Schema: `createTables` twice

### Behavior locked in

1. Not-found / not-authorized domain errors as returned by package  
2. `InsertUser` stores bcrypt hash; returned struct may still hold plaintext password  
3. `DeleteRegistration` with no row still returns nil  
4. No FK cascade assertions (pragma not enabled)  
5. Never open `./events.db`

---

## `utils/` (implemented)

### Files

| File | Purpose |
|---|---|
| `utils/hash_test.go` | bcrypt generate/compare |
| `utils/jwt_test.go` | generate/verify with test key |
| `utils/jwt.go` | test inject/reset hooks |

### `hash` cases

| Test | Assert |
|---|---|
| `TestGenerateHash_Success` | non-empty, ≠ plaintext, bcrypt prefix |
| `TestGenerateHash_DifferentSalts` | two hashes differ |
| `TestCompareHash_Match` | `(true, nil)` |
| `TestCompareHash_Mismatch` | `(false, nil)` |
| `TestCompareHash_InvalidHash` | `(false, err)` |

### `jwt` cases

| Test | Assert |
|---|---|
| `TestGenerateAndVerifyToken_RoundTrip` | same userId |
| `TestVerifyToken_InvalidSignature` | error |
| `TestVerifyToken_Malformed` / empty | error |
| `TestGenerateToken_IncludesClaims` | email + exp present |

---

## `middlewares/` (implemented)

### Files

| File | Purpose |
|---|---|
| `middlewares/authentication_test.go` | `Authenticate` via gin test router |

### Cases

| Test | Assert |
|---|---|
| Missing `Authorization` | 401 Unauthorized |
| Invalid token | 401 |
| Valid token | 200 + `userId` in context/response |
| Wrong signing key | 401 |

Uses JWT test key hook; no Vault.

---

## `models/` (implemented)

### Files

| File | Purpose |
|---|---|
| `models/user_test.go` | JSON + binding |
| `models/events_test.go` | JSON + binding |
| `models/registrations_test.go` | JSON round-trip |

### Cases

- JSON marshal/unmarshal for `User`, `Event`, `Registration`
- Gin `ShouldBindJSON` required-field validation for User/Event

---

## `secrets/` (implemented)

### Files

| File | Purpose |
|---|---|
| `secrets/client_test.go` | `NewClient` + Get* against mock Vault |

### Cases

| Area | Tests |
|---|---|
| `NewClient` | missing token; config token/address; env token |
| `GetSecret` / `GetSecretValue` | success, 404, missing key, non-string |
| `MustGetSecretValue` | success; panics on error |

Mock: `httptest.Server` returning Vault KV v2 JSON at `/v1/secret/data/...`.

---

## `routes/` (implemented)

### Files

| File | Purpose |
|---|---|
| `routes/test_helpers_test.go` | router, DB, request helpers |
| `routes/users_test.go` | signup/login |
| `routes/events_test.go` | event HTTP CRUD |
| `routes/register_test.go` | registration HTTP |

### Approach

Same-package (`package routes`) integration-style tests:

1. `db.InitInMemory()`
2. `utils.SetJWTSigningKeyForTest(...)`
3. `RegisterRoutes(gin.New())`
4. `httptest` requests

### Cases (status codes match **current** handlers)

**Users:** signup 201; bad body 400; duplicate 500; login 200+token; bad creds 401  

**Events:** list empty 200; bad id 400; not found 500; create without auth 401; create 201; update owner 200; not owner 500; delete owner 200  

**Registrations:** register 200; duplicate 500; cancel 200; list 200; bad id 400  

---

## Implementation order (done)

1. Expand this document  
2. `utils/hash_test.go`  
3. JWT hooks + `utils/jwt_test.go`  
4. `middlewares/authentication_test.go`  
5. `models/*_test.go`  
6. `secrets/client_test.go`  
7. `db.InitInMemory` + `routes/*_test.go`  
8. `go test ./...`

---

## Verification

```bash
go test ./db/ ./utils/ ./middlewares/ ./models/ ./secrets/ ./routes/ -count=1 -v
go test ./... -count=1
go build .
```

**Success criteria**

- All packages pass  
- No real Vault required for unit tests  
- No writes to `./events.db`  
- Existing `db` tests still pass  

---

## Out of scope

- E2E against server on `:8080`  
- Real HashiCorp Vault in CI  
- Fixing HTTP 500 → 403/404 domain mapping  
- OpenAPI/contract tests  
- Frontend tests  

---

## Summary

| Package | Style | Prod changes |
|---|---|---|
| `db` | In-memory SQLite white-box | none (helpers in `_test.go`) |
| `utils` | Unit | JWT inject/reset |
| `middlewares` | Gin HTTP | none |
| `models` | JSON/bind | none |
| `secrets` | httptest Vault mock | none |
| `routes` | Integration unit | `db.InitInMemory` |
