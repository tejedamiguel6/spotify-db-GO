package models

import "time"

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
	ID            int       `json:"id"`
	SpotifySongID string    `json:"spotify_song_id" binding:"required"`
	TrackName     string    `json:"track_name" binding:"required"`
	DurationMS    int       `json:"duration_ms"`
	ArtistName    string    `json:"artist_name" binding:"required"`
	AlbumName     string    `json:"album_name" binding:"required"`
	PlayedAt      time.Time `json:"played_at"`
	Source        string    `json:"source"`
	AlbumCoverUrl string    `json:"album_cover_url"`
	Genre         string    `json:"genre"`
}
