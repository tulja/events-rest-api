# Local HashiCorp Vault Integration

This package provides a simple wrapper to read secrets from a local HashiCorp Vault (KV v2).

## Prerequisites

1. Install Vault locally:
   ```bash
   brew install vault          # macOS
   # or
   # https://developer.hashicorp.com/vault/downloads
   ```

2. Start a local development Vault server (recommended: use fixed root token for convenience):
   ```bash
   # Fixed token "root" (easiest for local dev)
   vault server -dev -dev-root-token-id="root"

   # Or normal (generates a new random token every restart)
   # vault server -dev
   ```

   Copy the `Root Token` shown in the output.

3. Set environment variables:
   ```bash
   export VAULT_ADDR="http://127.0.0.1:8200"
   export VAULT_TOKEN="root"
   ```

## Quick Setup (in Vault)

```bash
# Enable KV v2 (usually enabled by default at "secret" in dev mode)
vault secrets enable -path=secret kv-v2

# Write the JWT secret your app needs
vault kv put secret/events-api/jwt signing-key="AbcXyz123"

# Verify
vault kv get secret/events-api/jwt
```

## Usage in Code

```go
import (
	"context"
	"log"

	"events-rest-api/secrets"
)

func initConfig() {
	client, err := secrets.NewClient(nil) // uses VAULT_ADDR + VAULT_TOKEN
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Read a single value
	signingKey, err := client.GetSecretValue(ctx, "events-api/jwt", "signing-key")
	if err != nil {
		log.Fatal(err)
	}

	// Or the whole secret
	secret, _ := client.GetSecret(ctx, "myapp/db")
	fmt.Println(secret["username"])
}
```

## Loading at Startup (Recommended Pattern)

```go
var (
	DBPassword string
	APIKey     string
)

func LoadSecrets() {
	client, err := secrets.NewClient(nil)
	if err != nil {
		log.Fatal("vault client:", err)
	}

	ctx := context.Background()
	DBPassword = client.MustGetSecretValue(ctx, "myapp/db", "password")
	APIKey     = client.MustGetSecretValue(ctx, "myapp/external", "api_key")
}
```

## Notes

- This example uses **token authentication** (ideal for local/dev).
- For production, prefer **AppRole**, **Kubernetes**, or **JWT/OIDC** auth methods.
- Always use `KV v2` (the path is `secret/data/...` under the hood, handled by the `KVv2()` helper).
- Do **not** commit tokens or secrets to source control.
