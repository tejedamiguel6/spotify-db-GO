package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"example.com/spotifydb/internal/models"
	"example.com/spotifydb/internal/repository"
	"example.com/spotifydb/internal/services"

	"github.com/gin-gonic/gin"
)

/* ---------- collection statistics ---------- */
func GetCollectionStats(c *gin.Context) {
	now := time.Now()

	// Get counts for different time periods
	counts := make(map[string]int)
	periods := map[string]time.Time{
		"last_24_hours": now.AddDate(0, 0, -1),
		"last_week":     now.AddDate(0, 0, -7),
		"last_month":    now.AddDate(0, -1, 0),
		"last_3_months": now.AddDate(0, -3, 0),
		"last_6_months": now.AddDate(0, -6, 0),
		"all_time":      time.Time{},
	}

	for period, since := range periods {
		count, err := repository.GetTrackCountSince(since)
		if err != nil {
			fmt.Printf("Error getting count for %s: %v\n", period, err)
			counts[period] = 0
		} else {
			counts[period] = count
		}
	}

	// Get latest track info
	latestTime, err := repository.GetLatestPlayedAt()
	var latestTrackInfo string
	if err != nil {
		latestTrackInfo = "No tracks collected yet"
	} else if latestTime.IsZero() {
		latestTrackInfo = "No tracks collected yet"
	} else {
		latestTrackInfo = latestTime.Format("2006-01-02 15:04:05 MST")
	}

	// Get daily breakdown for the last 30 days
	dailyStats, err := repository.GetTrackCountByDateRange()
	if err != nil {
		fmt.Printf("Error getting daily stats: %v\n", err)
	}

	// Calculate collection progress toward 6 months
	sixMonthsTarget := 6 * 30 * 24 * 2 // Rough estimate: 2 songs per hour for 6 months
	progressPercent := float64(counts["all_time"]) / float64(sixMonthsTarget) * 100
	if progressPercent > 100 {
		progressPercent = 100
	}

	c.JSON(200, gin.H{
		"collection_summary": gin.H{
			"total_tracks_collected":   counts["all_time"],
			"latest_track_time":        latestTrackInfo,
			"progress_toward_6_months": fmt.Sprintf("%.1f%%", progressPercent),
		},
		"track_counts_by_period":       counts,
		"daily_breakdown_last_30_days": dailyStats,
		"collection_tips": []string{
			"Keep the app running to continuously collect tracks",
			"The system checks every 1.5 minutes during active hours (6 AM - 11 PM)",
			"Spotify only stores ~50 recent tracks, so continuous collection is essential",
			"You'll have meaningful 6-month data after running for a few months",
		},
	})
}

/* ---------- enhanced background ticker ---------- */
func StartSpotifyCron() {
	// Check if we need to do initial historical fetch
	go func() {
		time.Sleep(5 * time.Second) // Wait for server to start up

		hasData, err := repository.HasHistoricalData()
		if err != nil {
			fmt.Printf("Error checking historical data: %v\n", err)
			return
		}

		if !hasData {
			fmt.Println("🎵 No historical data found. Starting continuous collection...")
			fmt.Println("📊 Spotify's recently played API only stores ~50 tracks at a time.")
			fmt.Println("⏰ Running every 2 minutes to ensure we don't miss any tracks.")
			fmt.Println("📈 Your listening history will build up over time!")
		} else {
			// Show some stats about existing data
			if thirtyDaysAgo := time.Now().AddDate(0, 0, -30); true {
				count, err := repository.GetTrackCountSince(thirtyDaysAgo)
				if err == nil {
					fmt.Printf("📊 You have %d tracks collected in the last 30 days\n", count)
				}
			}
		}
	}()

	// Adaptive frequency: run more often during likely listening hours
	go func() {
		for {
			now := time.Now()
			hour := now.Hour()

			var interval time.Duration

			// More frequent during typical listening hours (6 AM - 11 PM)
			if hour >= 6 && hour <= 23 {
				interval = 90 * time.Second // Every 1.5 minutes during active hours
			} else {
				interval = 5 * time.Minute // Every 5 minutes during sleep hours
			}

			time.Sleep(interval)
			CollectRecentTracks()
			CollectSavedTracks()
			GetCurrentlyPLaying()

		}
	}()
}

