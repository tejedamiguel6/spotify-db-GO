package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var Pool *pgxpool.Pool

func InitDB() {
	fmt.Println("🔌  Connecting to database…")

	// load .env for DATABASE_URL
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env:", err)
	}

	dsn := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	Pool = pool

	// Quick sanity check
	var greeting string
	if err := pool.QueryRow(context.Background(), "SELECT 'Hello, world!'").Scan(&greeting); err != nil {
		log.Fatalf("QueryRow failed: %v\n", err)
	}
	fmt.Println("✅", greeting)

	// 1) Ensure the auth table exists and seed placeholder row if needed
	createAuthTable(pool)
	seedAuthRow(pool)

	// 2) Drop only our play-tracking tables, leaving spotify_auth intact
	// dropDomainTables(pool)

	// 3) Recreate the domain tables from scratch
	createDomainTables(pool)

	fmt.Println("🎉  Database ready")
}

func createAuthTable(pool *pgxpool.Pool) {
	sql := `
    CREATE TABLE IF NOT EXISTS spotify_auth (
      id             SERIAL PRIMARY KEY,
      refresh_token  TEXT   NOT NULL,
      updated_at     TIMESTAMP NOT NULL DEFAULT NOW()
    );
    `
	if _, err := pool.Exec(context.Background(), sql); err != nil {
		log.Fatalf("Failed to create spotify_auth table: %v\n", err)
	}
}

func seedAuthRow(pool *pgxpool.Pool) {
	sql := `
    INSERT INTO spotify_auth (id, refresh_token)
      VALUES (1, 'placeholder')
    ON CONFLICT (id) DO NOTHING;
    `
	if _, err := pool.Exec(context.Background(), sql); err != nil {
		log.Fatalf("Failed to seed spotify_auth: %v\n", err)
	}
}

// func dropDomainTables(pool *pgxpool.Pool) {
// 	// only drop the tables we use to track plays
// 	sql := `
//     DROP TABLE IF EXISTS
//       plays,
//       recently_played,
//       tracks_on_repeat
//     CASCADE;
//     `
// 	if _, err := pool.Exec(context.Background(), sql); err != nil {
// 		log.Fatalf("Failed to drop domain tables: %v\n", err)
// 	}
// }

