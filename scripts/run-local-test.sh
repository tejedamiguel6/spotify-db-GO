#!/usr/bin/env bash
# Run the API locally against the Neon "local-test" branch with the cron off.
# It can never touch the live database: it refuses to start if the
# connection string points at the live endpoint.
#
# Needs two values, from the environment or from scripts/.local-test.env
# (gitignored, never committed):
#   NEON_PROJECT_ID     the Neon project ID
#   NEON_LIVE_ENDPOINT  the endpoint ID of the live branch (e.g. "ep-xxxx-xxxx")
set -euo pipefail

cd "$(dirname "$0")/.."

if [[ -f scripts/.local-test.env ]]; then
  # shellcheck disable=SC1091
  source scripts/.local-test.env
fi

: "${NEON_PROJECT_ID:?Set NEON_PROJECT_ID (or add it to scripts/.local-test.env)}"
: "${NEON_LIVE_ENDPOINT:?Set NEON_LIVE_ENDPOINT (or add it to scripts/.local-test.env)}"

TEST_BRANCH="local-test"

DATABASE_URL="$(neonctl connection-string "$TEST_BRANCH" --project-id "$NEON_PROJECT_ID" --pooled)"

if [[ -z "$DATABASE_URL" || "$DATABASE_URL" == *"$NEON_LIVE_ENDPOINT"* ]]; then
  echo "Refusing to start: connection string is empty or points at the live database." >&2
  exit 1
fi

# Exported vars win over .env (godotenv.Load never overrides existing ones).
export DATABASE_URL
export DISABLE_CRON=true
export PORT="${PORT:-8080}"

echo "Starting API on :$PORT against Neon branch '$TEST_BRANCH' (cron off)"
exec go run ./cmd/server