func CollectRecentTracks() {
	refreshTok, err := repository.GetRefreshToken()
	if err != nil || refreshTok == "" {
		// Only log this once per hour to avoid spam
		if time.Now().Minute() == 0 {
			fmt.Println("cron: no refresh token stored yet")
		}
		return
	}

	accessTok, newRefresh, err := services.RefreshAccessToken(refreshTok)
	if err != nil {
		fmt.Println("cron: refresh error:", err)
		return
	}
	if newRefresh != nil && *newRefresh != refreshTok {
		_ = repository.SaveOrUpdateRefreshToken(*newRefresh)
	}

	// Get the latest timestamp from our database to avoid duplicates
	latestTime, err := repository.GetLatestPlayedAt()
	if err != nil {
		fmt.Printf("cron: error getting latest timestamp: %v\n", err)
		latestTime = time.Time{} // Start from beginning if error
	}

	// Get recent tracks from Spotify
	items, err := services.GetRecentlyPlayed(accessTok, 50)
	if err != nil {
		fmt.Println("cron: recently-played error:", err)
		return
	}

	if len(items) == 0 {
		return // No tracks to process
	}

	success := 0
	skipped := 0
	var newestTrack, oldestTrack time.Time

	for i, it := range items {
		// Track the range of tracks we're processing
		if i == 0 {
			newestTrack = it.PlayedAt
		}
		if i == len(items)-1 {
			oldestTrack = it.PlayedAt
		}

		// Skip tracks we already have (based on timestamp)
		if !latestTime.IsZero() && it.PlayedAt.Before(latestTime.Add(time.Second)) {
			skipped++
			continue
		}

		artist := ""
		genre := ""
		albumCoverURL := ""

		if len(it.Track.Artists) > 0 {
			artistID := it.Track.Artists[0].ID
			artistObj, err := services.GetArtistById(accessTok, artistID)
			if err != nil {
				log.Printf("Failed to fetch artist %s: %v", artistID, err)
			} else if artistObj != nil {
				artist = artistObj.Name
				if len(artistObj.Genres) > 0 {
					genre = strings.Join(artistObj.Genres, ", ")
				}
			}
		}

		if len(it.Track.Album.Images) > 0 {
			albumCoverURL = it.Track.Album.Images[0].URL
		}

		// checks for existing track

		err = models.InsertRecentlyPlayed(
			it.Track.ID,
			it.Track.Name,
			artist,
			it.Track.Album.Name,
			albumCoverURL,
			genre,
			it.PlayedAt,
		)
		if err != nil {
			// Only log errors that aren't duplicate key violations
			if !strings.Contains(err.Error(), "duplicate") && !strings.Contains(err.Error(), "unique") {
				fmt.Printf("cron: insert error for %s: %v\n", it.Track.Name, err)
			}
		} else {
			success++
		}
	}

	// Only log when we actually find new tracks
	if success > 0 {
		fmt.Printf("🎵 collected %d new tracks (skipped %d) | range: %s to %s | %s\n",
			success, skipped,
			oldestTrack.Format("15:04:05"),
			newestTrack.Format("15:04:05"),
			time.Now().Format(time.Kitchen))
	}
}

/* ---------- route to accept refresh token from Next.js ---------- */
func SaveRefresh(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.RefreshToken == "" {
		c.JSON(400, gin.H{"msg": "no token"})
		return
	}
	if err := repository.SaveOrUpdateRefreshToken(body.RefreshToken); err != nil {
		c.JSON(500, gin.H{"msg": "db error"})
		return
	}
	c.JSON(200, gin.H{"msg": "saved"})
}

func GetMostPlayedTracks(context *gin.Context) {
	tracks := models.GetAllTracksonRepeat(repository.Pool)
	context.JSON(http.StatusOK, tracks)
}

// saves to database
func CreateTrack(context *gin.Context) {
	var track models.Track

	err := context.ShouldBindJSON(&track)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "could not parse data"})
		return
	}

	// this saves into database
	err = track.SaveToDatabase(repository.Pool)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "could not save to database"})
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "created track", "track": track})
}

