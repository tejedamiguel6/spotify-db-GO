package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// EnsureEnrichmentsTable creates the track_enrichments cache table.
// Enrichments are AI-generated (genre, moods, scores, palette) and keyed
// case-insensitively by track + artist so each song is analyzed once.
func EnsureEnrichmentsTable() error {
	ctx := context.Background()

	table := `
	CREATE TABLE IF NOT EXISTS track_enrichments (
		id SERIAL PRIMARY KEY,
		track_name TEXT NOT NULL,
		artist_name TEXT NOT NULL,
		enrichment JSONB NOT NULL,
		model VARCHAR(100),
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);`
	if _, err := Pool.Exec(ctx, table); err != nil {
		return fmt.Errorf("failed to create track_enrichments table: %v", err)
	}

	index := `CREATE UNIQUE INDEX IF NOT EXISTS idx_track_enrichments_key
		ON track_enrichments (LOWER(track_name), LOWER(artist_name));`
	if _, err := Pool.Exec(ctx, index); err != nil {
		return fmt.Errorf("failed to create track_enrichments index: %v", err)
	}
	return nil
}

// GetEnrichment returns the cached enrichment JSON for a track, or nil if
// the track has not been enriched yet.
func GetEnrichment(track, artist string) ([]byte, error) {
	var enrichment []byte
	query := `SELECT enrichment FROM track_enrichments
		WHERE LOWER(track_name) = LOWER($1) AND LOWER(artist_name) = LOWER($2)`
	err := Pool.QueryRow(context.Background(), query, track, artist).Scan(&enrichment)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get enrichment: %v", err)
	}
	return enrichment, nil
}

// SaveEnrichment upserts the enrichment JSON for a track.
func SaveEnrichment(track, artist string, enrichment []byte, model string) error {
	query := `INSERT INTO track_enrichments (track_name, artist_name, enrichment, model)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (LOWER(track_name), LOWER(artist_name))
		DO UPDATE SET enrichment = EXCLUDED.enrichment, model = EXCLUDED.model, updated_at = NOW()`
	if _, err := Pool.Exec(context.Background(), query, track, artist, enrichment, model); err != nil {
		return fmt.Errorf("failed to save enrichment: %v", err)
	}
	return nil
}
