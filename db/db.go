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
	dropDomainTables(pool)

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

func dropDomainTables(pool *pgxpool.Pool) {
	// only drop the tables we use to track plays
	sql := `
    DROP TABLE IF EXISTS
      plays,
      recently_played,
      tracks_on_repeat
    CASCADE;
    `
	if _, err := pool.Exec(context.Background(), sql); err != nil {
		log.Fatalf("Failed to drop domain tables: %v\n", err)
	}
}

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
