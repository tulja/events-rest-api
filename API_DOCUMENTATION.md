# API Documentation

> Generated from source. Last updated: 2026-07-14

## Overview

A lightweight REST API for creating and managing events, with user authentication (signup/login) and event registrations.

- **Base URL:** `http://localhost:8080`
- **Framework:** Gin (github.com/gin-gonic/gin)
- **API Version:** none detected
- **Database:** SQLite (`./events.db`, foreign keys enabled)
- **Authentication:** JWT (HS256) via `Authorization` header
- **Environments:** Local development only (port 8080)
- **Health:** `GET /health`

## Authentication

All protected routes require a valid JWT.

- **Method:** JWT (HS256)
- **Header:** `Authorization: Bearer <jwt>` **or** raw token `Authorization: <jwt>`
- **Note:** An optional `Bearer ` prefix is stripped case-insensitively before verification.
- **Obtaining credentials:** Use `POST /login` after signing up.
- **Startup:** JWT signing key is loaded from `JWT_SIGNING_KEY` (or `JWT_SECRET`) at process start (fail-fast).

**Example:**
```bash
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
     http://localhost:8080/events/registrations
```

**Token claims:**
- `user_id` (int64)
- `email` (string)
- `exp` (Unix timestamp, 24 hours)

## Request Format

- **Content-Type:** `application/json`
- **Accept:** `application/json`

## Error Format

All error responses follow this shape:

```json
{
  "error": "human-readable message"
}
```

## Endpoints

### Public Endpoints

#### `POST /signup`

Create a new user account.

**Auth required:** No

#### Request Body

```json
{
  "email": "user@example.com",
  "password": "secret123"
}
```

| Field    | Type   | Required | Description      |
|----------|--------|----------|------------------|
| email    | string | Yes      | User email       |
| password | string | Yes      | Plaintext password |

#### Success Response

**Status:** `201 Created`

```json
{
  "message": "Signup successful!"
}
```

#### Error Responses

| Status | Trigger                  | Response Body                     |
|--------|--------------------------|-----------------------------------|
| 400    | Invalid JSON / binding   | `{"error": "..."}`                |
| 409    | Email already exists     | `{"error": "Email already exists"}` |
| 500    | Database / hash failure  | `{"error": "..."}`                |

---

#### `POST /login`

Authenticate and receive a JWT.

**Auth required:** No

#### Request Body

```json
{
  "email": "user@example.com",
  "password": "secret123"
}
```

| Field    | Type   | Required | Description      |
|----------|--------|----------|------------------|
| email    | string | Yes      | User email       |
| password | string | Yes      | Plaintext password |

#### Success Response

**Status:** `200 OK`

