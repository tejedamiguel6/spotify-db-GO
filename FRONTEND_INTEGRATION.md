# Frontend Integration Guide

## Your API Key
```
<your_api_key>
```

---

## Quick Setup

### Step 1: Add to Environment Variables

Create or update `.env.local` in your frontend project:

```bash
# .env.local
NEXT_PUBLIC_SPOTIFY_API_URL=https://api-spotify-tracks.mtejeda.co
NEXT_PUBLIC_SPOTIFY_API_KEY=<your_api_key>
```

### Step 2: Update Your API Calls

---

## Option 1: Using Fetch (Vanilla JavaScript)

### Before (without API key):
```javascript
fetch('https://api-spotify-tracks.mtejeda.co/now-listening-to')
  .then(res => res.json())
  .then(data => console.log(data));
```

### After (with API key):
```javascript
const API_KEY = process.env.NEXT_PUBLIC_SPOTIFY_API_KEY;

fetch('https://api-spotify-tracks.mtejeda.co/now-listening-to', {
  headers: {
    'X-API-Key': API_KEY
  }
})
  .then(res => res.json())
  .then(data => console.log(data));
```

---

## Option 2: Using Axios (Recommended)

### Install Axios (if not already):
```bash
npm install axios
```

### Create an API client:

Create `lib/api.js` or `utils/api.js`:

```javascript
// lib/api.js
import axios from 'axios';

const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_SPOTIFY_API_URL || 'https://api-spotify-tracks.mtejeda.co',
  headers: {
    'X-API-Key': process.env.NEXT_PUBLIC_SPOTIFY_API_KEY
  }
});

export default api;
```

### Use it in your components:

```javascript
// In your component
import api from '@/lib/api';

// Example: Get now listening
const getNowListening = async () => {
  try {
    const response = await api.get('/now-listening-to');
    console.log(response.data);
    return response.data;
  } catch (error) {
    console.error('Error fetching now listening:', error);
  }
};

// Example: Get collection stats
const getStats = async () => {
  const { data } = await api.get('/collection-stats');
  return data;
};
```

---

## Option 3: React Hook (Best for React/Next.js)

### Create a custom hook:

```javascript
// hooks/useSpotifyAPI.js
import { useState, useEffect } from 'react';

const API_URL = process.env.NEXT_PUBLIC_SPOTIFY_API_URL;
const API_KEY = process.env.NEXT_PUBLIC_SPOTIFY_API_KEY;

export const useNowListening = (intervalMs = 4000) => {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchNowListening = async () => {
      try {
        const response = await fetch(`${API_URL}/now-listening-to`, {
          headers: {
            'X-API-Key': API_KEY
          }
        });

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const result = await response.json();
        setData(result);
        setError(null);
      } catch (err) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    // Fetch immediately
    fetchNowListening();

    // Set up polling interval
    const interval = setInterval(fetchNowListening, intervalMs);

    // Cleanup on unmount
    return () => clearInterval(interval);
  }, [intervalMs]);

  return { data, loading, error };
};

// Usage in component:
// const { data, loading, error } = useNowListening(4000);
```

---

## Option 4: For Your Current 4-Second Polling

If you're currently polling every 4 seconds, update it like this:

### Before:
```javascript
setInterval(() => {
  fetch('https://api-spotify-tracks.mtejeda.co/now-listening-to')
    .then(res => res.json())
    .then(data => setNowPlaying(data));
}, 4000);
```

### After:
```javascript
const API_KEY = process.env.NEXT_PUBLIC_SPOTIFY_API_KEY;

setInterval(() => {
  fetch('https://api-spotify-tracks.mtejeda.co/now-listening-to', {
    headers: {
      'X-API-Key': API_KEY
    }
  })
    .then(res => res.json())
    .then(data => setNowPlaying(data))
    .catch(err => console.error('API Error:', err));
}, 4000);
```

---

## Bonus: Pause Polling When Tab Hidden

To save API calls (and your rate limit), pause polling when the tab isn't visible:

```javascript
import { useState, useEffect } from 'react';

export const useNowListening = () => {
  const [data, setData] = useState(null);
  const [isPolling, setIsPolling] = useState(true);

  useEffect(() => {
    // Pause polling when tab is hidden
    const handleVisibilityChange = () => {
      setIsPolling(!document.hidden);
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, []);

  useEffect(() => {
    if (!isPolling) return;

    const fetchData = async () => {
      const response = await fetch(`${API_URL}/now-listening-to`, {
        headers: { 'X-API-Key': API_KEY }
      });
      const result = await response.json();
      setData(result);
    };

    fetchData();
    const interval = setInterval(fetchData, 4000);

    return () => clearInterval(interval);
  }, [isPolling]);

  return data;
};
```

---

## All Available Endpoints

> All GETs are public (no API key required). Writes (POST / PATCH) require the `X-API-Key` header. Setting the header on GETs is harmless and recommended for consistency.

### At a glance

