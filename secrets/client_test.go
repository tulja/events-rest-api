package secrets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient_MissingToken(t *testing.T) {
	t.Setenv("VAULT_TOKEN", "")
	t.Setenv("VAULT_ADDR", "")

	_, err := NewClient(&Config{Address: "http://127.0.0.1:8200", Token: ""})
	if err == nil {
		t.Fatal("expected error when token missing")
	}
	if !strings.Contains(err.Error(), "token") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewClient_WithConfig(t *testing.T) {
	client, err := NewClient(&Config{
		Address: "http://127.0.0.1:8200",
		Token:   "root",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client == nil || client.client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_EnvToken(t *testing.T) {
	t.Setenv("VAULT_TOKEN", "env-token")
	t.Setenv("VAULT_ADDR", "")

	client, err := NewClient(&Config{Address: "http://example.local:8200"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.client.Token() != "env-token" {
		t.Fatalf("token: got %q", client.client.Token())
	}
}

func TestNewClient_DefaultAddress(t *testing.T) {
	t.Setenv("VAULT_ADDR", "")
	client, err := NewClient(&Config{Token: "root"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.client.Address() != "http://127.0.0.1:8200" {
		t.Fatalf("address: got %q", client.client.Address())
	}
}

func TestGetSecret_Success(t *testing.T) {
	server := newVaultKVServer(t, map[string]any{
		"signing-key": "test-signing-key",
	}, http.StatusOK)
	defer server.Close()

	client, err := NewClient(&Config{Address: server.URL, Token: "root"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	data, err := client.GetSecret(context.Background(), "events-api/jwt")
	if err != nil {
		t.Fatalf("GetSecret: %v", err)
	}
	if data["signing-key"] != "test-signing-key" {
		t.Fatalf("data: %+v", data)
	}
}

func TestGetSecret_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := NewClient(&Config{Address: server.URL, Token: "root"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := client.GetSecret(context.Background(), "missing/path"); err == nil {
		t.Fatal("expected error")
	}
}

func TestGetSecretValue_Success(t *testing.T) {
	server := newVaultKVServer(t, map[string]any{
		"signing-key": "abc",
	}, http.StatusOK)
	defer server.Close()

	client, err := NewClient(&Config{Address: server.URL, Token: "root"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	val, err := client.GetSecretValue(context.Background(), "events-api/jwt", "signing-key")
	if err != nil {
		t.Fatalf("GetSecretValue: %v", err)
	}
	if val != "abc" {
		t.Fatalf("got %q", val)
	}
}

func TestGetSecretValue_KeyMissing(t *testing.T) {
	server := newVaultKVServer(t, map[string]any{
		"other": "x",
	}, http.StatusOK)
	defer server.Close()

	client, err := NewClient(&Config{Address: server.URL, Token: "root"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.GetSecretValue(context.Background(), "events-api/jwt", "signing-key")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetSecretValue_NotString(t *testing.T) {
	server := newVaultKVServer(t, map[string]any{
		"signing-key": 123,
	}, http.StatusOK)
	defer server.Close()

	client, err := NewClient(&Config{Address: server.URL, Token: "root"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.GetSecretValue(context.Background(), "events-api/jwt", "signing-key")
	if err == nil || !strings.Contains(err.Error(), "not a string") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMustGetSecretValue_Success(t *testing.T) {
	server := newVaultKVServer(t, map[string]any{
		"signing-key": "must-value",
	}, http.StatusOK)
	defer server.Close()

	client, err := NewClient(&Config{Address: server.URL, Token: "root"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	val := client.MustGetSecretValue(context.Background(), "events-api/jwt", "signing-key")
	if val != "must-value" {
		t.Fatalf("got %q", val)
	}
}

func TestMustGetSecretValue_Panics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := NewClient(&Config{Address: server.URL, Token: "root"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = client.MustGetSecretValue(context.Background(), "events-api/jwt", "signing-key")
}

func newVaultKVServer(t *testing.T, data map[string]any, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Vault KV v2: GET /v1/secret/data/<path>
		if !strings.HasPrefix(r.URL.Path, "/v1/secret/data/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		resp := map[string]any{
			"data": map[string]any{
				"data":     data,
				"metadata": map[string]any{},
			},
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
}
