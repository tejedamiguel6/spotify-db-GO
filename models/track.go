package models

import (
	"context"
	"time"

	"example.com/spotifydb/db"
)

// Cron writes one row per item; no touch on tracks_on_repeat
func InsertRecentlyPlayed(
	spotifyID, name, artist, album string,
	playedAt time.Time,
) error {

	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO recently_played
		      (spotify_song_id, track_name, artist_name, album_name,
		       played_at, source)
		VALUES ($1,$2,$3,$4,$5,'cron')
		ON CONFLICT DO NOTHING`,
		spotifyID, name, artist, album, playedAt)
	return err
}
