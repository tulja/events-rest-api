#!/usr/bin/env python3
"""
End-to-end API automation tests for the Events REST API.

Covers all endpoints documented in API_DOCUMENTATION.md against
http://localhost:8080 (override with BASE_URL env var).

Usage:
    # Ensure the API is running on :8080 first
    pip install -r scripts/requirements-api-tests.txt
    python scripts/test_api_automation.py

    # Or with a custom base URL:
    BASE_URL=http://127.0.0.1:8080 python scripts/test_api_automation.py
"""

from __future__ import annotations

import os
import sys
import time
import traceback
import uuid
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any, Callable, Optional

try:
    import requests
except ImportError:
    print("Missing dependency: requests")
    print("Install with: pip install requests")
    sys.exit(1)


# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

BASE_URL = os.environ.get("BASE_URL", "http://localhost:8080").rstrip("/")
TIMEOUT = float(os.environ.get("API_TEST_TIMEOUT", "10"))


# ---------------------------------------------------------------------------
# Test harness
# ---------------------------------------------------------------------------

@dataclass
class TestResult:
    name: str
    passed: bool
    detail: str = ""
    duration_ms: float = 0.0


@dataclass
class SuiteReport:
    results: list[TestResult] = field(default_factory=list)

    def add(self, result: TestResult) -> None:
        self.results.append(result)
        status = "PASS" if result.passed else "FAIL"
        line = f"  [{status}] {result.name} ({result.duration_ms:.0f} ms)"
        if result.detail and not result.passed:
            line += f"\n         {result.detail}"
        print(line)

    @property
    def passed_count(self) -> int:
        return sum(1 for r in self.results if r.passed)

    @property
    def failed_count(self) -> int:
        return sum(1 for r in self.results if not r.passed)


class APIClient:
    """Thin wrapper around requests with base URL and auth helpers."""

    def __init__(self, base_url: str = BASE_URL) -> None:
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers.update(
            {
                "Content-Type": "application/json",
                "Accept": "application/json",
            }
        )
        self.token: Optional[str] = None

    def set_token(self, token: Optional[str], *, bearer: bool = True) -> None:
        self.token = token
        if not token:
            self.session.headers.pop("Authorization", None)
            return
        if bearer:
            self.session.headers["Authorization"] = f"Bearer {token}"
        else:
            self.session.headers["Authorization"] = token

    def clear_auth(self) -> None:
        self.set_token(None)

    def request(
        self,
        method: str,
        path: str,
        *,
        expected_status: Optional[int] = None,
        json: Any = None,
        headers: Optional[dict[str, str]] = None,
    ) -> requests.Response:
        url = f"{self.base_url}{path}"
        resp = self.session.request(
            method,
            url,
            json=json,
            headers=headers,
            timeout=TIMEOUT,
        )
        if expected_status is not None and resp.status_code != expected_status:
            body_preview = (resp.text or "")[:500]
            raise AssertionError(
                f"{method} {path}: expected HTTP {expected_status}, "
                f"got {resp.status_code}. Body: {body_preview}"
            )
        return resp

    def get(self, path: str, **kwargs: Any) -> requests.Response:
        return self.request("GET", path, **kwargs)

    def post(self, path: str, **kwargs: Any) -> requests.Response:
        return self.request("POST", path, **kwargs)

    def put(self, path: str, **kwargs: Any) -> requests.Response:
        return self.request("PUT", path, **kwargs)

    def delete(self, path: str, **kwargs: Any) -> requests.Response:
        return self.request("DELETE", path, **kwargs)

    def json(self, resp: requests.Response) -> Any:
        try:
            return resp.json()
        except ValueError as exc:
            raise AssertionError(
                f"Expected JSON response, got: {(resp.text or '')[:300]}"
            ) from exc


def unique_email(prefix: str = "e2e") -> str:
    return f"{prefix}-{uuid.uuid4().hex[:12]}@example.com"


def sample_event_payload(
    name: str = "E2E Test Event",
    description: str = "Automated test event",
    location: str = "Test City",
    date_time: Optional[str] = None,
) -> dict[str, str]:
    if date_time is None:
        date_time = datetime(2026, 9, 15, 9, 0, 0, tzinfo=timezone.utc).isoformat().replace(
            "+00:00", "Z"
        )
    return {
        "name": name,
        "description": description,
        "location": location,
        "date_time": date_time,
    }