// update tracks
func UpdateTrack(context *gin.Context) {
	spotifyID := context.Param("spotify_song_id")

	var updateData struct {
		PlayCount int `json:"play_count"`
	}

	if err := context.ShouldBindJSON(&updateData); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "invalid JSON"})
		return
	}
	fmt.Printf("PATCH request received for song ID: %s\n", spotifyID)
	fmt.Printf("Incoming update data: %+v\n", updateData)
	// get the existing track from db
	existingTrack, err := models.GetSingleTrack(repository.Pool, spotifyID)
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"message:": "track not found!"})
		return
	}

	// update the fields
	existingTrack.PlayCount = updateData.PlayCount

	// Save back to DB
	if err := existingTrack.UpdateTrackDB(repository.Pool); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "could not update track"})
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "track updated"})
}

func RecentlyPlayedTracks(context *gin.Context) {
	recentPlayedTracks, err := models.GetAllRecentPlayedHistory(repository.Pool)
	if err != nil {
		fmt.Println("ERROR HERE IN HANDLERS:", err)
		context.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch recently played tracks",
			"details": err.Error(),
		})
		return
	}
	context.JSON(http.StatusOK, gin.H{
		"tracks":  recentPlayedTracks,
		"count":   len(recentPlayedTracks),
		"message": "succescfully retrieved tracks",
	})

}

// this is the now listeing to endpoint call
func NowListeningToTrack(context *gin.Context) {
	refreshTok, err := repository.GetRefreshToken()
	if err != nil || refreshTok == "" {
		fmt.Println("NowListeningToTrack: no refresh token stored yet")

	}

	// Exchange for access token
	accessTok, newRefresh, err := services.RefreshAccessToken(refreshTok)

	if err != nil {
		fmt.Print("error retrieving currently RefreshAccessToken ")
	}

	if newRefresh != nil && *newRefresh != refreshTok {
		_ = repository.SaveOrUpdateRefreshToken(*newRefresh)
	}

	listeingTrack, err := services.GetCurrentlyListening(accessTok)
	if err != nil {
		fmt.Printf("error getting currently listening to: %v\n", err)
	}

	context.JSON(http.StatusOK, gin.H{
		"data":    listeingTrack,
		"message": "success",
	})

}

// function to get all saved tracks
func CollectSavedTracks() {
	// Get refresh token
	refreshTok, err := repository.GetRefreshToken()
	if err != nil || refreshTok == "" {
		fmt.Println("CollectSavedTracks: no refresh token stored yet")
		return
	}

	// Exchange for access token
	accessTok, newRefresh, err := services.RefreshAccessToken(refreshTok)
	if err != nil {
		fmt.Println("CollectSavedTracks: refresh error:", err)
		return
	}
	if newRefresh != nil && *newRefresh != refreshTok {
		_ = repository.SaveOrUpdateRefreshToken(*newRefresh)
	}

	// Get latest added_at timestamp from DB
	latestAddedAt, err := repository.GetLatestAddedAt()
	if err != nil {
		fmt.Printf("⚠️ Error getting latest added_at: %v\n", err)
		latestAddedAt = time.Time{}
	}

	success := 0
	skipped := 0
	var newest, oldest time.Time

	offset := 0
	limit := 50

	for {
		page, err := services.GetUserSavedTracksPage(accessTok, offset, limit)
		if err != nil {
			fmt.Printf("❌ Failed to fetch saved tracks: %v\n", err)
			break
		}

		if len(page.Items) == 0 {
			break
		}

		for _, item := range page.Items {
			parsedAddedAt, err := time.Parse(time.RFC3339, item.AddedAt)
			if err != nil {
				continue
			}

			// ✅ Early exit if this item is older or equal to our latest saved
			if !latestAddedAt.IsZero() && !parsedAddedAt.After(latestAddedAt) {
				fmt.Println("🎯 Already up to date — stopping fetch early.")
				goto DONE
			}

			track := item.Track
			if len(track.Artists) == 0 || len(track.Album.Images) == 0 {
				continue // skip incomplete data
			}

			artist := track.Artists[0]
			album := track.Album
			image := album.Images[0]

			err = models.InsertRecentlyLiked(
				track.ID,
				track.Name,
				strconv.Itoa(track.Popularity),
				album.Name,
				album.AlbumType,
				image.URL,
				album.ReleaseDate,
				album.ReleaseDatePrecision,
				artist.Name,
				artist.ID,
				artist.Href,
				artist.URI,
				album.TotalTracks,
				image.Width,
				image.Height,
				parsedAddedAt,
			)
			if err != nil {
				if !strings.Contains(err.Error(), "duplicate") {
					fmt.Printf("InsertRecentlyLiked error: %v\n", err)
				}
			} else {
				success++
				if success == 1 {
					newest = parsedAddedAt
				}
				oldest = parsedAddedAt
			}
		}

		offset += limit
		time.Sleep(300 * time.Millisecond) // to avoid hitting rate limits
	}

DONE:
	if success > 0 {
		fmt.Printf("💚 saved %d new liked tracks (skipped %d) | range: %s to %s | %s\n",
			success, skipped,
			oldest.Format("15:04:05"),
			newest.Format("15:04:05"),
			time.Now().Format(time.Kitchen))
	} else {
		fmt.Printf("💤 no new liked tracks (skipped %d) | %s\n",
			skipped,
			time.Now().Format(time.Kitchen))
	}
}

