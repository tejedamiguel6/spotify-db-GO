# 🎵 Spotify Track Database

A Go-based REST API service that automatically collects and tracks your Spotify listening history with real-time analytics and comprehensive data management.

## 🌐 Production Deployment

**Live API:** https://api-spotify-tracks.mtejeda.co

**Status:** Deployed on AWS ECS Fargate with HTTPS, custom domain, and automatic data collection.

## ✨ Features

- **🔄 Automatic Data Collection**: Optimized background collection every 5 minutes during active hours
- **📊 Real-time Analytics**: Track listening patterns with detailed statistics and insights
- **🎧 Current Listening**: Get real-time information about what you're currently playing
- **📈 Historical Data**: Comprehensive storage and retrieval of your music history
- **🔧 RESTful API**: Clean, well-structured API endpoints for all operations
- **⚡ High Performance**: Built with Go and PostgreSQL for optimal performance
- **🛡️ Secure**: Rate limiting (100 req/min), API key protection for writes, and proper token management
- **🌍 Public Portfolio Ready**: GET endpoints are public, write operations protected

## 🏗️ Project Structure

```
go-spotify-track-db/
├── cmd/
│   └── server/           # Application entry point
│       └── main.go
├── internal/
│   ├── handlers/         # HTTP request handlers
│   │   └── tracks.go
│   ├── models/           # Data structures
│   │   ├── types.go      # Struct definitions
│   │   ├── track_operations.go  # Database operations
│   │   └── track.go      # Track management
│   ├── repository/       # Database layer
│   │   ├── db.go         # Database connection & queries
│   │   └── helpers.go    # Database utilities
│   └── services/         # External services
│       └── client.go     # Spotify API client
├── api-test/            # HTTP test files
└── go.mod              # Go dependencies
```

## 🚀 Quick Start

### Prerequisites

