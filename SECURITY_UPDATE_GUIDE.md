# Security Update Guide - Rate Limiting & API Key Protection

## What Was Added

Your API is now protected with **two layers of security**:

### 1. **Rate Limiting**- **Limit:** 100 requests per minute per IP address
- **Protection:** Prevents DDoS attacks and resource exhaustion
- **Response:** Returns 429 (Too Many Requests) when limit exceeded

### 2. **API Key Authentication**- **Requirement:** All requests (except `/save-refresh`) must include API key
- **Header:** `X-API-Key: your-api-key-here`
- **Protection:** Prevents unauthorized access

---

## Deployment Steps

### Step 1: Reauthenticate with AWS

```bash
aws sso login
# Or if using IAM credentials:
# aws configure
```

### Step 2: Add API_KEY to AWS Secrets Manager

Run the helper script:

```bash
./add-api-key-secret.sh
```

This will:
- Create the `spotify-track-db/API_KEY` secret in AWS Secrets Manager
- Display your API key (save this!)

### Step 3: Deploy the Updated Application

```bash
./deploy-to-ecs.sh
```

This will:
1. Build Docker image with rate limiting
2. Push to ECR
3. Update task definition with API_KEY secret
4. Deploy to ECS

---

## Your API Key

```
<your_api_key>
```

**IMPORTANT: Keep this secret! Don't commit it to Git.**

---

## Frontend Integration

### Add API Key to Your Frontend Requests

#### JavaScript/TypeScript Example:

```javascript
// In your frontend code
const API_KEY = '<your_api_key>';
const API_URL = 'https://api-spotify-tracks.mtejeda.co';

// Fetch example
fetch(`${API_URL}/now-listening-to`, {
  headers: {
    'X-API-Key': API_KEY
  }
})
.then(res => res.json())
.then(data => console.log(data));
```

#### Axios Example:

```javascript
import axios from 'axios';

const api = axios.create({
  baseURL: 'https://api-spotify-tracks.mtejeda.co',
  headers: {
    'X-API-Key': '<your_api_key>'
  }
});

// Use it
api.get('/now-listening-to')
  .then(res => console.log(res.data));
```

#### Environment Variables (Recommended):

```bash
# .env.local
NEXT_PUBLIC_SPOTIFY_API_URL=https://api-spotify-tracks.mtejeda.co
NEXT_PUBLIC_SPOTIFY_API_KEY=<your_api_key>
```

```javascript
// In your code
const API_KEY = process.env.NEXT_PUBLIC_SPOTIFY_API_KEY;
const API_URL = process.env.NEXT_PUBLIC_SPOTIFY_API_URL;
```

---

## Testing

### Test Rate Limiting:

```bash
# This should succeed (first request)
curl -H "X-API-Key: <your_api_key>" \
  https://api-spotify-tracks.mtejeda.co/collection-stats

# Try 101+ requests in 1 minute - should get 429 error
for i in {1..101}; do
  curl -H "X-API-Key: <your_api_key>" \
    https://api-spotify-tracks.mtejeda.co/collection-stats
done
```

### Test API Key Protection:

```bash
# Without API key - should get 401 Unauthorized
curl https://api-spotify-tracks.mtejeda.co/collection-stats

# With API key - should succeed
curl -H "X-API-Key: <your_api_key>" \
  https://api-spotify-tracks.mtejeda.co/collection-stats
```

---

## What Each Layer Protects Against

### Rate Limiting Protects Against:
**DDoS attacks** - Can't overwhelm server with requests
**Resource exhaustion** - Limits CPU/memory usage
**Cost attacks** - Caps AWS bills from excessive requests
**Database overload** - Prevents too many DB queries

### API Key Protects Against:
**Unauthorized access** - Only your frontend can access
**Public scraping** - Random users can't harvest data
**Bot attacks** - Automated tools can't access without key
**Abuse** - Must know the key to use the API

### Combined Protection:
- Even if someone gets your API key, they can only make 100 req/min
- Rate limiting works per IP, so multiple IPs with same key are still limited individually
- `/save-refresh` endpoint bypasses API key (for initial token setup only)

---

## Adjusting Rate Limits

If you need to change the rate limit, edit [cmd/server/main.go](cmd/server/main.go):

```go
// Current: 100 requests per minute
rate := limiter.Rate{
    Period: 1 * time.Minute,
    Limit:  100,
}

// Example: 200 requests per minute
rate := limiter.Rate{
    Period: 1 * time.Minute,
    Limit:  200,
}

// Example: 10 requests per second
rate := limiter.Rate{
    Period: 1 * time.Second,
    Limit:  10,
}
```

Then redeploy with `./deploy-to-ecs.sh`

---

## Rotating API Key

To change your API key:

```bash
# 1. Generate new key
NEW_KEY=$(openssl rand -base64 32)
echo $NEW_KEY

# 2. Update in AWS Secrets Manager
aws secretsmanager put-secret-value \
    --secret-id "spotify-track-db/API_KEY" \
    --secret-string "$NEW_KEY" \
    --region us-east-2

# 3. Force new deployment
aws ecs update-service \
    --cluster spotify-cluster \
    --service spotify-track-db \
    --force-new-deployment \
    --region us-east-2

# 4. Update your frontend with new key
```

---

## Security Best Practices

### DO:
- Keep your API key in environment variables
- Use HTTPS only (already configured)
- Monitor AWS CloudWatch logs for suspicious activity
- Rotate API key periodically (every 90 days)

### DON'T:
- Commit API key to Git
- Share API key publicly
- Use HTTP (only HTTPS)
- Hard-code API key in frontend source

---

## Troubleshooting

### "401 Unauthorized" Error
**Cause:** Missing or wrong API key
**Solution:** Check `X-API-Key` header value matches

### "429 Too Many Requests" Error
**Cause:** Rate limit exceeded
**Solution:** Wait 60 seconds or increase rate limit

### Frontend Can't Connect
**Cause:** CORS or API key issue
**Solution:**
1. Check browser console for errors
2. Verify API key is correct
3. Check CORS allowed origin in main.go

---

## Cost Impact

**Additional Monthly Costs:**
- Rate limiting: $0 (in-memory, no extra cost)
- API_KEY secret: $0.40/month (AWS Secrets Manager)

**Total additional cost: ~$0.40/month**

---

## Summary

Your API is now protected with:
- Rate limiting (100 req/min per IP)
- API key authentication
- CORS protection
- HTTPS encryption
- Secure secret management

**Next steps:**
1. Reauthenticate with AWS
2. Run `./add-api-key-secret.sh`
3. Run `./deploy-to-ecs.sh`
4. Update your frontend with API key
5. Test the endpoints!

---

**Questions?** Check the [AWS_DEPLOYMENT_SESSION.md](AWS_DEPLOYMENT_SESSION.md) for more details.
