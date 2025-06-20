package models

import (
	"context"
	"log"
	"strings"
	"time"

	"example.com/spotifydb/db"
	"example.com/spotifydb/spotify"
)

// Cron writes one row per item; no touch on tracks_on_repeat
func InsertRecentlyPlayed(
	spotifyID, name, artist, album string, albumCoverURL string, genre string,
	playedAt time.Time,
) error {

	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO recently_played
		      (spotify_song_id, track_name, artist_name, album_name, album_cover_url, genre,
		       played_at, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT DO NOTHING`,
		spotifyID, name, artist, album, albumCoverURL, genre, playedAt, "cron")
	return err
}

// backfilling
func BackfillMissingTrackData(accessToken string) error {
	rows, err := db.Pool.Query(context.Background(), `
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

		track, err := spotify.GetTrack(trackID, accessToken)

		if err != nil {
			log.Printf("error getting track id")
			continue
		}

		artistID := track.Artists[0].ID
		artist, err := spotify.GetArtistById(artistID, accessToken)
		if err != nil {
			log.Printf("Error fetching artist %s: %v", artistID, err)
			continue
		}

		coverURL := ""
		if len(track.Album.Images) > 0 {
			coverURL = track.Album.Images[0].URL
		}
		genre := strings.Join(artist.Genres, ", ")

		_, err = db.Pool.Exec(context.Background(), `
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