- **Go 1.23+** - [Download Go](https://golang.org/dl/)
- **PostgreSQL** - Local installation or cloud service (Neon, Supabase, etc.)
- **Spotify Developer Account** - [Create one here](https://developer.spotify.com/)

### 1. Clone the Repository

```bash
git clone <repository-url>
cd go-spotify-track-db
```

### 2. Set Up Spotify App

1. Go to [Spotify Developer Dashboard](https://developer.spotify.com/dashboard)
2. Create a new app
3. Note your `Client ID` and `Client Secret`
4. Add redirect URI: `http://localhost:8080/callback` (or your preferred callback)

### 3. Environment Configuration

Copy the example environment file and fill in your credentials:

```bash
cp .env.example .env
```

Then edit `.env` with your actual credentials:

```env
# Database Configuration
DATABASE_URL=postgresql://username:password@host:port/database

# Spotify API Credentials
SPOTIFY_CLIENT_ID=your_spotify_client_id
SPOTIFY_CLIENT_SECRET=your_spotify_client_secret

# Optional: Custom Port (defaults to 8080)
PORT=8080
```

### 4. Database Setup

Create the required PostgreSQL tables:

```sql
-- Main tracks collection table
CREATE TABLE tracks_on_repeat (
    id SERIAL PRIMARY KEY,
    spotify_song_id VARCHAR(255) UNIQUE NOT NULL,
    track_name VARCHAR(255) NOT NULL,
    artist_name VARCHAR(255) NOT NULL,
    album_name VARCHAR(255) NOT NULL,
    genre VARCHAR(100),
    preview_url TEXT,
    album_cover_url TEXT,
    play_count INTEGER DEFAULT 0,
    first_played TIMESTAMP DEFAULT NOW(),
    last_played TIMESTAMP DEFAULT NOW(),
    time_of_day VARCHAR(50),
    mood VARCHAR(100),
    activity VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Recently played tracks history
CREATE TABLE recently_played (
    id SERIAL PRIMARY KEY,
    spotify_song_id VARCHAR(255) NOT NULL,
    track_name VARCHAR(255) NOT NULL,
    artist_name VARCHAR(255) NOT NULL,
    album_name VARCHAR(255) NOT NULL,
    played_at TIMESTAMP NOT NULL,
    source VARCHAR(100) DEFAULT 'spotify_api',
    album_cover_url TEXT,
    genre VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Recently liked tracks
CREATE TABLE recently_liked (
    id SERIAL PRIMARY KEY,
    spotify_song_id VARCHAR(255) NOT NULL,
    track_name VARCHAR(255) NOT NULL,
    artist_name VARCHAR(255) NOT NULL,
    album_name VARCHAR(255) NOT NULL,
    added_at TIMESTAMP NOT NULL,
    album_cover_url TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Spotify authentication tokens
CREATE TABLE spotify_auth (
    id INTEGER PRIMARY KEY DEFAULT 1,
    refresh_token TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for better performance
CREATE INDEX idx_recently_played_played_at ON recently_played(played_at);
CREATE INDEX idx_recently_played_song_id ON recently_played(spotify_song_id);
CREATE INDEX idx_tracks_repeat_song_id ON tracks_on_repeat(spotify_song_id);
```

### 5. Install Dependencies & Run

```bash
# Install dependencies
go mod tidy

# Build the application
go build ./cmd/server

# Run the server
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`

## 📡 API Endpoints

### 🎵 Track Management

#### Get Most Played Tracks
```http
GET /mostPlayedTracks
```
Returns your curated collection of favorite tracks.

#### Get Recently Played History
```http
GET /recently-played-tracks
```
Returns complete history of recently played tracks.

#### Get Current Playing Track
```http
GET /now-listening-to
```
Returns what you're currently listening to on Spotify.

#### Add Track to Collection
```http
POST /mostPlayedTracks
Content-Type: application/json

{
  "spotify_song_id": "4iV5W9uYEdYUVa79Axb7Rh",
  "track_name": "Song Title",
  "artist_name": "Artist Name",
  "album_name": "Album Name",
  "genre": "Pop",
  "preview_url": "https://...",
  "album_cover_url": "https://...",
  "play_count": 1
}
```

#### Update Track Play Count
```http
PATCH /mostPlayedTracks/track/:spotify_song_id
Content-Type: application/json

{
  "play_count": 5
}
```

### 📊 Analytics & Stats

#### Get Collection Statistics
```http
GET /collection-stats
```
Returns comprehensive analytics including:
- Track counts by time period (24h, week, month, 3mo, 6mo, all-time)
- Latest collection timestamp
- Daily breakdown for last 30 days
- Collection progress insights

### 🔐 Authentication

#### Save Refresh Token
```http
POST /save-refresh
Content-Type: application/json

{
  "refresh_token": "your_spotify_refresh_token"
}
```

## 🔧 Development

### Project Architecture

- **Handlers** (`internal/handlers/`): HTTP request processing and response formatting
- **Models** (`internal/models/`): Data structures and business logic
- **Repository** (`internal/repository/`): Database operations and connection management
- **Services** (`internal/services/`): External API integrations (Spotify)

### Key Features

1. **Automatic Background Collection**: The system runs an optimized background cron job that:
   - **Checks every 5 minutes** during active hours (6 AM - 11 PM) - optimized for database efficiency
   - **Every 15 minutes** during sleep hours (11 PM - 6 AM)
   - **Genre updates every 30 minutes** (50 tracks per batch)
   - Automatically fetches new tracks from Spotify API
   - Handles token refresh automatically with retry logic
   - Stores both recently played and recently liked tracks
   - **Optimized to stay within database quotas** (~2,500 queries/day vs previous 13,000)

2. **Smart Data Management**:
   - Prevents duplicate entries
   - Automatically updates play counts
   - Tracks first and last played timestamps
   - Maintains comprehensive listening history
   - Rate limiting protection for Spotify API calls

3. **Analytics Engine**:
   - Real-time collection statistics
   - Historical trend analysis
   - Daily breakdown views
   - Progress tracking

### Testing

Use the provided HTTP test files in `api-test/` directory:

```bash
# Test basic endpoints
curl http://localhost:8080/mostPlayedTracks
curl http://localhost:8080/collection-stats
curl http://localhost:8080/now-listening-to
```

### Building for Production

```bash
# Build optimized binary
go build -ldflags="-s -w" -o spotify-tracker cmd/server/main.go

# Run in production
./spotify-tracker
```

## 📦 Dependencies

- **[Gin](https://github.com/gin-gonic/gin)** - HTTP web framework
- **[pgx](https://github.com/jackc/pgx)** - PostgreSQL driver and toolkit
- **[godotenv](https://github.com/joho/godotenv)** - Environment variable loading
- **[CORS](https://github.com/gin-contrib/cors)** - Cross-Origin Resource Sharing

## 🔐 Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `DATABASE_URL` | PostgreSQL connection string | ✅ |
| `SPOTIFY_CLIENT_ID` | Your Spotify app's client ID | ✅ |
| `SPOTIFY_CLIENT_SECRET` | Your Spotify app's client secret | ✅ |
| `PORT` | Server port (default: 8080) | ❌ |

## 🚀 Production Deployment (AWS ECS)

### Current Infrastructure

**Deployed to AWS ECS Fargate in `us-east-2`:**
- **API URL**: https://api-spotify-tracks.mtejeda.co
- **Task Definition**: `spotify-track-db:8` (optimized version)
- **Resources**: 0.25 vCPU, 512 MB RAM
- **Monthly Cost**: ~$49/month (AWS: $30 + Neon: $19)

### Security & Access Control

**Public Access (No Auth Required):**
- ✅ All **GET** endpoints - Anyone can read your music data for portfolio
- ✅ Rate limited to 100 requests/minute per IP

**Protected Access (API Key Required):**
- 🔐 **POST/PATCH/DELETE** endpoints - Write operations require `X-API-Key` header
- 🔐 API key stored in AWS Secrets Manager

### Deployment

**Quick Deployment:**
```bash
./deploy-to-ecs.sh
```

This script:
1. Builds Docker image for `linux/amd64` (required for AWS Fargate)
2. Pushes to ECR
3. Registers new task definition
4. Forces ECS service deployment (zero-downtime rolling update)

**See detailed guides:**
- [AWS_DEPLOYMENT_SESSION.md](AWS_DEPLOYMENT_SESSION.md) - Complete deployment history
- [AWS_DEPLOYMENT_GUIDE_FOR_DEVELOPERS.md](AWS_DEPLOYMENT_GUIDE_FOR_DEVELOPERS.md) - Educational guide
- [DEPLOYMENT.md](DEPLOYMENT.md) - Quick reference
- [DATABASE_OPTIMIZATION.md](DATABASE_OPTIMIZATION.md) - Database quota optimization details

### Recent Optimizations (Jan 2026)

**Problem:** Hit Neon's 5 GB/month data transfer quota due to aggressive polling (90 seconds)

**Solution Applied:**
- Reduced cron frequency: **90 seconds → 5 minutes** (70% reduction)
- Optimized genre updates: **Every run → Every 30 minutes** (83% reduction)
- Reduced batch size: **150 tracks → 50 tracks** (67% reduction)

**Results:**
- **Before**: ~13,000 queries/day (~6-8 GB/month) ❌
- **After**: ~2,500 queries/day (~1.8 GB/month) ✅
- **Impact**: 80% reduction in database load, well within quotas

### Frontend Integration

For portfolio sites, see [PORTFOLIO_SETUP.md](PORTFOLIO_SETUP.md) for:
- Public API access patterns
- React component examples
- Real-time "now playing" widgets
- CORS configuration

### Monitoring

```bash
# View logs
aws logs tail /ecs/spotify-track-db --follow --region us-east-2

# Check service status
aws ecs describe-services --cluster spotify-cluster --services spotify-track-db --region us-east-2

# View current task
aws ecs list-tasks --cluster spotify-cluster --service-name spotify-track-db --region us-east-2
```

## 📝 Notes

- **Spotify API Limits**: The system respects Spotify's rate limits and handles them gracefully with retry logic
- **Data Retention**: All data is stored permanently; implement your own cleanup policies if needed
- **Token Management**: Refresh tokens are automatically managed and stored securely in database
- **Background Processing**: Optimized cron job starts automatically and runs every 5 minutes
- **Database**: Requires Neon Launch plan ($19/month) or equivalent with 100 GB data transfer

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 📚 Additional Documentation

| Document | Purpose |
|----------|---------|
| [AWS_DEPLOYMENT_SESSION.md](AWS_DEPLOYMENT_SESSION.md) | Complete AWS deployment session history |
| [AWS_DEPLOYMENT_GUIDE_FOR_DEVELOPERS.md](AWS_DEPLOYMENT_GUIDE_FOR_DEVELOPERS.md) | Educational AWS guide for beginners |
| [DEPLOYMENT.md](DEPLOYMENT.md) | Quick deployment reference |
| [PORTFOLIO_SETUP.md](PORTFOLIO_SETUP.md) | Frontend integration guide for portfolio sites |
| [DATABASE_OPTIMIZATION.md](DATABASE_OPTIMIZATION.md) | Database query optimization details |
| [SECURITY_UPDATE_GUIDE.md](SECURITY_UPDATE_GUIDE.md) | Security features and API key setup |
| [API_KEYS_README.md](API_KEYS_README.md) | Environment variables and API key reference |
| [FRONTEND_INTEGRATION.md](FRONTEND_INTEGRATION.md) | Detailed frontend integration examples |
| [BACKUP_GUIDE.md](BACKUP_GUIDE.md) | Database backup and recovery procedures |
| [RECOVERY.md](RECOVERY.md) | Database recovery instructions |

## ⚠️ Current Status (For Next Session)

**Deployment Status:**
- ✅ Optimized code deployed (task-definition:8)
- ✅ Running on AWS ECS Fargate
- ✅ Security: Rate limiting + API key protection
- ✅ CORS: Public GET access, protected writes

**Database Status:**
- ❌ **Neon database currently SUSPENDED** (exceeded 5 GB free tier quota)
- ⏳ **Waiting for upgrade to Launch plan** ($19/month, 100 GB transfer)
- 📊 Optimizations deployed will reduce usage by 80% (from 13K to 2.5K queries/day)

**Next Steps:**
1. Upgrade Neon to Launch plan - database will reactivate instantly
2. Verify cron job runs at new 5-minute intervals
3. Monitor database usage to confirm optimization works
4. Build frontend portfolio to showcase the data

**Important Files:**
- `.env.api-key` - Contains backend API key (gitignored)
- `deploy-to-ecs.sh` - One-command deployment script
- `cmd/server/main.go` - Main application entry with security middleware
- `internal/handlers/tracks.go:132` - Cron interval configuration (5 minutes)

## 🔗 Links

- [Spotify Web API Documentation](https://developer.spotify.com/documentation/web-api/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Gin Framework Documentation](https://gin-gonic.com/docs/)
- [Neon Database](https://neon.tech)
- [AWS ECS Documentation](https://docs.aws.amazon.com/ecs/)

---

**Built with ❤️ and Go** 🎵
