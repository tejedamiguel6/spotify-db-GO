# 🎵 Spotify Track Database

A Go-based REST API service that automatically collects and tracks your Spotify listening history with real-time analytics and comprehensive data management.

## ✨ Features

- **🔄 Automatic Data Collection**: Continuous background collection of your Spotify listening history
- **📊 Real-time Analytics**: Track listening patterns with detailed statistics and insights
- **🎧 Current Listening**: Get real-time information about what you're currently playing
- **📈 Historical Data**: Comprehensive storage and retrieval of your music history
- **🔧 RESTful API**: Clean, well-structured API endpoints for all operations
- **⚡ High Performance**: Built with Go and PostgreSQL for optimal performance
- **🛡️ Secure**: Proper token management and refresh handling

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

- **Go 1.24+** - [Download Go](https://golang.org/dl/)
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

1. **Automatic Background Collection**: The system runs a background cron job that:
   - Checks every 1.5 minutes during active hours (6 AM - 11 PM)
   - Automatically fetches new tracks from Spotify API
   - Handles token refresh automatically
   - Stores both recently played and recently liked tracks

2. **Smart Data Management**: 
   - Prevents duplicate entries
   - Automatically updates play counts
   - Tracks first and last played timestamps
   - Maintains comprehensive listening history

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

## 📝 Notes

- **Spotify API Limits**: The system respects Spotify's rate limits and handles them gracefully
- **Data Retention**: All data is stored permanently; implement your own cleanup policies if needed
- **Token Management**: Refresh tokens are automatically managed and stored securely
- **Background Processing**: The cron job starts automatically when the server starts

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🔗 Links

- [Spotify Web API Documentation](https://developer.spotify.com/documentation/web-api/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Gin Framework Documentation](https://gin-gonic.com/docs/)

---

**Built with ❤️ and Go** 🎵
