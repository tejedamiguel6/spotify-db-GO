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
	PlayedAt      time.Time `json:"played_at"`
	FirstPlayed   time.Time `json:"first_played"`
	LastPlayed    time.Time `json:"last_played"`
	MonthYear     string    `json:"month_year"`
	TimeOfDay     string    `json:"time_of_day"`
	Mood          string    `json:"mood"`
	Activity      string    `json:"activity"`
}

type RecentlyPlayedTrack struct {
	ID            int    `json:"id"`
	SpotifySongID string `json:"spotify_song_id" binding:"required"`
	TrackName     string `json:"track_name" binding:"required"`
	ArtistName    string `json:"artist_name" binding:"required"`
	AlbumName     string `json:"album_name" binding:"required"`

	PlayedAt time.Time `json:"played_at"`
	Source   string    `json:"source"`
}

func (t Track) SaveToDatabase(pool *pgxpool.Pool) error {

	var existingTrackID string
	checkQuery := `SELECT id FROM tracks_on_repeat WHERE spotify_song_id = $1`

	err := pool.QueryRow((context.Background()), checkQuery, t.SpotifySongID).Scan(&existingTrackID)

	if err == nil {
		fmt.Println("Track already exists in database with ID:", existingTrackID)
		return nil
	}

	// If the error is not "no rows found", return the error
	if err.Error() != "no rows in result set" {
		fmt.Printf("Error checking for existing track: %v\n", err)
		return err
	}

	// now insert exactly 13 columns with 13 placeholders
	query := `
	 INSERT INTO tracks_on_repeat (
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
		 time_of_day,
		 mood,
		 activity
	 ) VALUES (
		 $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
	 );
	 `

	_, err = pool.Exec(
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

func GetAllTracksonRepeat(pool *pgxpool.Pool) []Track {
	fmt.Println("this is the GETALLSAVE TRACKS")

	query := `
  SELECT *
  FROM   tracks_on_repeat
  ORDER  BY COALESCE(last_played, first_played, '1970-01-01') DESC,
           id ASC;
`

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

func GetSingleTrack(pool *pgxpool.Pool, spotifyID string) (*Track, error) {
	// might need to edit this serialization since im using postgres
	query := "SELECT * FROM tracks_on_repeat WHERE spotify_song_id  = $1"

	row := pool.QueryRow(context.Background(), query, spotifyID)
	var t Track
	err := row.Scan(
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
		return nil, err
	}
	return &t, nil

}

// update track in DB
func (t Track) UpdateTrackDB(pool *pgxpool.Pool) error {

	fmt.Print("ABOUT TO UPDATE!")

	query := `
	UPDATE tracks_on_repeat
	SET play_count = $1,
		last_played = $2
	WHERE spotify_song_id = $3;
`

	_, err := pool.Exec(
		context.Background(),
		query,
		t.PlayCount,
		time.Now(), // set last_played to now
		t.SpotifySongID,
	)
	if err != nil {
		fmt.Printf("Failed to update track: %v\n", err)
		return err
	}

	fmt.Println("Track updated successfully.")
	return nil

}

// function that gets most recent plays from db
func GetAllRecentPlayedHistory(pool *pgxpool.Pool) ([]RecentlyPlayedTrack, error) {
	query := `
		SELECT 
			id,
			spotify_song_id,
			track_name,
			artist_name,
			album_name,
			played_at,
			source
		FROM recently_played
		ORDER BY played_at DESC
	`

	fmt.Println("query--->", query)

	rows, err := pool.Query(context.Background(), query)
	if err != nil {
		fmt.Println("Failed to query recently_played:", err)
		return nil, err
	}
	defer rows.Close()

	var results []RecentlyPlayedTrack

	for rows.Next() {
		var rpt RecentlyPlayedTrack

		err := rows.Scan(
			&rpt.ID,
			&rpt.SpotifySongID,
			&rpt.TrackName,
			&rpt.ArtistName,
			&rpt.AlbumName,
			&rpt.PlayedAt,
			&rpt.Source,
		)
		if err != nil {
			fmt.Println("Error scanning row:", err)
			continue
		}
		results = append(results, rpt)
	}

	return results, nil

}
