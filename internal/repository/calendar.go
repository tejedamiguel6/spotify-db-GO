package repository

import (
	"context"
	"fmt"
	"time"
)

// CalendarDay holds one day's listening totals plus the day's most-played track
type CalendarDay struct {
	Date          string `json:"date"`
	PlayCount     int    `json:"play_count"`
	TotalMs       int64  `json:"total_ms"`
	TopSongID     string `json:"top_song_id"`
	TopTrackName  string `json:"top_track_name"`
	TopArtistName string `json:"top_artist_name"`
	AlbumCoverURL string `json:"album_cover_url"`
}

// GetDailyCalendar returns per-day play totals and each day's most-played
// track (ties broken by listening time) for the inclusive date range.
// played_at is stored as UTC wall-clock time; days are grouped by the
// calendar date in tz (an IANA name such as "America/Denver").
func GetDailyCalendar(from, to time.Time, tz string) ([]CalendarDay, error) {
	query := `
	WITH per_track AS (
		SELECT DATE((played_at AT TIME ZONE 'UTC') AT TIME ZONE $3) AS d,
		       spotify_song_id,
		       MAX(track_name) AS track_name,
		       MAX(artist_name) AS artist_name,
		       MAX(album_cover_url) AS album_cover_url,
		       COUNT(*) AS plays,
		       COALESCE(SUM(duration_ms), 0) AS total_ms
		FROM recently_played
		-- Local midnight of 'from' up to (not including) local midnight after 'to',
		-- converted to UTC so the played_at index can still be used.
		WHERE played_at >= ($1::date::timestamp AT TIME ZONE $3) AT TIME ZONE 'UTC'
		  AND played_at <  (($2::date + 1)::timestamp AT TIME ZONE $3) AT TIME ZONE 'UTC'
		GROUP BY 1, spotify_song_id
	),
	ranked AS (
		SELECT per_track.*,
		       ROW_NUMBER() OVER (
		           PARTITION BY d
		           ORDER BY plays DESC, total_ms DESC, spotify_song_id
		       ) AS rn,
		       SUM(plays) OVER (PARTITION BY d) AS day_plays,
		       SUM(total_ms) OVER (PARTITION BY d) AS day_ms
		FROM per_track
	)
	SELECT TO_CHAR(d, 'YYYY-MM-DD') AS date,
	       day_plays,
	       day_ms,
	       spotify_song_id,
	       track_name,
	       COALESCE(artist_name, ''),
	       COALESCE(album_cover_url, '')
	FROM ranked
	WHERE rn = 1
	ORDER BY d`

	rows, err := Pool.Query(context.Background(), query,
		from.Format("2006-01-02"), to.Format("2006-01-02"), tz)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily calendar: %v", err)
	}
	defer rows.Close()

	var days []CalendarDay
	for rows.Next() {
		var d CalendarDay
		if err := rows.Scan(&d.Date, &d.PlayCount, &d.TotalMs,
			&d.TopSongID, &d.TopTrackName, &d.TopArtistName, &d.AlbumCoverURL); err != nil {
			return nil, err
		}
		days = append(days, d)
	}
	return days, nil
}
