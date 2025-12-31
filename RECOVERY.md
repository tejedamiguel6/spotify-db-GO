# Database Recovery Guide

## Overview
This guide helps you recover your Spotify track database after a crash. Your backup is from June 21st, and we need to:

1. **Recreate missing database tables** (especially `recently_liked`)
2. **Recover your listening data** from June 21st to now
3. **Ensure the cron job continues working**

## Recovery Steps

### Step 1: Build the recovery tools
```bash
go mod tidy
go build -o bin/server ./cmd/server
go build -o bin/recovery-safe ./cmd/recovery-safe
```

### Step 2: Run the server once to create missing tables
The server will automatically create missing database tables (including `recently_liked`) when it starts:

```bash
./bin/server
```

**Wait for this message:** `🏗️ Database schema verified/created`

Then stop the server with `Ctrl+C`.

### Step 3: Run the SAFE recovery script (RECOMMENDED)
```bash
./bin/recovery-safe
```

**This is the SAFE version that prevents Spotify API rate limits:**
- ✅ **Rate limiting protection** - Won't exceed Spotify's API limits
- ✅ **Exponential backoff** - Handles rate limit responses gracefully  
- ✅ **Retry logic** - Automatically retries failed requests
- ✅ **Progress tracking** - Shows detailed progress during recovery
- ✅ **Smaller batch sizes** - Processes 20 tracks per page vs 50
- ✅ **Built-in delays** - 2 second delays between API calls

This script will:
- ✅ Fetch your current recently played tracks (last ~50 available from Spotify)
- ✅ Recover ALL your saved/liked tracks from Spotify and populate `recently_liked`
- ✅ Focus on tracks added since June 21st, 2024

### Step 4: Start your server normally
```bash
./bin/server
```

The cron job will now run every 90 seconds during active hours (6 AM - 11 PM) with **RATE LIMITING PROTECTION** and:
- ✅ **Collect recently played tracks** (with retry logic)
- ✅ **Update recently_liked tracks** (was missing, now fixed!)
- ✅ **Get currently playing track info** 
- ✅ **Fill in genre information** (with smart rate limit handling)

**New Rate Limiting Features:**
- 🛡️ **60 requests per minute limit** (conservative vs Spotify's ~100)
- 🔄 **Automatic retries** with exponential backoff
- ⏳ **Smart delays** between API calls
- 🚨 **Rate limit detection** and graceful handling
- 💾 **Marks rate-limited data** for later retry

## What the Recovery Script Does

### Recently Played Recovery
⚠️ **Important:** Spotify's API only stores ~50 recent tracks at any time. The script will collect what's currently available, but cannot recover historical "recently played" data from June 21st.

### Recently Liked Recovery
✅ **Full Recovery:** This can recover ALL your liked/saved tracks, including those added since June 21st, because Spotify stores your complete saved tracks history.

## Verification

After running the recovery, check your data:

1. **Visit:** `http://localhost:8080/recently-liked`
2. **Check stats:** `http://localhost:8080/collection-stats`

## Cron Job Details

The background cron job (running automatically when server starts) does:

- **CollectRecentTracks()** - Gets recently played tracks
- **CollectSavedTracks()** - Gets new liked tracks ✅ 
- **GetCurrentlyPlaying()** - Gets what's currently playing
- **GetGenreOfRecentlyLiked()** - Fills in genre info for liked tracks

## Files Created

- `cmd/recovery/main.go` - Recovery script
- `cmd/init-db/main.go` - Database initialization (alternative)
- Updated `internal/repository/db.go` - Auto-creates missing tables

## Database Tables

After recovery, you'll have:

- `recently_played` - Your listening history
- **`recently_liked`** - Your saved/liked tracks ✅ (was missing)
- `spotify_auth` - Authentication tokens
- `tracks_on_repeat` - Legacy table (optional)

The cron job will now properly populate all tables continuously.