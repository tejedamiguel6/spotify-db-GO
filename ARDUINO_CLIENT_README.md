# Building an Arduino/ESP32 Client for Miguel's Spotify API

**Audience:** the agent building the Arduino/embedded device. This document is
self-contained — you do not need access to the API's source code. Everything
below has been tested against the running server.

## What this API is

A Go service that continuously tracks Miguel's Spotify listening history. It
exposes **embedded-friendly, read-only endpoints** specifically designed for
microcontrollers: flat JSON, a plain-text format, ETag-based change detection,
and a Server-Sent Events stream. A background poller on the server keeps a
now-playing cache fresh (polls Spotify every 20s from 6 AM–11 PM, every 90s
overnight), so **your device only ever hits the cache** — poll as often as
every 2–5 seconds without worrying about Spotify rate limits.

## Connection details

| | |
|---|---|
| Production base URL | `https://api-spotify-tracks.mtejeda.co` |
| Local dev base URL | `http://<dev-machine-ip>:8080` (port configurable via `PORT` env) |
| Auth | **None** — all endpoints below are public GETs |
| Rate limit | 100 requests/min per IP — stay at ≥2s between polls |
| TLS | Production is HTTPS. ESP32: works out of the box (use `WiFiClientSecure` + `setInsecure()` for simplicity, or pin the cert). ESP8266: must call `client.setInsecure()` or set a fingerprint. |

> **Deploy status:** the embedded endpoints exist in the codebase but may
> not be live on production yet (they ship with the next ECS deploy). Verify
> with `GET /health` first; if it 404s on production, develop against a
> locally running server and the same code will work in production later.

> **Use GET, not HEAD.** The server does not route HEAD requests — a HEAD
> to any endpoint returns 404.

## Endpoints

### `GET /health`

Liveness check. Use it on boot to confirm connectivity.

```json
{"status":"ok","time":1783008115917}
```

### `GET /embedded/now` — the main endpoint

Current playback state, flat JSON, ~400 bytes:

```json
{
  "playing": true,
  "track_id": "0HB0UXiIT3HUjDid2OGXH5",
  "track": "Jeans",
  "artist": "Boy Harsher",
  "album": "Jeans",
  "progress_ms": 12344,
  "duration_ms": 207466,
  "art": "https://i.scdn.co/image/ab67616d00004851cb05cc5559a4fa3f6499b26c",
  "art_large": "https://i.scdn.co/image/ab67616d0000b273cb05cc5559a4fa3f6499b26c",
  "fetched_at": 1783008113349
}
```

Field semantics:

- `playing` — true only while actively playing; false when paused **or** when
  nothing is playing at all.
- `progress_ms` — already extrapolated to the moment of your request; you can
  drive a progress bar directly from `progress_ms / duration_ms`. Between your
  own polls, advance it locally with `millis()`.
- `artist` — multiple artists come comma-joined in one string (`"A, B"`).
- `art` — smallest album cover (usually 64×64 JPEG); `art_large` — largest
  (usually 640×640). Both are plain HTTPS image URLs on `i.scdn.co`.
- `fetched_at` — unix ms of the server's last Spotify poll. If it's more than
  ~3 minutes old, treat the data as stale (server-side issue).
- **Nothing playing:** string fields are omitted entirely —
  `{"playing":false,"progress_ms":0,"duration_ms":0,"fetched_at":...}`.
  Handle missing keys.
- Track/artist names are UTF-8 and can contain any characters.

#### Change detection with ETag (recommended)

The `ETag` response header changes **only when the track or play/pause state
changes** (format: `"<track_id>-<true|false>"`). Send it back on the next poll:

- `304 Not Modified` + empty body same song, same state. Do nothing.
- `200` + body something changed. Re-render, store the new `ETag`.

This makes each "no change" poll cost ~100 bytes. Note the ETag value includes
its surrounding double quotes — echo it back exactly as received.

