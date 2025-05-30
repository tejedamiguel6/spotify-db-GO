package models

import (
	"context"
	"fmt"
	"time"

	// must be v5
	"github.com/jackc/pgx/v5/pgxpool"
)

type Track struct {
	ID            int       `json:"id"`
	SpotifySongID string    `json:"spotify_song_id" binding:"required"`
	TrackName     string    `json:"track_name" binding:"required"`
	ArtistName    string    `json:"artist_name" binding:"required"`
	AlbumName     string    `json:"album_name" binding:"required"`
	Genre         string    `json:"genre"`
	PreviewURL    string    `json:"preview_url"`
	AlbumCoverURL string    `json:"album_cover_url"` // Optional
	PlayCount     int       `json:"play_count"`      // Default to 0 if not provided
	FirstPlayed   time.Time `json:"first_played"`
	LastPlayed    time.Time `json:"last_played"`
	MonthYear     string    `json:"month_year"`
	TimeOfDay     string    `json:"time_of_day"`
	Mood          string    `json:"mood"`
	Activity      string    `json:"activity"`
}

var tracks = []Track{}

// method
func (t Track) SaveToMemory() {
	// add to a database later
	fmt.Println("SAVED TO MEMORY")
	tracks = append(tracks, t)

}

func (t Track) SaveToDatabase(pool *pgxpool.Pool) error {
	query := `
    INSERT INTO tracks (
        spotify_song_id,
        track_name,
        artist_name,
        album_name,
        genre,
        preview_url,
        album_cover_url,
        play_count,
        first_played,
        last_played,
        month_year,
        time_of_day,
        mood,
        activity
    ) VALUES (
        $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
    );
    `

	_, err := pool.Exec(
		context.Background(),
		query,
		t.SpotifySongID,
		t.TrackName,
		t.ArtistName,
		t.AlbumName,
		t.Genre,
		t.PreviewURL,
		t.AlbumCoverURL,
		t.PlayCount,
		t.FirstPlayed,
		t.LastPlayed,
		t.MonthYear,
		t.TimeOfDay,
		t.Mood,
		t.Activity,
	)

	if err != nil {
		fmt.Printf("Failed to save track to database: %v\n", err)
		return err
	}

	fmt.Println("Track saved to database successfully.")
	return nil
}

func GetAllTracks(pool *pgxpool.Pool) []Track {
	fmt.Println("this is the GETALLSAVE TRACKS")

	query := "SELECT * FROM tracks"

	rows, err := pool.Query(context.Background(), query)
	if err != nil {
		fmt.Println("Failed to query tracks:", err)
		return nil
	}
	defer rows.Close()

	var results []Track

	for rows.Next() {
		var t Track

		err := rows.Scan(
			&t.ID,
			&t.SpotifySongID,
			&t.TrackName,
			&t.ArtistName,
			&t.AlbumName,
			&t.Genre,
			&t.PreviewURL,
			&t.AlbumCoverURL,
			&t.PlayCount,
			&t.FirstPlayed,
			&t.LastPlayed,
			&t.MonthYear,
			&t.TimeOfDay,
			&t.Mood,
			&t.Activity,
		)
		if err != nil {
			fmt.Println("Error scanning row:", err)
			continue
		}

		results = append(results, t)
	}

	return results
}
