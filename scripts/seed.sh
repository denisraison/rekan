#!/usr/bin/env bash
# Reset dev environment and seed with test data.
# Usage: make seed

set -euo pipefail

PB_URL="http://localhost:8090"
PB_DIR="api/pb_data"

ADMIN_EMAIL="${SEED_ADMIN_EMAIL:-admin@rekan.local}"
ADMIN_PASSWORD="${SEED_ADMIN_PASSWORD:-admin1234567}"
USER_EMAIL="${SEED_USER_EMAIL:-operador@rekan.local}"
USER_PASSWORD="${SEED_USER_PASSWORD:-senha1234567}"

command -v jq >/dev/null 2>&1 || { echo "jq is required"; exit 1; }
command -v curl >/dev/null 2>&1 || { echo "curl is required"; exit 1; }

echo "=== Resetting dev environment ==="

# Kill any process using port 8090
ss -lptn 'sport = :8090' | grep -oP '(?<=pid=)\d+' | sort -u | xargs kill -9 2>/dev/null || true
sleep 0.3

# Remove existing data
echo "Removing $PB_DIR..."
rm -rf "$PB_DIR"

# Bootstrap DB + superadmin (runs migrations, no server needed)
echo "Creating superadmin $ADMIN_EMAIL..."
(cd api && go run . superuser upsert "$ADMIN_EMAIL" "$ADMIN_PASSWORD") 2>&1 | tail -1

# Start PocketBase in background
echo "Starting PocketBase..."
(cd api && go run . serve --http=0.0.0.0:8090 >/tmp/pocketbase-seed.log 2>&1) &
trap "ss -lptn 'sport = :8090' | grep -oP '(?<=pid=)\d+' | sort -u | xargs kill -9 2>/dev/null || true" EXIT

# Wait for PocketBase
echo "Waiting for PocketBase on :8090..."
for _ in $(seq 1 60); do
  nc -z localhost 8090 2>/dev/null && break
  sleep 0.5
done
nc -z localhost 8090 2>/dev/null || { echo "PocketBase failed to start"; exit 1; }

# Authenticate as superadmin
TOKEN=$(curl -sf -X POST "$PB_URL/api/collections/_superusers/auth-with-password" \
  -H "Content-Type: application/json" \
  -d "{\"identity\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}" \
  | jq -r '.token')

auth=(-H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN")

# Create operator user
echo "Creating user $USER_EMAIL..."
curl -sf -X POST "$PB_URL/api/collections/users/records" \
  "${auth[@]}" \
  -d "{
    \"email\": \"$USER_EMAIL\",
    \"password\": \"$USER_PASSWORD\",
    \"passwordConfirm\": \"$USER_PASSWORD\",
    \"emailVisibility\": true
  }" | jq -r '"  created user: " + .email'

# Create sample business (fully onboarded, active)
echo "Creating sample business..."
curl -sf -X POST "$PB_URL/api/collections/businesses/records" \
  "${auth[@]}" \
  -d '{
    "name": "Confeitaria da Elenice",
    "type": "Confeitaria",
    "city": "São Paulo",
    "state": "SP",
    "phone": "5511988887777",
    "services": ["Bolos personalizados", "Docinhos", "Cupcakes", "Tortas"],
    "target_audience": "Famílias que buscam bolos artesanais para festas",
    "brand_vibe": "Acolhedor, artesanal, delicioso",
    "quirks": "Usa somente ingredientes naturais sem corantes artificiais",
    "client_name": "Elenice Silva",
    "client_email": "elenice@example.com",
    "invite_status": "active",
    "tier": "parceiro",
    "commitment": "mensal",
    "onboarding_step": 4
  }' | jq -r '"  created business: " + .name'

echo ""
echo "=== Seed complete ==="
echo "  Admin:    $ADMIN_EMAIL / $ADMIN_PASSWORD"
echo "  Operador: $USER_EMAIL / $USER_PASSWORD"
echo ""
echo "Run: make dev"
