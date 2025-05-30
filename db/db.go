package db

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var Pool *pgxpool.Pool

func InitDB() {
	fmt.Println("database running")

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env")
	}

	// This assigns to the GLOBAL, not a local variable!
	dsn := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), dsn)

	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	Pool = pool

	var greeting string

	err = pool.QueryRow(context.Background(), "select 'Hello, world!'").Scan(&greeting)
	if err != nil {
		log.Fatalf("QueryRow failed: %v\n", err)
	}

	fmt.Println(greeting)

	createTable(pool)
}

func createTable(pool *pgxpool.Pool) {

	// dropTable := `DROP TABLE IF EXISTS tracks`

	createMostPlayedTracksTable := `
	CREATE TABLE IF NOT EXISTS tracks (
	    id SERIAL PRIMARY KEY,
	    spotify_song_id TEXT NOT NULL,
	    track_name TEXT NOT NULL,
	    artist_name TEXT NOT NULL,
	    album_name TEXT NOT NULL,
	    genre TEXT,
	    preview_url TEXT,
	    album_cover_url TEXT,
	    play_count INT DEFAULT 0,
	    first_played TIMESTAMP,
	    last_played TIMESTAMP,
	    month_year TEXT,
	    time_of_day TEXT,
	    mood TEXT,
	    activity TEXT
	);
    `

	// Drop old table (dev only — remove later for production)
	// _, err := pool.Exec(context.Background(), dropTable)
	// if err != nil {
	// 	log.Fatalf("Failed to drop tracks table: %v\n", err)
	// } else {
	// 	fmt.Println("Dropped existing 'tracks' table.")
	// }

	// Create new table
	_, err := pool.Exec(context.Background(), createMostPlayedTracksTable)
	if err != nil {
		log.Fatalf("Failed to create tracks table: %v\n", err)
	} else {
		fmt.Println("Table 'tracks' is ready.")
	}
}
