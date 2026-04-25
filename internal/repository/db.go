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

	// Load .env for DATABASE_URL (optional - may not exist in production)
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found (this is normal in production environments)")
	}

	dsn := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	Pool = pool

	// Quick sanity check
	var currentDB string
	var currentUser string
	err = pool.QueryRow(context.Background(), "SELECT current_database(), current_user").Scan(&currentDB, &currentUser)
	if err != nil {
		log.Fatalf("Failed to check DB and user: %v", err)
	}
	fmt.Printf("🧠 Connected to DB: %s as user: %s\n", currentDB, currentUser)

	var greeting string
	if err := pool.QueryRow(context.Background(), "SELECT 'Connected to Neon DB!'").Scan(&greeting); err != nil {
		log.Fatalf("QueryRow failed: %v\n", err)

	}

	fmt.Println("✅", greeting)

	// Ensure required tables exist
	if err := ensureTablesExist(); err != nil {
		log.Fatalf("Failed to create required tables: %v", err)
	}
}

// ensureTablesExist creates all required tables if they don't exist
func ensureTablesExist() error {
	ctx := context.Background()

	// Create recently_liked table (the one that's missing after crash)
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

	if _, err := Pool.Exec(ctx, recentlyLikedTable); err != nil {
		return fmt.Errorf("failed to create recently_liked table: %v", err)
	}

	// Create spotify_auth table if it doesn't exist
	authTable := `
	CREATE TABLE IF NOT EXISTS spotify_auth (
		id INT PRIMARY KEY DEFAULT 1,
		refresh_token TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);`

	if _, err := Pool.Exec(ctx, authTable); err != nil {
		return fmt.Errorf("failed to create spotify_auth table: %v", err)
	}

	// Create recently_played table if it doesn't exist
	recentlyPlayedTable := `
	CREATE TABLE IF NOT EXISTS recently_played (
		id SERIAL PRIMARY KEY,
		spotify_song_id VARCHAR(255) NOT NULL,
		track_name TEXT NOT NULL,
		artist_name TEXT,
		album_name TEXT,
		album_cover_url TEXT,
		genre TEXT,
		duration_ms INTEGER,
		played_at TIMESTAMP NOT NULL,
		source VARCHAR(50) DEFAULT 'cron',
		created_at TIMESTAMP DEFAULT NOW(),
		UNIQUE(spotify_song_id, played_at)
	);`

	if _, err := Pool.Exec(ctx, recentlyPlayedTable); err != nil {
		return fmt.Errorf("failed to create recently_played table: %v", err)
	}

	if _, err := Pool.Exec(ctx, `ALTER TABLE recently_played ADD COLUMN IF NOT EXISTS duration_ms INTEGER`); err != nil {
		return fmt.Errorf("failed to ensure duration_ms column: %v", err)
	}

	// Create useful indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_recently_liked_added_at ON recently_liked(added_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_recently_liked_genre ON recently_liked(genre);",
		"CREATE INDEX IF NOT EXISTS idx_recently_played_played_at ON recently_played(played_at DESC);",
	}

	for _, indexSQL := range indexes {
		if _, err := Pool.Exec(ctx, indexSQL); err != nil {
			// Don't fail on index creation errors, just log them
			fmt.Printf("⚠️  Warning: Failed to create index: %v\n", err)
		}
	}

	fmt.Println("🏗️  Database schema verified/created")
	return nil
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

// DailyListeningStat is one day's row in /listening-stats daily_breakdown_last_30_days.
type DailyListeningStat struct {
	Date    string `json:"date"`
	TotalMs int64  `json:"total_ms"`
	Count   int    `json:"count"`
}

// GetDailyListeningBreakdown returns per-day aggregates for the last 30 days
// (only days with at least one play are included).
func GetDailyListeningBreakdown() ([]DailyListeningStat, error) {
	query := `
		SELECT
			TO_CHAR(DATE(played_at), 'YYYY-MM-DD') AS date,
			COALESCE(SUM(duration_ms), 0)          AS total_ms,
			COUNT(*)                               AS count
		FROM recently_played
		WHERE played_at >= NOW() - INTERVAL '30 days'
		GROUP BY DATE(played_at)
		ORDER BY DATE(played_at) DESC
	`
	rows, err := Pool.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed daily listening breakdown: %v", err)
	}
	defer rows.Close()

	results := make([]DailyListeningStat, 0, 30)
	for rows.Next() {
		var s DailyListeningStat
		if err := rows.Scan(&s.Date, &s.TotalMs, &s.Count); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, nil
}