| Method | Path | Filters | Use case |
|---|---|---|---|
| GET | `/now-listening-to` | – | Live "currently playing" (4s polling) |
| GET | `/recently-played-tracks` | – | Full play history, newest first |
| GET | `/recently-liked` | – | All saved/liked tracks |
| GET | `/genre/:genre` | path param | Liked artists matching a genre |
| GET | `/top-tracks` | `from`, `to`, `limit` | **Most played in a date range** |
| GET | `/tracks/:id/stats` | `from`, `to` | Per-track totals (plays, time) |
| GET | `/tracks/:id/streak` | – | Per-track current + longest streak |
| GET | `/tracks/:id/daily` | `from`, `to` | Per-track day-by-day plays |
| GET | `/collection-stats` | – | Track counts by fixed periods |
| GET | `/listening-stats` | – | Listening time by fixed periods + last 30 days |
| PATCH | `/mostPlayedTracks/track/:spotify_song_id` | – | Update a track's play_count |
| POST | `/save-refresh` | – | Store/rotate Spotify refresh token (no auth) |
| POST | `/backfill-duration` | – | Admin: backfill `duration_ms` from Spotify |

---

### Filtering — the only knobs you have

Three endpoints support a date range: `/top-tracks`, `/tracks/:id/stats`, `/tracks/:id/daily`. The other "stats" endpoints (`/collection-stats`, `/listening-stats`) only expose **fixed periods** baked into the response (e.g. `last_24_hours`, `last_week`, `last_month`, `last_3_months`, `all_time`).

**Date format:** `YYYY-MM-DD` (no time-of-day, no timezone offset).

**Inclusive end:** `to` is **inclusive of the entire day**. So `from=2026-03-01&to=2026-03-31` returns all of March, including plays at `2026-03-31 23:59:59`.

**Optional bounds:** Either or both of `from` / `to` may be omitted. `from` alone = "since X, up to now"; `to` alone = "everything up to and including X"; neither = "all-time".

**Limit (`/top-tracks` only):** default `20`, hard-capped at `100`.

#### Filter recipes

```js
const base = process.env.NEXT_PUBLIC_SPOTIFY_API_URL;

// Top 20 tracks for March 2026
`${base}/top-tracks?from=2026-03-01&to=2026-03-31`

// Top 50 of all time
`${base}/top-tracks?limit=50`

// Top 10 of the last 7 days
const since = new Date(Date.now() - 7 * 86400_000).toISOString().slice(0, 10);
`${base}/top-tracks?from=${since}&limit=10`

// Plays of one specific song during March
`${base}/tracks/6rua8r6OWbwBisNEsXtyRW/daily?from=2026-03-01&to=2026-03-31`

// All-time stats for one specific song
`${base}/tracks/6rua8r6OWbwBisNEsXtyRW/stats`
```

---

### Endpoint reference

#### `GET /top-tracks` — most played in a date range

**Query params:** `from`, `to` (both `YYYY-MM-DD`, both optional), `limit` (default 20, max 100).

```js
const res = await fetch(`${base}/top-tracks?from=2026-03-01&to=2026-03-31&limit=20`);
const { count, from, to, tracks } = await res.json();
// tracks: [{ rank, song_id, track_name, artist_name, album_name, album_cover_url, play_count, total_ms, formatted }]
// formatted is "Xh Ym" (e.g. "1h 18m"). total_ms is sum of duration_ms across all plays.
```

#### `GET /tracks/:id/stats` — totals for one track

**Query params:** `from`, `to` (both `YYYY-MM-DD`, both optional). Without them: lifetime totals.

```js
const res = await fetch(`${base}/tracks/${songId}/stats`);
// { song_id, track_name, artist_name, play_count, total_ms, formatted, first_listen, last_listen }
```

#### `GET /tracks/:id/streak` — listening streaks for one track

No query params. Returns longest-ever streak plus current streak (or `null` if not currently listening on consecutive days).

```js
const res = await fetch(`${base}/tracks/${songId}/streak`);
// { song_id, track_name, artist_name,
//     current_streak: { days, start, end } | null,
//     longest_streak: { days, start, end } }
```

#### `GET /tracks/:id/daily` — day-by-day plays for one track

**Query params:** `from`, `to` (`YYYY-MM-DD`, optional). Returns one entry per day with at least one play.

```js
const res = await fetch(`${base}/tracks/${songId}/daily?from=2026-03-01&to=2026-03-31`);
// { song_id, track_name, artist_name,
//     days: [{ date, play_count, total_ms }] }
```

#### `GET /listening-stats` — duration totals + 30-day breakdown

No query params. Returns totals for fixed periods plus a daily breakdown for the last 30 days (only days with at least one play are included).

```js
const res = await fetch(`${base}/listening-stats`);
// { listening_time: { all_time, last_24_hours, last_week, last_month, last_3_months },
//     daily_breakdown_last_30_days: [{ date, total_ms, formatted, count }] }
// Each listening_time entry: { total_ms, formatted }
```

#### `GET /collection-stats` — track counts by period

No query params. Returns counts (not durations) per period plus a 6-month progress estimate.

