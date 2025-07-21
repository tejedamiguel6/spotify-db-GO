package repository

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

// InitDB initializes the connection pool to Neon
func InitDB() {
	fmt.Println("🔌  Connecting to  database…")

	// Load .env for DATABASE_URL
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
	if err := pool.QueryRow(context.Background(), "SELECT 'Connected to Neon DB!'").Scan(&greeting); err != nil {
		log.Fatalf("QueryRow failed: %v\n", err)

	}

	fmt.Println("✅", greeting)
}

// GetLatestPlayedAt returns the most recent played_at timestamp from recently_played
func GetLatestPlayedAt() (time.Time, error) {
	var latestTime time.Time
	query := `SELECT COALESCE(MAX(played_at), '1970-01-01'::timestamp) FROM recently_played`
	err := Pool.QueryRow(context.Background(), query).Scan(&latestTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get latest played_at: %v", err)
	}
	return latestTime, nil
}

// GetLatestAddedAt returns the most recent added_at timestamp from recently_liked
func GetLatestAddedAt() (time.Time, error) {
	var latest time.Time
	query := `SELECT COALESCE(MAX(added_at), '1970-01-01'::timestamp) FROM recently_liked`
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

// HasHistoricalData checks if we have any data in recently_played
func HasHistoricalData() (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM recently_played`
	err := Pool.QueryRow(context.Background(), query).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check historical data: %v", err)
	}
	return count > 0, nil
}
