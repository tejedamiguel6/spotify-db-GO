package models

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"example.com/spotifydb/internal/repository"
	"example.com/spotifydb/internal/services"
)

// RecentlyLikedTracks represents a track from the recently_liked table
type RecentlyLikedTracks struct {
	ID                        int       `json:"id"`
	SpotifyID                 string    `json:"spotify_song_id"`
	TrackName                 string    `json:"track_name"`
	TrackPopularity           string    `json:"track_popularity"`
	AlbumName                 string    `json:"album_name"`
	AlbumType                 string    `json:"album_type"`
	AlbumCoverURL             string    `json:"album_cover_url"`
	AlbumReleaseDate          string    `json:"album_release_date"`
	AlbumReleaseDatePrecision string    `json:"album_release_date_precision"`
	ArtistName                string    `json:"artist_name"`
	ArtistID                  string    `json:"artist_id"`
	ArtistHref                string    `json:"artist_href"`
	ArtistURI                 string    `json:"artist_uri"`
	AlbumTotalTracks          int       `json:"album_total_tracks"`
	AlbumCoverWidth           int       `json:"album_cover_width"`
	AlbumCoverHeight          int       `json:"album_cover_height"`
	Genre                     string    `json:"genre"`
	AddedAt                   time.Time `json:"added_at"`
}

// Cron writes one row per item; no touch on tracks_on_repeat
func InsertRecentlyPlayed(
	spotifyID, name, artist, album string, albumCoverURL string, genre string,
	playedAt time.Time,
) error {

	_, err := repository.Pool.Exec(context.Background(), `
		INSERT INTO recently_played
		      (spotify_song_id, track_name, artist_name, album_name, album_cover_url, genre,
		       played_at, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT DO NOTHING`,
		spotifyID, name, artist, album, albumCoverURL, genre, playedAt, "cron")
	return err
}

// inserts into DATABSE Recently LIKED table
func InsertRecentlyLiked(
	spotifyID, trackName, trackPopularity, albumName,
	albumType, albumCoverURL, albumReleaseDate, albumReleaseDatePrecision,
	artistName, artistID, href, artistURI string,
	albumTotalTracks, width, height int,
	addedAt time.Time,
) error {

	query := `
		INSERT INTO recently_liked (
			spotify_song_id,
			track_name,
			track_popularity,
			album_name,
			album_type,
			album_cover_url,
			album_release_date,
			album_release_date_precision,
			artist_name,
			artist_id,
			artist_href,
			artist_uri,
			album_total_tracks,
			album_cover_width,
			album_cover_height,
			added_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, 
			$8, $9, $10, $11, $12, $13, $14, $15, $16
		);
	`

	_, err := repository.Pool.Exec(
		context.Background(), query,
		spotifyID,
		trackName,
		trackPopularity,
		albumName,
		albumType,
		albumCoverURL,
		albumReleaseDate,
		albumReleaseDatePrecision,
		artistName,
		artistID,
		href,
		artistURI,
		albumTotalTracks,
		width,
		height,
		addedAt,
	)

	if err != nil {
		fmt.Printf("InsertRecentlyLiked error: %v\n", err)
	}

	return err

}

// backfilling
func BackfillMissingTrackData(accessToken string) error {
	rows, err := repository.Pool.Query(context.Background(), `
	SELECT DISTINCT spotify_song_id
	FROM recently_played
	WHERE album_cover_url IS NULL OR genre IS NULL
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var trackID string
		if err := rows.Scan(&trackID); err != nil { //will need a syntax explanation
			continue
		}

		track, err := services.GetTrack(trackID, accessToken)

		if err != nil {
			log.Printf("error getting track id")
			continue
		}

		artistID := track.Artists[0].ID
		artist, err := services.GetArtistById(artistID, accessToken)
		if err != nil {
			log.Printf("Error fetching artist %s: %v", artistID, err)
			continue
		}

		coverURL := ""
		if len(track.Album.Images) > 0 {
			coverURL = track.Album.Images[0].URL
		}
		genre := strings.Join(artist.Genres, ", ")

		_, err = repository.Pool.Exec(context.Background(), `
			UPDATE recently_played
			SET album_cover_url = $1, genre = $2
			WHERE spotify_song_id = $3
		`, coverURL, genre, trackID)
		if err != nil {
			log.Printf("Failed to update %s: %v", trackID, err)
		}
	}

	return nil

}

// CollectRecentlyLiked gets all recently liked tracks from the database
func CollectRecentlyLiked() []RecentlyLikedTracks {
	fmt.Println("🔍 CollectRecentlyLiked called")

	ctx := context.Background()

	// First, let's check if the table has any data using a fresh context
	var count int
	countQuery := "SELECT COUNT(*) FROM recently_liked"
	err := repository.Pool.QueryRow(ctx, countQuery).Scan(&count)
	if err != nil {
		fmt.Printf("❌ Error counting rows: %v\n", err)
		return []RecentlyLikedTracks{}
	}
	fmt.Printf("📊 Found %d rows in recently_liked table\n", count)

	if count == 0 {
		fmt.Println("📭 No tracks found in recently_liked table")
		return []RecentlyLikedTracks{}
	}

	// Use a simple query without complex formatting to avoid prepared statement conflicts
	query := `
		SELECT id, spotify_song_id, track_name, track_popularity,
			album_name,
			album_type, album_cover_url,
			album_release_date, album_release_date_precision,
			artist_name, artist_id, artist_href, artist_uri,
			album_total_tracks, album_cover_width,
			album_cover_height, genre, added_at
		FROM recently_liked
		ORDER BY added_at DESC
	`

	rows, err := repository.Pool.Query(ctx, query)
	if err != nil {
		fmt.Printf("❌ Error executing CollectRecentlyLiked query: %v\n", err)
		return []RecentlyLikedTracks{}
	}
	defer rows.Close()

	var tracks []RecentlyLikedTracks
	rowCount := 0
	for rows.Next() {
		var track RecentlyLikedTracks
		err := rows.Scan(
			&track.ID, &track.SpotifyID, &track.TrackName, &track.TrackPopularity,
			&track.AlbumName, &track.AlbumType, &track.AlbumCoverURL,
			&track.AlbumReleaseDate, &track.AlbumReleaseDatePrecision,
			&track.ArtistName, &track.ArtistID, &track.ArtistHref, &track.ArtistURI,
			&track.AlbumTotalTracks, &track.AlbumCoverWidth, &track.AlbumCoverHeight,
			&track.Genre, &track.AddedAt,
		)
		if err != nil {
			fmt.Printf("❌ Error scanning row %d: %v\n", rowCount+1, err)
			continue
		}
		tracks = append(tracks, track)
		rowCount++

		// Log first track for debugging
		if rowCount == 1 {
			fmt.Printf("📀 First track: %s by %s (Genre: %s)\n", track.TrackName, track.ArtistName, track.Genre)
		}
	}

	if err = rows.Err(); err != nil {
		fmt.Printf("❌ Error iterating rows: %v\n", err)
	}

	fmt.Printf("✅ Returning %d tracks from CollectRecentlyLiked\n", len(tracks))
	return tracks
}
