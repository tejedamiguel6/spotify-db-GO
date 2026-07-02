# Embedded Integration (Raspberry Pi / Arduino / ESP32)

Endpoints designed for microcontrollers and single-board computers. All are
**GET, no API key required**, and served from an in-memory cache — a background
poller talks to Spotify (every 20s during 6 AM–11 PM, every 90s overnight), so
your devices can poll as often as every couple of seconds without touching
Spotify's rate limits. The server-wide limit of 100 requests/min per IP still
applies.

Base URL: `https://api-spotify-tracks.mtejeda.co` (or `http://localhost:8080` locally).

## GET /health

```json
{"status":"ok","time":1783008115917}
```

## GET /embedded/now

Flat now-playing state:

```json
{
  "playing": true,
  "track_id": "0HB0UXiIT3HUjDid2OGXH5",
  "track": "Jeans",
  "artist": "Boy Harsher",
  "album": "Jeans",
  "progress_ms": 12344,
  "duration_ms": 207466,
  "art": "https://i.scdn.co/image/...4851...",
  "art_large": "https://i.scdn.co/image/...b273...",
  "fetched_at": 1783008113349
}
```

- `progress_ms` is extrapolated to request time, so it's safe to drive a live progress bar.
- `art` is the smallest album image (~64px, good for LED matrices / thumbnails); `art_large` is ~640px.
- When nothing is playing: `{"playing":false,"progress_ms":0,"duration_ms":0,"fetched_at":...}`.

### Cheap change detection with ETag

The `ETag` only changes when the **track or play/pause state** changes. Send it
back and a `304 Not Modified` (zero-byte body) means nothing changed — ideal
for battery-powered boards:

```
GET /embedded/now
ETag: "0HB0UXiIT3HUjDid2OGXH5-true"

GET /embedded/now
If-None-Match: "0HB0UXiIT3HUjDid2OGXH5-true"
304
```

### `?format=text` — no JSON parser needed

For boards where ArduinoJson is too heavy (e.g. an Uno with an Ethernet shield),
`/embedded/now?format=text` returns key=value lines, parseable with
`readStringUntil('\n')`:

```
playing=1
track=Jeans
artist=Boy Harsher
album=Jeans
progress_ms=102434
duration_ms=207466
fetched_at=1783008194208
```

## GET /events — Server-Sent Events

Long-lived HTTP stream. On connect you immediately get the current state, then
an event each time the track or play state changes, with `: keepalive`
comments every 25s in between:

```
event: now-playing
data: {"playing":true,"track":"Jeans","artist":"Boy Harsher",...}
```

### Raspberry Pi (Python)

```python
# pip install sseclient-py requests
import json, requests, sseclient

resp = requests.get("https://api-spotify-tracks.mtejeda.co/events", stream=True)
for event in sseclient.SSEClient(resp).events():
    now = json.loads(event.data)
    if now["playing"]:
        print(f"▶ {now['track']} — {now['artist']}")
    else:
        print("⏸ paused")
```

### ESP32 (Arduino) — polling with ETag

```cpp
#include <WiFi.h>
#include <HTTPClient.h>
#include <ArduinoJson.h>

String etag = "";

void checkNowPlaying() {
  HTTPClient http;
  http.begin("https://api-spotify-tracks.mtejeda.co/embedded/now");
  if (etag.length()) http.addHeader("If-None-Match", etag);
  const char* keep[] = {"ETag"};
  http.collectHeaders(keep, 1);

  int code = http.GET();
  if (code == 304) { http.end(); return; }  // nothing changed
  if (code == 200) {
    etag = http.header("ETag");
    JsonDocument doc;
    deserializeJson(doc, http.getString());
    if (doc["playing"]) {
      Serial.printf("Now playing: %s - %s\n",
                    doc["track"].as<const char*>(),
                    doc["artist"].as<const char*>());
      // doc["progress_ms"] / doc["duration_ms"] progress bar
      // doc["art"] 64px album cover URL
    }
  }
  http.end();
}

void loop() {
  checkNowPlaying();
  delay(5000);
}
```

## Notes

- ESP8266 users: HTTPS needs `client.setInsecure()` or a certificate
  fingerprint; ESP32 handles TLS out of the box.
- The port is configurable with the `PORT` env var (defaults to 8080).
