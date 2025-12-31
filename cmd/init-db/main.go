package main

import (
	"context"
	"fmt"
	"log"

	"example.com/spotifydb/internal/repository"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	// Initialize database connection
	repository.InitDB()

	fmt.Println("🔧 Initializing database schema...")

	// Create all required tables
	if err := createTables(); err != nil {
		log.Fatal("❌ Failed to create tables:", err)
	}

	fmt.Println("✅ Database schema initialization complete!")
}

func createTables() error {
	ctx := context.Background()

	// Create spotify_auth table
	authTable := `
	CREATE TABLE IF NOT EXISTS spotify_auth (
		id INT PRIMARY KEY DEFAULT 1,
		refresh_token TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);`

	if _, err := repository.Pool.Exec(ctx, authTable); err != nil {
		return fmt.Errorf("failed to create spotify_auth table: %v", err)
	}
	fmt.Println("✅ Created/verified spotify_auth table")

	// Create recently_played table
	recentlyPlayedTable := `
	CREATE TABLE IF NOT EXISTS recently_played (
		id SERIAL PRIMARY KEY,
		spotify_song_id VARCHAR(255) NOT NULL,
		track_name TEXT NOT NULL,
		artist_name TEXT,
		album_name TEXT,
		album_cover_url TEXT,
		genre TEXT,
		played_at TIMESTAMP NOT NULL,
		source VARCHAR(50) DEFAULT 'cron',
		created_at TIMESTAMP DEFAULT NOW(),
		UNIQUE(spotify_song_id, played_at)
	);`

	if _, err := repository.Pool.Exec(ctx, recentlyPlayedTable); err != nil {
		return fmt.Errorf("failed to create recently_played table: %v", err)
	}
	fmt.Println("✅ Created/verified recently_played table")

	// Create recently_liked table
	recentlyLikedTable := `
	CREATE TABLE IF NOT EXISTS recently_liked (
		id SERIAL PRIMARY KEY,
		spotify_song_id VARCHAR(255) UNIQUE NOT NULL,
		track_name TEXT NOT NULL,
		track_popularity VARCHAR(10),
		album_name TEXT,
		album_type VARCHAR(50),
		album_cover_url TEXT,
		album_release_date VARCHAR(20),
		album_release_date_precision VARCHAR(10),
		artist_name TEXT,
		artist_id VARCHAR(255),
		artist_href TEXT,
		artist_uri TEXT,
		album_total_tracks INTEGER,
		album_cover_width INTEGER,
		album_cover_height INTEGER,
		genre TEXT,
		added_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT NOW()
	);`

	if _, err := repository.Pool.Exec(ctx, recentlyLikedTable); err != nil {
		return fmt.Errorf("failed to create recently_liked table: %v", err)
	}
	fmt.Println("✅ Created/verified recently_liked table")

	// Create tracks_on_repeat table (legacy, keeping for compatibility)
	tracksOnRepeatTable := `
	CREATE TABLE IF NOT EXISTS tracks_on_repeat (
		id SERIAL PRIMARY KEY,
		spotify_song_id VARCHAR(255) UNIQUE NOT NULL,
		track_name TEXT NOT NULL,
		artist_name TEXT,
		album_name TEXT,
		genre TEXT,
		preview_url TEXT,
		album_cover_url TEXT,
		play_count INTEGER DEFAULT 1,
		first_played TIMESTAMP,
		last_played TIMESTAMP,
		month_year VARCHAR(10),
		time_of_day VARCHAR(20),
		mood VARCHAR(50),
		activity VARCHAR(50),
		created_at TIMESTAMP DEFAULT NOW()
	);`

	if _, err := repository.Pool.Exec(ctx, tracksOnRepeatTable); err != nil {
		return fmt.Errorf("failed to create tracks_on_repeat table: %v", err)
	}
	fmt.Println("✅ Created/verified tracks_on_repeat table")

	// Create indexes for better performance
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_recently_played_played_at ON recently_played(played_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_recently_played_spotify_id ON recently_played(spotify_song_id);",
		"CREATE INDEX IF NOT EXISTS idx_recently_liked_added_at ON recently_liked(added_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_recently_liked_spotify_id ON recently_liked(spotify_song_id);",
		"CREATE INDEX IF NOT EXISTS idx_recently_liked_genre ON recently_liked(genre);",
		"CREATE INDEX IF NOT EXISTS idx_recently_liked_artist_id ON recently_liked(artist_id);",
	}

	for _, indexSQL := range indexes {
		if _, err := repository.Pool.Exec(ctx, indexSQL); err != nil {
			fmt.Printf("⚠️  Warning: Failed to create index: %v\n", err)
		}
	}
	fmt.Println("✅ Created/verified database indexes")

	return nil
}