def assert_error_shape(body: Any) -> None:
    assert isinstance(body, dict), f"Error body should be object, got {type(body)}"
    assert "error" in body, f"Error body missing 'error' key: {body}"
    assert isinstance(body["error"], str) and body["error"], "error message empty"


def assert_event_shape(event: Any, *, require_id: bool = True) -> None:
    assert isinstance(event, dict), f"Event should be object, got {type(event)}"
    for key in ("name", "description", "location", "date_time", "user_id"):
        assert key in event, f"Event missing '{key}': {event}"
    if require_id:
        assert "id" in event and event["id"], f"Event missing id: {event}"
    assert "password" not in event


def as_list(body: Any, *, context: str = "response") -> list[Any]:
    """Normalize API list payloads.

    Go's encoding/json serializes a nil slice as JSON null, so empty list
    endpoints may return null instead of []. Treat both as a list.
    """
    if body is None:
        return []
    assert isinstance(body, list), f"{context}: expected list or null, got {type(body)}: {body}"
    return body


# ---------------------------------------------------------------------------
# Individual tests
# ---------------------------------------------------------------------------

def test_health_ok(client: APIClient) -> None:
    resp = client.get("/health", expected_status=200)
    body = client.json(resp)
    assert body.get("status") == "ok", f"Unexpected health body: {body}"


def test_signup_success(client: APIClient, email: str, password: str) -> None:
    resp = client.post(
        "/signup",
        json={"email": email, "password": password},
        expected_status=201,
    )
    body = client.json(resp)
    assert body.get("message") == "Signup successful!", body


def test_signup_duplicate_email(client: APIClient, email: str, password: str) -> None:
    resp = client.post(
        "/signup",
        json={"email": email, "password": password},
        expected_status=409,
    )
    body = client.json(resp)
    assert_error_shape(body)
    assert "already exists" in body["error"].lower(), body


def test_signup_invalid_body(client: APIClient) -> None:
    resp = client.post("/signup", json={"email": "only-email@example.com"}, expected_status=400)
    assert_error_shape(client.json(resp))

    resp = client.post("/signup", json={"password": "no-email"}, expected_status=400)
    assert_error_shape(client.json(resp))


def test_login_success(client: APIClient, email: str, password: str) -> str:
    resp = client.post(
        "/login",
        json={"email": email, "password": password},
        expected_status=200,
    )
    body = client.json(resp)
    assert body.get("message") == "Login successful!", body
    token = body.get("token")
    assert token and isinstance(token, str), f"Missing token: {body}"
    return token


def test_login_invalid_credentials(client: APIClient, email: str) -> None:
    resp = client.post(
        "/login",
        json={"email": email, "password": "definitely-wrong-password"},
        expected_status=401,
    )
    body = client.json(resp)
    assert_error_shape(body)
    assert "invalid credentials" in body["error"].lower(), body

    resp = client.post(
        "/login",
        json={"email": "nobody-exists-" + uuid.uuid4().hex + "@example.com", "password": "x"},
        expected_status=401,
    )
    assert_error_shape(client.json(resp))


def test_login_invalid_body(client: APIClient) -> None:
    resp = client.post("/login", json={"email": "a@b.com"}, expected_status=400)
    assert_error_shape(client.json(resp))


def test_list_events_public(client: APIClient) -> None:
    client.clear_auth()
    resp = client.get("/events", expected_status=200)
    body = as_list(client.json(resp), context="GET /events")
    for event in body:
        assert_event_shape(event)


def test_get_event_not_found(client: APIClient) -> None:
    client.clear_auth()
    # Large ID unlikely to exist; domain error may be 404 or 500 per older docs,
    # current API maps domain not-found to 404.
    resp = client.get("/events/999999999")
    assert resp.status_code in (404, 500), f"Unexpected status: {resp.status_code}"
    body = client.json(resp)
    assert_error_shape(body)