```json
{
  "message": "Login successful!",
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

#### Error Responses

| Status | Trigger                     | Response Body                     |
|--------|-----------------------------|-----------------------------------|
| 400    | Invalid JSON / binding      | `{"error": "..."}`                |
| 401    | Invalid email or password   | `{"error": "Invalid credentials!"}` |
| 500    | Token generation failure    | `{"error": "could not authenticate user..."}` |

---

#### `GET /events`

List all events.

**Auth required:** No

#### Success Response

**Status:** `200 OK`

Array of events:

```json
[
  {
    "id": 1,
    "name": "Go Conference",
    "description": "Annual Go conference",
    "location": "San Francisco",
    "date_time": "2026-09-15T09:00:00Z",
    "user_id": 42
  }
]
```

#### Error Responses

| Status | Trigger          | Response Body          |
|--------|------------------|------------------------|
| 500    | Database error   | `{"error": "..."}`     |

---

#### `GET /events/:id`

Get a single event by ID.

**Auth required:** No

#### Path Parameters

| Parameter | Type  | Required | Description          |
|-----------|-------|----------|----------------------|
| id        | int64 | Yes      | Event ID             |

#### Success Response

**Status:** `200 OK`

```json
{
  "id": 1,
  "name": "Go Conference",
  "description": "Annual Go conference",
  "location": "San Francisco",
  "date_time": "2026-09-15T09:00:00Z",
  "user_id": 42
}
```

#### Error Responses

| Status | Trigger                  | Response Body                     |
|--------|--------------------------|-----------------------------------|
| 400    | Invalid ID format        | `{"error": "..."}`                |
| 500    | Not found or DB error    | `{"error": "Event not found!"}` or other |

---

### Authenticated Endpoints

All endpoints below require a valid JWT in the `Authorization` header.

#### `POST /events`

Create a new event. The authenticated user becomes the owner.

**Auth required:** Yes

#### Request Body

```json
{
  "name": "Go Conference",
  "description": "Annual Go conference",
  "location": "San Francisco",
  "date_time": "2026-09-15T09:00:00Z"
}
```

| Field       | Type      | Required | Description                  |
|-------------|-----------|----------|------------------------------|
| name        | string    | Yes      | Event name                   |
| description | string    | Yes      | Event description            |
| location    | string    | Yes      | Event location               |
| date_time   | string    | Yes      | ISO 8601 / RFC3339 timestamp |

#### Success Response

**Status:** `201 Created`

Returns the created event (including generated `id` and `user_id`).

```json
{
  "id": 1,
  "name": "Go Conference",
  "description": "Annual Go conference",
  "location": "San Francisco",
  "date_time": "2026-09-15T09:00:00Z",
  "user_id": 42
}
```

#### Error Responses

| Status | Trigger               | Response Body          |
|--------|-----------------------|------------------------|
| 400    | Validation error      | `{"error": "..."}`     |
| 500    | Database error        | `{"error": "..."}`     |

---

#### `PUT /events/:id`

Update an existing event. Only the owner (user who created it) is allowed.

**Auth required:** Yes (owner only)

#### Path Parameters

| Parameter | Type  | Required | Description |
|-----------|-------|----------|-------------|
| id        | int64 | Yes      | Event ID    |

#### Request Body

Same shape as create (partial updates not supported — all fields are required in the current implementation).

#### Success Response

**Status:** `200 OK`

Returns the updated event.

#### Error Responses

| Status | Trigger            | Response Body |
|--------|--------------------|---------------|
| 400    | Invalid ID or JSON | `{"error": "..."}` |
| 403    | Not owner          | `{"error": "You are not authorized to update this event"}` |
| 404    | Event not found    | `{"error": "Event not found"}` |
| 500    | Unexpected DB error | `{"error": "..."}` |

---

#### `DELETE /events/:id`

Delete an event. Only the owner is allowed.

**Auth required:** Yes (owner only)

#### Path Parameters

| Parameter | Type  | Required | Description |
|-----------|-------|----------|-------------|
| id        | int64 | Yes      | Event ID    |

#### Success Response

**Status:** `200 OK`

```json
{
  "message": "Event deleted successfully!"
}
```

#### Error Responses

| Status | Trigger | Response Body |
|--------|---------|---------------|
| 400    | Invalid ID | `{"error": "..."}` |
| 403    | Not owner | `{"error": "You are not authorized to delete this event"}` |
| 404    | Event not found | `{"error": "Event not found"}` |
| 500    | Unexpected DB error | `{"error": "..."}` |

---

#### `POST /events/:id/register`

Register the authenticated user for an event.

**Auth required:** Yes

#### Path Parameters

| Parameter | Type  | Required | Description |
|-----------|-------|----------|-------------|
| id        | int64 | Yes      | Event ID    |

#### Success Response

**Status:** `200 OK`

```json
{
  "message": "Registered for event successfully!"
}
```

#### Error Responses

| Status | Trigger | Response Body |
|--------|---------|---------------|
| 400    | Invalid event ID | `{"error": "..."}` |
| 404    | Event not found | `{"error": "Event not found"}` |
| 409    | Already registered | `{"error": "You have already registered for this event"}` |
| 500    | Unexpected DB error | `{"error": "..."}` |

---

#### `DELETE /events/:id/register`

Cancel the authenticated user's registration for an event.

**Auth required:** Yes

#### Path Parameters

| Parameter | Type  | Required | Description |
|-----------|-------|----------|-------------|
| id        | int64 | Yes      | Event ID    |

#### Success Response

**Status:** `200 OK`

```json
{
  "message": "Cancelled event registration successfully!"
}
```

#### Error Responses

| Status | Trigger | Response Body |
|--------|---------|---------------|
| 400    | Invalid ID | `{"error": "..."}` |
| 404    | Event or registration not found | `{"error": "Event not found"}` / `{"error": "Registration not found"}` |
| 500    | Unexpected DB error | `{"error": "..."}` |

---

#### `GET /events/registrations`

List all events the authenticated user is registered for.

**Auth required:** Yes

#### Success Response

**Status:** `200 OK`

Array of Event objects (same shape as `GET /events`).

---

## Pagination

No pagination is implemented. All list endpoints return complete result sets.

## Rate Limiting

No rate limiting middleware is present.

## Notes & Implementation Details

- **Ownership enforcement:** `PUT` and `DELETE` on `/events/:id` verify that `event.user_id` matches the `userId` from the JWT. The check happens in the DB layer (403 if not owner).
- **Registrations:** Unique constraint on (`event_id`, `user_id`). SQLite foreign keys are enabled; deleting an event cascades registrations.
- **Password handling:** Passwords are bcrypt-hashed on signup. The `password` field is never returned in responses.
- **JWT secret:** Loaded from the `JWT_SIGNING_KEY` environment variable (legacy alias: `JWT_SECRET`) at startup. Process exits if neither is set.
- **Date handling:** `date_time` uses Go's `time.Time` (serialized as RFC3339).
- **Error handling:** Domain errors map to 404 (not found), 403 (forbidden), 409 (conflict); unexpected failures remain 500.
- **Health:** `GET /health` returns `{"status":"ok"}` when SQLite is reachable.

## Changelog

| Date       | Version | Change | Breaking? |
|------------|---------|--------|-----------|
| 2026-07-14 | v1.1.0  | Bearer support; FK pragma; domain HTTP statuses; JWT at startup; `/health` | Yes (status codes + Bearer optional) |
| 2026-06-27 | v1.0.0  | Initial documentation generated from source | — |