// GetListeningTimeByPeriod returns total_ms summed for each named period,
// matching the periods used by /collection-stats.
func GetListeningTimeByPeriod() (map[string]int64, error) {
	query := `
		SELECT
			COALESCE(SUM(duration_ms) FILTER (WHERE played_at >= NOW() - INTERVAL '24 hours'), 0) AS last_24_hours,
			COALESCE(SUM(duration_ms) FILTER (WHERE played_at >= NOW() - INTERVAL '7 days'),     0) AS last_week,
			COALESCE(SUM(duration_ms) FILTER (WHERE played_at >= NOW() - INTERVAL '1 month'),    0) AS last_month,
			COALESCE(SUM(duration_ms) FILTER (WHERE played_at >= NOW() - INTERVAL '3 months'),   0) AS last_3_months,
			COALESCE(SUM(duration_ms),                                                            0) AS all_time
		FROM recently_played
	`
	var l24, lw, lm, l3m, all int64
	if err := Pool.QueryRow(context.Background(), query).Scan(&l24, &lw, &lm, &l3m, &all); err != nil {
		return nil, fmt.Errorf("failed listening time by period: %v", err)
	}
	return map[string]int64{
		"last_24_hours": l24,
		"last_week":     lw,
		"last_month":    lm,
		"last_3_months": l3m,
		"all_time":      all,
	}, nil
}

// TopTrack is one row of the /top-tracks aggregation
type TopTrack struct {
	SpotifySongID string `json:"song_id"`
	TrackName     string `json:"track_name"`
	ArtistName    string `json:"artist_name"`
	AlbumName     string `json:"album_name"`
	AlbumCoverURL string `json:"album_cover_url"`
	PlayCount     int    `json:"play_count"`
	TotalMs       int64  `json:"total_ms"`
}

// GetTopTracks aggregates recently_played by song and returns the highest-played
// tracks within the given window. Pass a zero time.Time to leave that bound open.
// Tie-breaker: total_ms DESC, then spotify_song_id ASC for determinism.
func GetTopTracks(from, to time.Time, limit int) ([]TopTrack, error) {
	query := `
		SELECT
			spotify_song_id,
			MAX(track_name)                     AS track_name,
			COALESCE(MAX(artist_name), '')      AS artist_name,
			COALESCE(MAX(album_name), '')       AS album_name,
			COALESCE(MAX(album_cover_url), '')  AS album_cover_url,
			COUNT(*)                            AS play_count,
			COALESCE(SUM(duration_ms), 0)       AS total_ms
		FROM recently_played
		WHERE ($1::timestamp IS NULL OR played_at >= $1)
		  AND ($2::timestamp IS NULL OR played_at <  $2)
		GROUP BY spotify_song_id
		ORDER BY play_count DESC, total_ms DESC, spotify_song_id ASC
		LIMIT $3
	`

	var fromArg, toArg any
	if !from.IsZero() {
		fromArg = from
	}
	if !to.IsZero() {
		toArg = to
	}

	rows, err := Pool.Query(context.Background(), query, fromArg, toArg, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top tracks: %v", err)
	}
	defer rows.Close()

	results := make([]TopTrack, 0, limit)
	for rows.Next() {
		var t TopTrack
		if err := rows.Scan(
			&t.SpotifySongID,
			&t.TrackName,
			&t.ArtistName,
			&t.AlbumName,
			&t.AlbumCoverURL,
			&t.PlayCount,
			&t.TotalMs,
		); err != nil {
			return nil, fmt.Errorf("scan top tracks row: %v", err)
		}
		results = append(results, t)
	}
	return results, nil
}

// GetArtistsByGenre returns unique artists from specified table that match the given genre
func GetArtistsByGenre(tableName, genre string) ([]map[string]any, error) {
	query := fmt.Sprintf(`
		SELECT artist_name, artist_id, 
		       COUNT(*) as track_count,
		       STRING_AGG(DISTINCT genre, ', ') as genres,
		       (SELECT album_cover_url FROM %s r2 
		        WHERE r2.artist_id = r1.artist_id 
		        AND r2.album_cover_url IS NOT NULL 
		        ORDER BY r2.added_at DESC LIMIT 1) as artist_image_url
		FROM %s r1
		WHERE genre ILIKE $1 
		GROUP BY artist_name, artist_id
		ORDER BY track_count DESC, artist_name
	`, tableName, tableName)

	rows, err := Pool.Query(context.Background(), query, "%"+genre+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to get artists by genre: %v", err)
	}
	defer rows.Close()

	var artists []map[string]any
	for rows.Next() {
		var artistName, artistID, genres string
		var artistImageURL *string
		var trackCount int

		err := rows.Scan(&artistName, &artistID, &trackCount, &genres, &artistImageURL)
		if err != nil {
			continue
		}

		imageURL := ""
		if artistImageURL != nil {
			imageURL = *artistImageURL
		}

		artist := map[string]any{
			"artist_name":      artistName,
			"artist_id":        artistID,
			"track_count":      trackCount,
			"genres":           genres,
			"artist_image_url": imageURL,
		}
		artists = append(artists, artist)
	}

	return artists, nil
}