def test_get_event_invalid_id(client: APIClient) -> None:
    client.clear_auth()
    resp = client.get("/events/not-a-number", expected_status=400)
    assert_error_shape(client.json(resp))


def test_create_event_requires_auth(client: APIClient) -> None:
    client.clear_auth()
    resp = client.post("/events", json=sample_event_payload())
    assert resp.status_code in (401, 403), f"Expected unauthorized, got {resp.status_code}"


def test_create_event_success(client: APIClient, token: str) -> dict[str, Any]:
    client.set_token(token, bearer=True)
    payload = sample_event_payload(
        name="E2E Conference",
        description="Annual automated conference",
        location="San Francisco",
    )
    resp = client.post("/events", json=payload, expected_status=201)
    event = client.json(resp)
    assert_event_shape(event)
    assert event["name"] == payload["name"]
    assert event["description"] == payload["description"]
    assert event["location"] == payload["location"]
    assert event["user_id"] is not None
    return event


def test_create_event_validation(client: APIClient, token: str) -> None:
    client.set_token(token)
    resp = client.post("/events", json={"name": "incomplete"}, expected_status=400)
    assert_error_shape(client.json(resp))


def test_get_event_by_id(client: APIClient, event_id: int) -> None:
    client.clear_auth()
    resp = client.get(f"/events/{event_id}", expected_status=200)
    event = client.json(resp)
    assert_event_shape(event)
    assert event["id"] == event_id


def test_auth_raw_token_header(client: APIClient, token: str) -> None:
    """Authorization may be raw JWT without Bearer prefix."""
    client.set_token(token, bearer=False)
    resp = client.get("/events/registrations", expected_status=200)
    as_list(client.json(resp), context="GET /events/registrations (raw JWT)")


def test_update_event_as_owner(client: APIClient, token: str, event_id: int) -> dict[str, Any]:
    client.set_token(token)
    payload = sample_event_payload(
        name="E2E Conference Updated",
        description="Updated description",
        location="New York",
    )
    resp = client.put(f"/events/{event_id}", json=payload, expected_status=200)
    event = client.json(resp)
    assert_event_shape(event)
    assert event["id"] == event_id
    assert event["name"] == payload["name"]
    assert event["location"] == payload["location"]
    return event


def test_update_event_not_owner(
    client: APIClient, other_token: str, event_id: int
) -> None:
    client.set_token(other_token)
    resp = client.put(
        f"/events/{event_id}",
        json=sample_event_payload(name="Hijack attempt"),
        expected_status=403,
    )
    body = client.json(resp)
    assert_error_shape(body)
    assert "not authorized" in body["error"].lower(), body


def test_update_event_not_found(client: APIClient, token: str) -> None:
    client.set_token(token)
    resp = client.put(
        "/events/999999999",
        json=sample_event_payload(name="Ghost"),
        expected_status=404,
    )
    assert_error_shape(client.json(resp))


def test_update_event_invalid_id(client: APIClient, token: str) -> None:
    client.set_token(token)
    resp = client.put(
        "/events/abc",
        json=sample_event_payload(),
        expected_status=400,
    )
    assert_error_shape(client.json(resp))


def test_register_for_event(client: APIClient, token: str, event_id: int) -> None:
    client.set_token(token)
    resp = client.post(f"/events/{event_id}/register", expected_status=200)
    body = client.json(resp)
    assert body.get("message") == "Registered for event successfully!", body


def test_register_duplicate(client: APIClient, token: str, event_id: int) -> None:
    client.set_token(token)
    resp = client.post(f"/events/{event_id}/register", expected_status=409)
    body = client.json(resp)
    assert_error_shape(body)
    assert "already registered" in body["error"].lower(), body


def test_register_event_not_found(client: APIClient, token: str) -> None:
    client.set_token(token)
    resp = client.post("/events/999999999/register", expected_status=404)
    assert_error_shape(client.json(resp))


def test_register_invalid_id(client: APIClient, token: str) -> None:
    client.set_token(token)
    resp = client.post("/events/xyz/register", expected_status=400)
    assert_error_shape(client.json(resp))


