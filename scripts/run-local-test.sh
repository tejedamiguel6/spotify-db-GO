#!/usr/bin/env bash
# Run the API locally against the Neon "local-test" branch with the cron off.
# It can never touch the live database: it refuses to start if the
# connection string points at the live (development) endpoint.
set -euo pipefail

PROJECT_ID="damp-salad-66273459"
TEST_BRANCH="local-test"
LIVE_ENDPOINT="ep-withered-boat"   # Neon "development" branch = live data

cd "$(dirname "$0")/.."

DATABASE_URL="$(neonctl connection-string "$TEST_BRANCH" --project-id "$PROJECT_ID" --pooled)"

if [[ -z "$DATABASE_URL" || "$DATABASE_URL" == *"$LIVE_ENDPOINT"* ]]; then
  echo "Refusing to start: connection string is empty or points at the live database." >&2
  exit 1
fi

# Exported vars win over .env (godotenv.Load never overrides existing ones).
export DATABASE_URL
export DISABLE_CRON=true
export PORT="${PORT:-8080}"

echo "Starting API on :$PORT against Neon branch '$TEST_BRANCH' (cron off)"
exec go run ./cmd/server
