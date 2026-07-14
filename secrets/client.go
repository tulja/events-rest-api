package secrets

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	vaultapi "github.com/hashicorp/vault/api"
)

// Client wraps the HashiCorp Vault API client.
type Client struct {
	client *vaultapi.Client
}

// Config holds connection settings for Vault.
type Config struct {
	Address string // e.g. "http://127.0.0.1:8200"
	Token   string // Vault token (for dev/local)
}

// NewClient creates a Vault client configured for local development.
// It reads VAULT_ADDR and VAULT_TOKEN from environment if not provided.
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = &Config{}
	}

	if cfg.Address == "" {
		cfg.Address = os.Getenv("VAULT_ADDR")
	}
	if cfg.Address == "" {
		cfg.Address = "http://127.0.0.1:8200"
	}

	if cfg.Token == "" {
		cfg.Token = os.Getenv("VAULT_TOKEN")
	}

	config := vaultapi.DefaultConfig()
	config.Address = cfg.Address

	client, err := vaultapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	if cfg.Token != "" {
		client.SetToken(cfg.Token)
	} else {
		return nil, fmt.Errorf("no Vault token provided (set VAULT_TOKEN env var or pass Config.Token)")
	}

	slog.Info("vault client created", "address", cfg.Address)
	return &Client{client: client}, nil
}

// GetSecret reads a secret from the KV v2 secrets engine.
// path is the path under the mount, e.g. "myapp/db" for secret/data/myapp/db
func (c *Client) GetSecret(ctx context.Context, path string) (map[string]interface{}, error) {
	secret, err := c.client.KVv2("secret").Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret at %s: %w", path, err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("secret not found at %s", path)
	}
	return secret.Data, nil
}

// GetSecretValue is a convenience helper to fetch a single value from a KV v2 secret.
func (c *Client) GetSecretValue(ctx context.Context, path, key string) (string, error) {
	data, err := c.GetSecret(ctx, path)
	if err != nil {
		return "", err
	}

	val, ok := data[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret %s", key, path)
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("key %q in secret %s is not a string", key, path)
	}

	return str, nil
}

// MustGetSecretValue is like GetSecretValue but panics on error. Useful for startup.
func (c *Client) MustGetSecretValue(ctx context.Context, path, key string) string {
	val, err := c.GetSecretValue(ctx, path, key)
	if err != nil {
		slog.Error("failed to load required secret", "path", path, "key", key, "err", err)
		panic(fmt.Sprintf("failed to load required secret %s/%s: %v", path, key, err))
	}
	return val
}
