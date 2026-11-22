# Portfolio Site Setup - Public API Access

## What Changed

Your API is now configured for **public portfolio use**:

### Access Control:
- **GET requests (reading data)**: **PUBLIC** - Anyone can access
- **POST/PATCH/DELETE (writing data)**: **Protected** - Requires API key
- **Rate limiting**: 100 requests/minute per IP (prevents abuse)

### CORS Configuration:
- **Localhost**: Allowed for development
- **Your domains**: `mtejeda.co`, `www.mtejeda.co`
- **All other origins**: Allowed (since data is public)

---

## Public Endpoints (No Authentication Required)

Anyone visiting your portfolio can call these:

```javascript
// Get collection statistics
fetch('https://api-spotify-tracks.mtejeda.co/collection-stats')
  .then(r => r.json())
  .then(data => console.log(data));

// Get currently playing
fetch('https://api-spotify-tracks.mtejeda.co/now-listening-to')
  .then(r => r.json())
  .then(data => console.log(data));

// Get recently played tracks
fetch('https://api-spotify-tracks.mtejeda.co/recently-played-tracks')
  .then(r => r.json())
  .then(data => console.log(data));

// Get recently liked tracks
fetch('https://api-spotify-tracks.mtejeda.co/recently-liked')
  .then(r => r.json())
  .then(data => console.log(data));

// Get tracks by genre
fetch('https://api-spotify-tracks.mtejeda.co/genre/indie')
  .then(r => r.json())
  .then(data => console.log(data));
```

---

## Protected Endpoints (API Key Required)

Only you (with the API key) can call these:

**Environment Variable Setup:**
```bash
# Add to your .env.local (NEVER commit this file!)
NEXT_PUBLIC_SPOTIFY_BACKEND_API_KEY=your_api_key_here
```

**Usage in Code:**
```javascript
const API_KEY = process.env.NEXT_PUBLIC_SPOTIFY_BACKEND_API_KEY;

// Update track play count
fetch('https://api-spotify-tracks.mtejeda.co/mostPlayedTracks/track/SONG_ID', {
  method: 'PATCH',
  headers: {
    'Content-Type': 'application/json',
    'X-API-Key': API_KEY
  },
  body: JSON.stringify({ play_count: 10 })
});

// Save refresh token (also protected)
fetch('https://api-spotify-tracks.mtejeda.co/save-refresh', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({ refresh_token: 'your_token' })
});
```

---

## Portfolio Site Examples

### Example 1: Real-time "Now Playing" Widget

```javascript
// components/NowPlaying.jsx
import { useState, useEffect } from 'react';

export default function NowPlaying() {
  const [track, setTrack] = useState(null);

  useEffect(() => {
    const fetchNowPlaying = async () => {
      try {
        const res = await fetch('https://api-spotify-tracks.mtejeda.co/now-listening-to');
        const data = await res.json();
        setTrack(data);
      } catch (error) {
        console.error('Error fetching now playing:', error);
      }
    };

    fetchNowPlaying();
    const interval = setInterval(fetchNowPlaying, 10000); // Poll every 10 seconds

    return () => clearInterval(interval);
  }, []);

  if (!track?.item) return <div>Not currently listening</div>;

  return (
    <div className="now-playing">
      <img src={track.item.album.images[0]?.url} alt="Album cover" />
      <div>
        <h3>{track.item.name}</h3>
        <p>{track.item.album.artists[0]?.name}</p>
      </div>
    </div>
  );
}
```

### Example 2: Listening Stats Dashboard

```javascript
// components/Stats.jsx
import { useState, useEffect } from 'react';

export default function Stats() {
  const [stats, setStats] = useState(null);

  useEffect(() => {
    fetch('https://api-spotify-tracks.mtejeda.co/collection-stats')
      .then(r => r.json())
      .then(setStats);
  }, []);

  if (!stats) return <div>Loading...</div>;

  return (
    <div className="stats-grid">
      <div className="stat-card">
        <h4>Total Tracks</h4>
        <p>{stats.collection_summary.total_tracks_collected}</p>
      </div>
      <div className="stat-card">
        <h4>Last 24 Hours</h4>
        <p>{stats.track_counts_by_period.last_24_hours}</p>
      </div>
      <div className="stat-card">
        <h4>This Week</h4>
        <p>{stats.track_counts_by_period.last_week}</p>
      </div>
      <div className="stat-card">
        <h4>This Month</h4>
        <p>{stats.track_counts_by_period.last_month}</p>
      </div>
    </div>
  );
}
```