```js
const res = await fetch(`${base}/collection-stats`);
// { collection_summary: { total_tracks_collected, latest_track_time, progress_toward_6_months },
//     track_counts_by_period: { all_time, last_24_hours, last_week, last_month, last_3_months, last_6_months },
//     daily_breakdown_last_30_days, collection_tips }
```

#### `GET /recently-played-tracks` — full play history

No query params. **Returns the entire `recently_played` table** — currently 5,000+ rows and growing. Cache aggressively or switch to `/top-tracks` if you only need top N.

```js
const res = await fetch(`${base}/recently-played-tracks`);
// { count, message, tracks: [{ id, spotify_song_id, track_name, duration_ms, artist_name, album_name, played_at, source, album_cover_url, genre }] }
```

#### `GET /recently-liked` — all saved tracks

No query params. **Returns the entire `recently_liked` table** — currently 7,000+ rows.

```js
const res = await fetch(`${base}/recently-liked`);
// { count, message, data: [{ id, spotify_song_id, track_name, artist_name, genre, added_at, ...album fields }] }
```

#### `GET /genre/:genre` — liked artists by genre

Path param is matched with `ILIKE '%genre%'` — partial matches work (`/genre/indie` returns indie-rock, indie-pop, etc.).

```js
const res = await fetch(`${base}/genre/indie`);
// { genre, count, message, artists: [{ artist_id, artist_name, artist_image_url, genres, track_count }] }
```

#### `GET /now-listening-to` — what's playing right now

No query params. Hits Spotify live (not the DB). Returns `data: null` when nothing is playing.

```js
const res = await fetch(`${base}/now-listening-to`);
// { data: null | { id, timestamp, progress_ms, item: { name, popularity, album: {...} } }, message }
```

#### `PATCH /mostPlayedTracks/track/:spotify_song_id` — update a track's play_count

**Requires `X-API-Key` header.**

```js
await fetch(`${base}/mostPlayedTracks/track/${songId}`, {
  method: 'PATCH',
  headers: {
    'Content-Type': 'application/json',
    'X-API-Key': API_KEY,
  },
  body: JSON.stringify({ play_count: 5 }),
});
```

#### `POST /save-refresh` — store the Spotify refresh token (no auth)

This endpoint **bypasses the API key** since it's used during initial OAuth setup.

```js
await fetch(`${base}/save-refresh`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ refresh_token: 'spotify_refresh_token_here' }),
});
```

#### `POST /backfill-duration` — admin: refill duration_ms from Spotify

**Requires `X-API-Key`.** Long-running. Only useful if `recently_played` rows are missing `duration_ms`. Don't call from the frontend.

---

## Testing Your Setup

### 1. Test in Browser Console:

Open your browser console and paste:

```javascript
// Without API key (should fail)
fetch('https://api-spotify-tracks.mtejeda.co/collection-stats')
  .then(r => r.json())
  .then(console.log);
// Expected: {error: "Unauthorized"}

// With API key (should succeed)
fetch('https://api-spotify-tracks.mtejeda.co/collection-stats', {
  headers: {
    'X-API-Key': '<your_api_key>'
  }
})
  .then(r => r.json())
  .then(console.log);
// Expected: Your stats data
```

### 2. Test Your React Component:

```javascript
// TestAPI.jsx
import { useEffect, useState } from 'react';

export default function TestAPI() {
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetch('https://api-spotify-tracks.mtejeda.co/collection-stats', {
      headers: {
        'X-API-Key': process.env.NEXT_PUBLIC_SPOTIFY_API_KEY
      }
    })
      .then(res => res.json())
      .then(setData)
      .catch(setError);
  }, []);

  if (error) return <div>Error: {error.message}</div>;
  if (!data) return <div>Loading...</div>;

  return (
    <pre>{JSON.stringify(data, null, 2)}</pre>
  );
}
```

---

## Troubleshooting

### Still getting 401 Unauthorized?

**Check:**
1. API key is correct: `<your_api_key>`
2. Header name is exact: `X-API-Key` (case-sensitive)
3. Environment variable is loaded: `console.log(process.env.NEXT_PUBLIC_SPOTIFY_API_KEY)`
4. Restarted dev server after adding .env.local

### Getting CORS errors?

The backend is configured for `http://localhost:3000`. If you're using a different port, let me know and we'll update the CORS settings.

### Rate limit errors (429)?

You're making more than 100 requests per minute. This should only happen if you have multiple tabs/windows polling simultaneously.

---

## Security Notes

**Important:**
- The API key is in the frontend code, so users can see it in the browser
- This is expected for client-side apps
- The rate limiting (100 req/min per IP) prevents abuse even if someone copies the key
- For production with many users, consider moving to backend-to-backend auth

**Good practices:**
- Keep the key in environment variables (`.env.local`)
- Don't commit `.env.local` to Git
- The key is already gitignored by default in Next.js

---

## Need Help?

If you're still stuck, share:
1. Your framework (Next.js, React, Vue, etc.)
2. Where you're making the API call (component name)
3. Any error messages in the console

Happy coding!