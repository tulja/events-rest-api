#!/usr/bin/env bash
#
# Starts a local Vault dev server with a fixed root token ("root").
# This makes local development much easier (no random token every time).

set -e

echo "==> Starting Vault dev server with fixed root token..."
echo "==> Token will be: root"
echo ""

# Kill any existing dev server
pkill -f 'vault server -dev' 2>/dev/null || true
sleep 1

# Start with fixed token
vault server -dev -dev-root-token-id="root" > /tmp/vault.log 2>&1 &

# Wait for it to be ready
sleep 3

export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=root

vault kv put secret/events-api/jwt signing-key=\"AbcXyz123\"

echo "==> Vault started!"
echo ""
echo "Export these for your app:"
echo "  export VAULT_ADDR=http://127.0.0.1:8200"
echo "  export VAULT_TOKEN=root"
echo ""
echo "Insert the JWT secret (run this once):"
echo "  vault kv put secret/events-api/jwt signing-key=\"AbcXyz123\""
echo ""
echo "Verify:"
echo "  vault kv get secret/events-api/jwt"
echo ""
echo "Then start your Go app:"
echo "  go run ."
echo ""
echo "Vault logs are at: /tmp/vault.log"
echo "To stop: pkill -f 'vault server -dev'"