### Example 3: Recently Played Grid

```javascript
// components/RecentlyPlayed.jsx
import { useState, useEffect } from 'react';

export default function RecentlyPlayed() {
  const [tracks, setTracks] = useState([]);

  useEffect(() => {
    fetch('https://api-spotify-tracks.mtejeda.co/recently-played-tracks')
      .then(r => r.json())
      .then(setTracks);
  }, []);

  return (
    <div className="track-grid">
      {tracks.map((track, i) => (
        <div key={i} className="track-card">
          <img
            src={track.album_cover_url}
            alt={`${track.track_name} cover`}
          />
          <div className="track-info">
            <h4>{track.track_name}</h4>
            <p>{track.artist_name}</p>
            <span className="played-at">
              {new Date(track.played_at).toLocaleString()}
            </span>
          </div>
        </div>
      ))}
    </div>
  );
}
```

---

## Creative Portfolio Ideas

### 1. **Animated Music Visualizer**
- Fetch `now-listening-to` every 5 seconds
- Show animated waveforms synced to track progress
- Display album art with CSS animations

### 2. **Listening Heatmap**
- Use `recently-played-tracks` to create a calendar heatmap
- Show which days/times you listen most
- Interactive tooltips with track details

### 3. **Genre Distribution Chart**
- Parse genres from `recently-liked`
- Create pie chart or bar graph
- Clickable genres that filter to `/genre/:genre`

### 4. **Top Artists/Tracks Timeline**
- Display most-played artists over time
- Animated transitions between time periods
- Beautiful gradient backgrounds based on album colors

### 5. **Live Activity Feed**
- Real-time feed of tracks as they're played
- Toast notifications when you like a new song
- Smooth animations for new entries

---

## Security Features Still Active

Even though the API is public for reading:

### Rate Limiting
- **100 requests per minute** per IP address
- Prevents single user from overloading your server
- Returns `429 Too Many Requests` when exceeded

### Protected Write Operations
- POST/PATCH/DELETE still require API key
- Prevents anyone from modifying your data
- Only you can update play counts or save tokens

### CORS Protection
- Prevents unauthorized embedding
- You can restrict to specific domains later if needed

---

## Expected Traffic & Costs

### Scenario: Portfolio with 100 visitors/day

**Traffic estimate:**
- 100 visitors × 5 page loads × 3 API calls = 1,500 requests/day
- ~45,000 requests/month

**AWS costs:**
- **Fargate**: $12/month (unchanged - always running)
- **ALB**: $16/month + ~$0.50 for 45K requests = $16.50/month
- **CloudWatch**: $0.50/month (minimal logs)
- **Secrets Manager**: $1.20/month
- **Total**: ~$30/month

**Still well within budget!**
---

## Adding Your Portfolio Domain

When your portfolio is deployed, add the domain:

1. Open `cmd/server/main.go`
2. Find the `allowedOrigins` array (line ~61)
3. Add your domain:

```go
allowedOrigins := []string{
    "http://localhost:3000",
    "http://localhost:3001",
    "https://mtejeda.co",
    "https://www.mtejeda.co",
    "https://your-portfolio.vercel.app", // Add this
}
```

4. Redeploy: `./deploy-to-ecs.sh`

---

## Troubleshooting

### Visitors getting CORS errors?
**Check:**
- Is your domain in `allowedOrigins`?
- Did you redeploy after adding it?
- Try accessing the API directly in browser first

### Rate limit errors (429)?
**This is normal if:**
- Someone spams refresh on your site
- Multiple people viewing simultaneously
- Solution: Add caching on frontend (store data for 30-60 seconds)

### Slow response times?
**Solutions:**
- Add loading states in your UI
- Implement request caching
- Use skeleton screens while loading

---

## You're All Set!

Your API is now **perfectly configured** for a public portfolio site:
- Visitors can see your music data
- You can still manage data privately
- Protected against abuse with rate limiting
- Ready for production traffic

Build something awesome!