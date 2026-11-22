# CLAUDE.md

Guidance for Claude Code (and other agents) working in this repository.

## What this is

A Go (Gin + pgx) API that records Miguel's Spotify listening history into Neon PostgreSQL and serves it to his music-journal website.

- Live API: `https://api-spotify-tracks.mtejeda.co`, on AWS ECS Fargate (`us-east-2`)
- Website that consumes it: separate repo `tejedamiguel6/music-journal-spotify-api`, checked out locally at `/Users/migueltejeda/Documents/Developer/website`

## Commands

- `go build ./...`: build everything (CI runs this plus `go test ./...` on PRs to `main`)
- `go run ./cmd/server`: run the API on `:8080` (`PORT` overrides). **Don't run this against the live database; see below.**
- `scripts/run-local-test.sh`: the safe way to run locally (see "Local testing")
- `git config core.hooksPath .githooks`: enables the pre-commit hook, which blocks secrets, the `server` binary, `DEBUG_*.json` and files over 1 MB

## Layout

- `cmd/server/main.go`: routes, API-key middleware, CORS, and startup of the background jobs
- `internal/handlers/`: HTTP handlers (`tracks.go` also holds the Spotify cron, `nowplaying.go` the now-playing poller and SSE, `calendar.go` the calendar)
- `internal/repository/`: DB pool, schema setup (`ensureTablesExist` runs on every startup), queries
- `internal/models/`: insert and select helpers for `recently_played` and `recently_liked`
- `internal/services/client.go`: Spotify Web API client (token refresh, recently played, saved tracks)
- `cmd/init-db`, `cmd/recovery*`: one-off maintenance tools

## Background jobs: they write to the database

On startup the server launches `StartSpotifyCron` (polls Spotify every 5 to 15 min and inserts into `recently_played` and `recently_liked`, refreshing and storing the Spotify token) and `StartNowPlayingPoller`. **Set `DISABLE_CRON=true` to start the API without them.**

## Database: read this before touching it

Neon project `damp-salad-66273459`:
- **Branch `development` holds the LIVE data.** The ECS server writes there. Despite the name, treat it as production.
- Branch `production` is empty and unused (a known naming mix-up).
- Branch `local-test` is a disposable copy for local runs.
- Backup branches are named `backup-*`.

Rules:
- Never point a local server with the cron on at the live branch. A second writer races the live cron, and `recently_liked` has **no unique constraint** (it's deduped only by app logic), so duplicates are possible. Both servers would also rotate the shared Spotify token.
- `recently_played` is protected by `UNIQUE (spotify_song_id, played_at)` and `ON CONFLICT DO NOTHING`.
- Query the live branch read-only: `PGOPTIONS='-c default_transaction_read_only=on' psql "$(neonctl connection-string development --project-id damp-salad-66273459)"`.
- Take a backup before any schema change on live (`neonctl branches create --parent development --no-compute`, plus a `pg_dump`; use Homebrew `postgresql@17`, because the default v14 client is too old for the server), and ask Miguel first.
- `played_at` is stored as UTC wall-clock time (`timestamp without time zone`).

## Local testing

`scripts/run-local-test.sh` gets the `local-test` connection string from `neonctl`, sets `DISABLE_CRON=true`, and refuses to start if the URL points at the live branch. Variables you export take priority over `.env`, because `godotenv.Load` never overwrites existing variables. To point the website at it, add `NEXT_PUBLIC_BACKEND_API_URL=http://localhost:8080` to the site's `.env.local`, and remove it afterwards.

## Auth

- GET requests are public, rate-limited to 100/min per IP.
- POST/PATCH/DELETE require `X-API-Key` (the `API_KEY` env var). Exception on `main` today: `/save-refresh` bypasses the key. A fix exists but must ship together with a website change that sends the key server-side.
- Never commit key values. A key leaked once (2026-07-02), so treat any hardcoded key as burned.

## Deploying (AWS ECS)

Cluster `spotify-cluster`, service `spotify-track-db`, task definition `spotify-track-db:13` (image `…/spotify-track-db:latest` in ECR; secrets come from Secrets Manager). AWS auth is `aws login`, done by Miguel in the browser.

Safe procedure (preferred over `deploy-to-ecs.sh`, which registers a new task definition from the repo's JSON file):
1. `git worktree add --detach /tmp/deploy origin/main`. Build from that clean copy, never from a working folder with uncommitted work.
2. `docker buildx build --platform linux/amd64 -t spotify-track-db:main-<sha> .`
3. Smoke-test that image against the `local-test` branch with `DISABLE_CRON=true`.
4. Tag and push `"${REPO}:main-<sha>"` and `"${REPO}:latest"`. Use the braces: in zsh, `$REPO:latest` triggers the `:l` lowercase modifier.
5. `aws ecs update-service --cluster spotify-cluster --service spotify-track-db --task-definition spotify-track-db:13 --force-new-deployment --region us-east-2`, then `aws ecs wait services-stable …`.
6. Verify `/health` and `/calendar`, and check the logs (`aws logs tail /ecs/spotify-track-db --region us-east-2`) for "Starting Spotify cron".

Rollback: re-tag an earlier image (e.g. ECR tag `rollback-2026-02-27`, or any `main-<sha>`) as `latest` and force a new deployment.

## Working with Miguel

He prefers short, plain-language explanations and step-by-step instructions over jargon. Confirm before deploying, before writing to the live database, and before any outward-facing action.