// GET CURRENTLY PLAYIN
func GetCurrentlyPLaying() (*services.CurrentlyPlaying, error) {
	refreshTok, err := repository.GetRefreshToken()
	if err != nil {
		fmt.Print("failed to get refresh token ")
	}

	accessTok, newRefresh, err := services.RefreshAccessToken(refreshTok)

	if newRefresh != nil && *newRefresh != refreshTok {
		_ = repository.SaveOrUpdateRefreshToken(*newRefresh)
	}

	currentlyListening, err := services.GetCurrentlyListening(accessTok)
	if err != nil {
		fmt.Print(err)
	}
	jsonData, err := json.Marshal(currentlyListening)
	if err != nil {
		fmt.Printf("Error serializing userTracks: %v\n", err)
		fmt.Print(err, "error here ")
	}
	os.WriteFile("DEBUG_current_listening_track.json", jsonData, 0644)

	// fmt.Print(currentlyListening, "YAYYYYY")

	return currentlyListening, nil

}

// call get artists genre by calling the get artist function and add genre to table
func GetGenreOfRecentlyLiked(batchSize int) int {
	fmt.Println("🎶 Updating genres for recently_liked table...")

	refreshTok, err := repository.GetRefreshToken()
	if err != nil {
		fmt.Println("Failed to get refresh token")
		return 0
	}

	accessTok, _, err := services.RefreshAccessToken(refreshTok)
	if err != nil {
		fmt.Printf("Failed to refresh token: %v\n", err)
		return 0
	}

	// Fetch only a batch
	query := `
        SELECT id, artist_id
        FROM recently_liked
        WHERE genre IS NULL OR genre = ''
        ORDER BY id
        LIMIT $1;
    `
	rows, err := repository.Pool.Query(context.Background(), query, batchSize)
	if err != nil {
		fmt.Printf("Failed to fetch rows: %v\n", err)
		return 0
	}
	defer rows.Close()

	updated := 0
	for rows.Next() {
		var id int
		var artistID string
		if err := rows.Scan(&id, &artistID); err != nil {
			fmt.Printf("Row scan error: %v\n", err)
			continue
		}

		artistObj, err := services.GetArtistById(accessTok, artistID)
		if err != nil {
			if strings.Contains(err.Error(), "429") {
				fmt.Println("⚠️ Rate limited! Sleeping for 5 seconds...")
				time.Sleep(5 * time.Second)
				continue
			}
			fmt.Printf("Failed to get artist (%s): %v\n", artistID, err)
			continue
		}

		genre := ""
		if artistObj != nil && len(artistObj.Genres) > 0 {
			genre = strings.Join(artistObj.Genres, ", ")
		}

		if _, err := repository.Pool.Exec(context.Background(),
			"UPDATE recently_liked SET genre = $1 WHERE id = $2", genre, id); err != nil {
			fmt.Printf("Failed to update genre for ID %d: %v\n", id, err)
		} else {
			updated++
		}
	}
	fmt.Printf("✅ Updated %d tracks in this batch.\n", updated)
	return updated
}