#### `GET /embedded/now?format=text` — no JSON parser needed

For very constrained boards. Returns `key=value` lines, `\n`-terminated,
fixed order, always all 7 lines:

```
playing=1
track=Jeans
artist=Boy Harsher
album=Jeans
progress_ms=102434
duration_ms=207466
fetched_at=1783008194208
```

Parse with `readStringUntil('\n')` and split on the **first** `=`. Values may
be empty when nothing is playing. ETag works identically on this format.

### `GET /events` — Server-Sent Events (push)

Long-lived HTTP stream (`Content-Type: text/event-stream`). On connect you
immediately receive the current state, then one event per track/play-state
change. A `: keepalive` comment line arrives every 25s — use its absence
(say, 60s of silence) to detect a dead connection and reconnect.

```
event: now-playing
data: {"playing":true,"track":"Jeans","artist":"Boy Harsher", ...same JSON as /embedded/now...}
```

**Recommendation:** on ESP32/Arduino, prefer ETag polling — it's simpler and
survives flaky Wi-Fi better. SSE shines on a Raspberry Pi (Python
`sseclient-py`). If you do SSE on ESP32, keep the socket read loop non-blocking
and reconnect with backoff.

## Recommended device behavior

1. **Boot:** join Wi-Fi `GET /health` `GET /embedded/now` (no ETag) render.
2. **Loop every 2–5s:** `GET /embedded/now` with `If-None-Match`.
   - 304 advance the progress bar locally, nothing else.
   - 200 parse, re-render, save new ETag.
   - Network error / non-200/304 keep last state on screen, retry with
     exponential backoff (5s 10s 20s, cap 60s). Don't blank the display on
     a single failed poll.
3. **`playing=false`:** show a paused/idle state; keep polling (the ETag will
   change the instant playback resumes).

## Working ESP32 sketch (ETag polling)

```cpp
#include <WiFi.h>
#include <HTTPClient.h>
#include <ArduinoJson.h>   // v7

const char* BASE = "https://api-spotify-tracks.mtejeda.co";
String etag = "";

void checkNowPlaying() {
  HTTPClient http;
  http.begin(String(BASE) + "/embedded/now");
  if (etag.length()) http.addHeader("If-None-Match", etag);
  const char* keep[] = {"ETag"};
  http.collectHeaders(keep, 1);

  int code = http.GET();
  if (code == 200) {
    etag = http.header("ETag");
    JsonDocument doc;
    if (deserializeJson(doc, http.getString()) == DeserializationError::Ok) {
      bool playing = doc["playing"] | false;
      const char* track  = doc["track"]  | "";
      const char* artist = doc["artist"] | "";
      long progress = doc["progress_ms"] | 0L;
      long duration = doc["duration_ms"] | 0L;
      // TODO: render to display; doc["art"] has a 64px album cover URL
      Serial.printf("%s %s - %s (%ld/%ld)\n",
                    playing ? ">" : "||", track, artist, progress, duration);
    }
  }
  // 304 no change; other codes leave last state, let loop() back off
  http.end();
}

void loop() {
  checkNowPlaying();
  delay(4000);
}
```

For album art on a TFT/LED matrix: fetch the `art` URL (64×64 JPEG) and decode
with `TJpg_Decoder`, or ask Miguel — a server-side raw-RGB565 endpoint is a
planned addition if JPEG decoding is too heavy for your board.

## Other read-only endpoints (bigger payloads — Pi-class devices only)

These return large nested JSON; avoid them on microcontrollers:

- `GET /top-tracks` — most-played tracks
- `GET /listening-stats` — listening time by period + daily breakdown
- `GET /collection-stats` — collection totals and 30-day daily counts
- `GET /tracks/:id/streak`, `/tracks/:id/stats`, `/tracks/:id/daily` — per-track detail

Good raw material for an e-ink stats dashboard on a Raspberry Pi.