func createDomainTables(pool *pgxpool.Pool) {
	ctx := context.Background()

	// 1) tracks_on_repeat holds only songs replayed more than once
	sql1 := `
    CREATE TABLE IF NOT EXISTS tracks_on_repeat (
      id               SERIAL PRIMARY KEY,
      spotify_song_id  TEXT   NOT NULL UNIQUE,
      track_name       TEXT   NOT NULL,
      artist_name      TEXT   NOT NULL,
      album_name       TEXT   NOT NULL,
      genre            TEXT,
      preview_url      TEXT,
      album_cover_url  TEXT,
      play_count       INT    DEFAULT 0,
      first_played     TIMESTAMP,
      last_played      TIMESTAMP,
      time_of_day      TEXT,
      mood             TEXT,
      activity         TEXT
    );
    `
	if _, err := pool.Exec(ctx, sql1); err != nil {
		log.Fatalf("Failed to create tracks_on_repeat: %v\n", err)
	}

	// ALTER TABLE to ensure fields exist even if table was created earlier
	if _, err := pool.Exec(ctx, `
ALTER TABLE tracks_on_repeat
ADD COLUMN IF NOT EXISTS album_cover_url TEXT;
`); err != nil {
		log.Fatalf("Failed to add album_cover_url: %v\n", err)
	}

	if _, err := pool.Exec(ctx, `
ALTER TABLE tracks_on_repeat
ADD COLUMN IF NOT EXISTS genre TEXT;
`); err != nil {
		log.Fatalf("Failed to add genre: %v\n", err)
	}

	if _, err := pool.Exec(ctx, `
	ALTER TABLE recently_played
	ADD COLUMN IF NOT EXISTS album_cover_url TEXT;
	`); err != nil {
		log.Fatalf("Failed to add album_cover_url: %v\n", err)
	}

	if _, err := pool.Exec(ctx, `
	ALTER TABLE recently_played
	ADD COLUMN IF NOT EXISTS genre TEXT;
	`); err != nil {
		log.Fatalf("Failed to add genre: %v\n", err)
	}

	// if _, err := pool.Exec(ctx, `
	// 	ALTER TABLE recently_played
	// 	DROP COLUMN play_count
	// `); err != nil {
	// 	log.Fatalf("Failed to add genre: %v\n", err)
	// }

	// 2) recently_played logs every single play (cron job)
	sql2 := `
    CREATE TABLE IF NOT EXISTS recently_played (
      id               SERIAL PRIMARY KEY,
      spotify_song_id  TEXT   NOT NULL,
      track_name       TEXT   NOT NULL,
      artist_name      TEXT   NOT NULL,
      album_name       TEXT   NOT NULL,
      played_at        TIMESTAMP NOT NULL,
      source           TEXT      NOT NULL,       -- 'cron' or 'web'
      UNIQUE (spotify_song_id, played_at)
    );
    `
	if _, err := pool.Exec(ctx, sql2); err != nil {
		log.Fatalf("Failed to create recently_played: %v\n", err)
	}

	// 3) optional: plays table to reference every play by foreign-key
	sql3 := `
    CREATE TABLE IF NOT EXISTS plays (
      play_id          SERIAL PRIMARY KEY,
      spotify_song_id  TEXT      NOT NULL REFERENCES tracks_on_repeat (spotify_song_id),
      played_at        TIMESTAMP NOT NULL,
      source           TEXT      NOT NULL
    );
    `
	if _, err := pool.Exec(ctx, sql3); err != nil {
		log.Fatalf("Failed to create plays: %v\n", err)
	}

	// create recently liked songs
	sql4 := `
	 CREATE TABLE IF NOT EXISTS  recently_liked (
		id SERIAL PRIMARY KEY,
		spotify_song_id TEXT NOT NULL,
		added_at TIMESTAMP NOT NULL,
		track_name TEXT NOT NULL,
		track_popularity INTEGER,
		
		album_name TEXT,
		album_type TEXT,
		album_total_tracks INTEGER,
		album_release_date TEXT,
		album_release_date_precision TEXT,

		album_cover_url TEXT,
		album_cover_height INTEGER,
		album_cover_width INTEGER,

		artist_id TEXT,
		artist_name TEXT,
		artist_href TEXT,
		artist_uri TEXT
);`
	if _, err := pool.Exec(ctx, sql4); err != nil {
		log.Fatalf("Failed to create RECENTLY_LIKED", err)
	}
}

func GetLatestPlayedAt() (time.Time, error) {
	var latestTime time.Time

	query := `
		SELECT COALESCE(MAX(played_at), '1970-01-01'::timestamp) 
		FROM recently_played
	`

	err := Pool.QueryRow(context.Background(), query).Scan(&latestTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get latest played_at: %v", err)
	}

	return latestTime, nil
}

// GetLatestAddedAt returns the most recent added_at timestamp from recently_liked
func GetLatestAddedAt() (time.Time, error) {
	var latest time.Time

	query := `
		SELECT COALESCE(MAX(added_at), '1970-01-01'::timestamp)
		FROM recently_liked
	`

	err := Pool.QueryRow(context.Background(), query).Scan(&latest)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get latest added_at: %v", err)
	}

	return latest, nil
}

// GetTrackCountSince returns how many tracks we have since a given date
func GetTrackCountSince(since time.Time) (int, error) {
	var count int

	query := `SELECT COUNT(*) FROM recently_played WHERE played_at >= $1`

	err := Pool.QueryRow(context.Background(), query, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count tracks since %v: %v", since, err)
	}

	return count, nil
}

// GetTrackCountByDateRange returns counts grouped by date for analytics
func GetTrackCountByDateRange() ([]struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}, error) {
	query := `
		SELECT 
			DATE(played_at) as date,
			COUNT(*) as count
		FROM recently_played 
		WHERE played_at >= NOW() - INTERVAL '30 days'
		GROUP BY DATE(played_at)
		ORDER BY date DESC
	`

	rows, err := Pool.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to get date range counts: %v", err)
	}
	defer rows.Close()

	var results []struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}

	for rows.Next() {
		var result struct {
			Date  string `json:"date"`
			Count int    `json:"count"`
		}
		if err := rows.Scan(&result.Date, &result.Count); err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// Check if we have any data in recently_played table
func HasHistoricalData() (bool, error) {
	var count int

	query := `SELECT COUNT(*) FROM recently_played`

	err := Pool.QueryRow(context.Background(), query).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check historical data: %v", err)
	}

	return count > 0, nil
}