def test_list_my_registrations(
    client: APIClient, token: str, expected_event_id: int
) -> None:
    client.set_token(token)
    resp = client.get("/events/registrations", expected_status=200)
    events = as_list(client.json(resp), context="GET /events/registrations")
    ids = {e["id"] for e in events}
    assert expected_event_id in ids, f"Expected event {expected_event_id} in {ids}"
    for e in events:
        assert_event_shape(e)


def test_list_registrations_requires_auth(client: APIClient) -> None:
    client.clear_auth()
    resp = client.get("/events/registrations")
    assert resp.status_code in (401, 403), f"Expected unauthorized, got {resp.status_code}"


def test_cancel_registration(client: APIClient, token: str, event_id: int) -> None:
    client.set_token(token)
    resp = client.delete(f"/events/{event_id}/register", expected_status=200)
    body = client.json(resp)
    assert body.get("message") == "Cancelled event registration successfully!", body


def test_cancel_registration_not_found(client: APIClient, token: str, event_id: int) -> None:
    """Cancel when not registered should 404."""
    client.set_token(token)
    resp = client.delete(f"/events/{event_id}/register", expected_status=404)
    body = client.json(resp)
    assert_error_shape(body)


def test_cancel_registration_invalid_id(client: APIClient, token: str) -> None:
    client.set_token(token)
    resp = client.delete("/events/bad-id/register", expected_status=400)
    assert_error_shape(client.json(resp))


def test_delete_event_not_owner(
    client: APIClient, other_token: str, event_id: int
) -> None:
    client.set_token(other_token)
    resp = client.delete(f"/events/{event_id}", expected_status=403)
    body = client.json(resp)
    assert_error_shape(body)
    assert "not authorized" in body["error"].lower(), body


def test_delete_event_as_owner(client: APIClient, token: str, event_id: int) -> None:
    client.set_token(token)
    resp = client.delete(f"/events/{event_id}", expected_status=200)
    body = client.json(resp)
    assert body.get("message") == "Event deleted successfully!", body


def test_delete_event_not_found(client: APIClient, token: str) -> None:
    client.set_token(token)
    resp = client.delete("/events/999999999", expected_status=404)
    assert_error_shape(client.json(resp))


def test_delete_event_invalid_id(client: APIClient, token: str) -> None:
    client.set_token(token)
    resp = client.delete("/events/!!", expected_status=400)
    assert_error_shape(client.json(resp))


def test_deleted_event_not_fetchable(client: APIClient, event_id: int) -> None:
    client.clear_auth()
    resp = client.get(f"/events/{event_id}")
    assert resp.status_code in (404, 500), f"Deleted event still reachable: {resp.status_code}"
    assert_error_shape(client.json(resp))


def test_registration_cascade_on_delete(
    client: APIClient, owner_token: str, attendee_token: str
) -> None:
    """Deleting an event should remove registrations (FK cascade)."""
    client.set_token(owner_token)
    create = client.post(
        "/events",
        json=sample_event_payload(name="Cascade Event"),
        expected_status=201,
    )
    event = client.json(create)
    event_id = event["id"]

    client.set_token(attendee_token)
    client.post(f"/events/{event_id}/register", expected_status=200)
    regs = as_list(
        client.json(client.get("/events/registrations", expected_status=200)),
        context="registrations before cascade",
    )
    assert event_id in {e["id"] for e in regs}

    client.set_token(owner_token)
    client.delete(f"/events/{event_id}", expected_status=200)

    client.set_token(attendee_token)
    regs_after = as_list(
        client.json(client.get("/events/registrations", expected_status=200)),
        context="registrations after cascade",
    )
    assert event_id not in {e["id"] for e in regs_after}, "Registration should cascade-delete"


# ---------------------------------------------------------------------------
# Runner
# ---------------------------------------------------------------------------

def run_test(report: SuiteReport, name: str, fn: Callable[[], None]) -> None:
    start = time.perf_counter()
    try:
        fn()
        report.add(
            TestResult(
                name=name,
                passed=True,
                duration_ms=(time.perf_counter() - start) * 1000,
            )
        )
    except Exception as exc:
        detail = f"{type(exc).__name__}: {exc}"
        # Keep traceback only for unexpected errors (not AssertionError with clear message)
        if not isinstance(exc, AssertionError):
            detail += "\n         " + traceback.format_exc().replace("\n", "\n         ")
        report.add(
            TestResult(
                name=name,
                passed=False,
                detail=detail,
                duration_ms=(time.perf_counter() - start) * 1000,
            )
        )


