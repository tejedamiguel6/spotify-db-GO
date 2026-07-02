# API Keys & Environment Variables

## For Frontend Development

### Environment Variable Name:
```
NEXT_PUBLIC_SPOTIFY_BACKEND_API_KEY
```

### Where to Find the API Key:
The actual API key value is stored in `.env.api-key` (gitignored for security)

### Setup for Your Frontend:

1. **Copy the key from `.env.api-key`**
2. **Add to your frontend's `.env.local`:**
   ```bash
   NEXT_PUBLIC_SPOTIFY_BACKEND_API_KEY=<paste_value_here>
   ```

3. **Use in your code:**
   ```javascript
   // For read operations (public - no key needed)
   fetch('https://api-spotify-tracks.mtejeda.co/now-listening-to')
     .then(r => r.json())
     .then(data => console.log(data));

   // For write operations (protected - key required)
   const API_KEY = process.env.NEXT_PUBLIC_SPOTIFY_BACKEND_API_KEY;

   fetch('https://api-spotify-tracks.mtejeda.co/mostPlayedTracks/track/SONG_ID', {
     method: 'PATCH',
     headers: {
       'Content-Type': 'application/json',
       'X-API-Key': API_KEY
     },
     body: JSON.stringify({ play_count: 10 })
   });
   ```

---

## Current API Configuration

### Public Endpoints (No Authentication Required):
**All GET requests** - Anyone can read your music data
- `/now-listening-to`
- `/collection-stats`
- `/recently-played-tracks`
- `/recently-liked`
- `/genre/:genre`

### Protected Endpoints (API Key Required):
**POST, PATCH, DELETE requests** - Only you can modify data
- `POST /save-refresh`
- `PATCH /mostPlayedTracks/track/:id`

### Rate Limiting:
- **100 requests per minute** per IP address
- Applies to all users (including you)

---

## Security Notes

### What's Safe:
- GET endpoints are public (your music data is meant to be showcased)
- Rate limiting prevents abuse
- Write operations are protected

### What to Keep Secret:
- The API key in `.env.api-key`
- Never commit `.env.local` or `.env.api-key` to Git
- Don't expose the key in client-side code (it's okay for Next.js env vars since it's only for your portfolio)

### Rotating the API Key:
If you need to change the key:
1. Generate new key: `openssl rand -base64 32`
2. Update in AWS Secrets Manager
3. Update `.env.api-key` locally
4. Redeploy backend: `./deploy-to-ecs.sh`
5. Update frontend `.env.local`

---

## Quick Reference

| Variable Name | Used For | Where |
|--------------|----------|-------|
| `NEXT_PUBLIC_SPOTIFY_BACKEND_API_KEY` | Backend write operations | Frontend `.env.local` |
| `API_KEY` | Backend authentication | AWS Secrets Manager |
| `DATABASE_URL` | Database connection | AWS Secrets Manager |
| `SPOTIFY_CLIENT_ID` | Spotify API | AWS Secrets Manager |
| `SPOTIFY_CLIENT_SECRET` | Spotify API | AWS Secrets Manager |

---

## For Local Development

If testing locally with the backend:
```bash
# In your backend .env file
API_KEY=<value_from_.env.api-key>
DATABASE_URL=<your_database_url>
SPOTIFY_CLIENT_ID=<your_spotify_client_id>
SPOTIFY_CLIENT_SECRET=<your_spotify_client_secret>
```

**Remember:** Never commit `.env` files to Git!