def check_server_reachable(client: APIClient) -> bool:
    try:
        client.get("/health", expected_status=200)
        return True
    except requests.exceptions.ConnectionError:
        print(f"ERROR: Cannot connect to API at {BASE_URL}")
        print("Start the server first, e.g.: go run main.go   (or make run)")
        return False
    except Exception as exc:
        print(f"ERROR: Health check failed against {BASE_URL}: {exc}")
        return False


def main() -> int:
    print("=" * 70)
    print("Events REST API — automated E2E suite")
    print(f"Base URL: {BASE_URL}")
    print("=" * 70)

    client = APIClient()
    report = SuiteReport()

    if not check_server_reachable(client):
        return 2

    # Unique users for this run so tests are re-runnable against a persistent DB
    owner_email = unique_email("owner")
    owner_password = "OwnerSecret123!"
    other_email = unique_email("other")
    other_password = "OtherSecret123!"
    attendee_email = unique_email("attendee")
    attendee_password = "AttendeeSecret123!"

    owner_token: str = ""
    other_token: str = ""
    attendee_token: str = ""
    event_id: int = 0
    secondary_event_id: int = 0

    # --- Health ---
    run_test(report, "GET /health returns ok", lambda: test_health_ok(client))

    # --- Signup ---
    run_test(
        report,
        "POST /signup creates owner user",
        lambda: test_signup_success(client, owner_email, owner_password),
    )
    run_test(
        report,
        "POST /signup creates other user",
        lambda: test_signup_success(client, other_email, other_password),
    )
    run_test(
        report,
        "POST /signup creates attendee user",
        lambda: test_signup_success(client, attendee_email, attendee_password),
    )
    run_test(
        report,
        "POST /signup rejects duplicate email (409)",
        lambda: test_signup_duplicate_email(client, owner_email, owner_password),
    )
    run_test(
        report,
        "POST /signup rejects invalid body (400)",
        lambda: test_signup_invalid_body(client),
    )

    # --- Login ---
    def do_login_owner() -> None:
        nonlocal owner_token
        owner_token = test_login_success(client, owner_email, owner_password)

    def do_login_other() -> None:
        nonlocal other_token
        other_token = test_login_success(client, other_email, other_password)

    def do_login_attendee() -> None:
        nonlocal attendee_token
        attendee_token = test_login_success(client, attendee_email, attendee_password)

    run_test(report, "POST /login returns JWT for owner", do_login_owner)
    run_test(report, "POST /login returns JWT for other user", do_login_other)
    run_test(report, "POST /login returns JWT for attendee", do_login_attendee)
    run_test(
        report,
        "POST /login rejects invalid credentials (401)",
        lambda: test_login_invalid_credentials(client, owner_email),
    )
    run_test(
        report,
        "POST /login rejects invalid body (400)",
        lambda: test_login_invalid_body(client),
    )

    # --- Public event reads ---
    run_test(report, "GET /events lists events (public)", lambda: test_list_events_public(client))
    run_test(
        report,
        "GET /events/:id invalid id (400)",
        lambda: test_get_event_invalid_id(client),
    )
    run_test(
        report,
        "GET /events/:id not found",
        lambda: test_get_event_not_found(client),
    )

    # --- Auth guards ---
    run_test(
        report,
        "POST /events requires auth",
        lambda: test_create_event_requires_auth(client),
    )
    run_test(
        report,
        "GET /events/registrations requires auth",
        lambda: test_list_registrations_requires_auth(client),
    )

    if not owner_token:
        print("\nSkipping authenticated flows — owner login failed.")
        return _finish(report)

    # --- Create / read / update events ---
    def do_create() -> None:
        nonlocal event_id
        event = test_create_event_success(client, owner_token)
        event_id = int(event["id"])

    run_test(report, "POST /events creates event (201)", do_create)
    run_test(
        report,
        "POST /events validation error (400)",
        lambda: test_create_event_validation(client, owner_token),
    )

    if event_id:
        run_test(
            report,
            "GET /events/:id returns created event",
            lambda: test_get_event_by_id(client, event_id),
        )
        run_test(
            report,
            "PUT /events/:id updates as owner (200)",
            lambda: test_update_event_as_owner(client, owner_token, event_id),
        )

    if other_token and event_id:
        run_test(
            report,
            "PUT /events/:id rejects non-owner (403)",
            lambda: test_update_event_not_owner(client, other_token, event_id),
        )

    run_test(
        report,
        "PUT /events/:id not found (404)",
        lambda: test_update_event_not_found(client, owner_token),
    )
    run_test(
        report,
        "PUT /events/:id invalid id (400)",
        lambda: test_update_event_invalid_id(client, owner_token),
    )
    run_test(
        report,
        "Authorization raw JWT (no Bearer) accepted",
        lambda: test_auth_raw_token_header(client, owner_token),
    )

    # --- Registrations ---
    if event_id:
        run_test(
            report,
            "POST /events/:id/register (200)",
            lambda: test_register_for_event(client, owner_token, event_id),
        )
        run_test(
            report,
            "POST /events/:id/register duplicate (409)",
            lambda: test_register_duplicate(client, owner_token, event_id),
        )
        run_test(
            report,
            "GET /events/registrations includes registered event",
            lambda: test_list_my_registrations(client, owner_token, event_id),
        )
        run_test(
            report,
            "DELETE /events/:id/register cancels (200)",
            lambda: test_cancel_registration(client, owner_token, event_id),
        )
        run_test(
            report,
            "DELETE /events/:id/register when not registered (404)",
            lambda: test_cancel_registration_not_found(client, owner_token, event_id),
        )

    run_test(
        report,
        "POST /events/:id/register not found (404)",
        lambda: test_register_event_not_found(client, owner_token),
    )
    run_test(
        report,
        "POST /events/:id/register invalid id (400)",
        lambda: test_register_invalid_id(client, owner_token),
    )
    run_test(
        report,
        "DELETE /events/:id/register invalid id (400)",
        lambda: test_cancel_registration_invalid_id(client, owner_token),
    )

    # --- Delete + ownership ---
    if other_token and event_id:
        run_test(
            report,
            "DELETE /events/:id rejects non-owner (403)",
            lambda: test_delete_event_not_owner(client, other_token, event_id),
        )

    if event_id:
        run_test(
            report,
            "DELETE /events/:id as owner (200)",
            lambda: test_delete_event_as_owner(client, owner_token, event_id),
        )
        run_test(
            report,
            "GET /events/:id after delete fails",
            lambda: test_deleted_event_not_fetchable(client, event_id),
        )

    run_test(
        report,
        "DELETE /events/:id not found (404)",
        lambda: test_delete_event_not_found(client, owner_token),
    )
    run_test(
        report,
        "DELETE /events/:id invalid id (400)",
        lambda: test_delete_event_invalid_id(client, owner_token),
    )

    # --- Cascade ---
    if owner_token and attendee_token:
        run_test(
            report,
            "DELETE event cascades registrations",
            lambda: test_registration_cascade_on_delete(
                client, owner_token, attendee_token
            ),
        )

    # Secondary create so list endpoint still has data to exercise after deletes
    def do_secondary_create() -> None:
        nonlocal secondary_event_id
        event = test_create_event_success(client, owner_token)
        secondary_event_id = int(event["id"])
        # Clean up leftover test event so re-runs do not accumulate forever
        client.set_token(owner_token)
        client.delete(f"/events/{secondary_event_id}", expected_status=200)

    run_test(
        report,
        "POST /events + cleanup secondary event",
        do_secondary_create,
    )

    return _finish(report)


def _finish(report: SuiteReport) -> int:
    print("-" * 70)
    total = len(report.results)
    print(
        f"Results: {report.passed_count} passed, {report.failed_count} failed, "
        f"{total} total"
    )
    print("=" * 70)
    if report.failed_count:
        print("Failed tests:")
        for r in report.results:
            if not r.passed:
                print(f"  - {r.name}: {r.detail}")
        return 1
    print("All tests passed.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
