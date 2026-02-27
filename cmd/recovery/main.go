package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"example.com/spotifydb/internal/models"
	"example.com/spotifydb/internal/repository"
	"example.com/spotifydb/internal/services"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	// Initialize database connection
	repository.InitDB()

	fmt.Println("🔄 Starting data recovery from June 21, 2024...")

	// Get refresh token from database
	refreshToken, err := repository.GetRefreshToken()
	if err != nil || refreshToken == "" {
		log.Fatal("❌ No refresh token found. Please authenticate first using your web app.")
	}

	// Get access token
	accessToken, newRefresh, err := services.RefreshAccessToken(refreshToken)
	if err != nil {
		log.Fatal("❌ Failed to refresh access token:", err)
	}
	if newRefresh != nil && *newRefresh != refreshToken {
		repository.SaveOrUpdateRefreshToken(*newRefresh)
	}

	// Set recovery start date (June 21, 2024)
	recoveryStartDate := time.Date(2024, 6, 21, 0, 0, 0, 0, time.UTC)
	fmt.Printf("📅 Recovery period: %s to %s\n", 
		recoveryStartDate.Format("2006-01-02"), 
		time.Now().Format("2006-01-02"))

	fmt.Println("\n🎵 Starting recently played recovery...")
	recoverRecentlyPlayed(accessToken, recoveryStartDate)

	fmt.Println("\n💚 Starting recently liked recovery...")
	recoverRecentlyLiked(accessToken, recoveryStartDate)

	fmt.Println("\n✅ Recovery complete!")
}

func recoverRecentlyPlayed(accessToken string, startDate time.Time) {
	fmt.Println("⚠️  Note: Spotify's recently played API only stores ~50 recent tracks.")
	fmt.Println("📊 This will collect what's currently available, but won't recover historical data from June 21st.")
	
	// Get recent tracks from Spotify (max 50 available)
	items, err := services.GetRecentlyPlayed(accessToken, 50)
	if err != nil {
		fmt.Printf("❌ Error fetching recently played: %v\n", err)
		return
	}

	if len(items) == 0 {
		fmt.Println("📭 No recently played tracks found")
		return
	}

	success := 0
	var oldest, newest time.Time

	for i, item := range items {
		if i == 0 {
			newest = item.PlayedAt
		}
		if i == len(items)-1 {
			oldest = item.PlayedAt
		}

		// Get artist info for genre
		artist := ""
		genre := ""
		albumCoverURL := ""

		if len(item.Track.Artists) > 0 {
			artistID := item.Track.Artists[0].ID
			artistObj, err := services.GetArtistById(accessToken, artistID)
			if err != nil {
				log.Printf("Failed to fetch artist %s: %v", artistID, err)
			} else if artistObj != nil {
				artist = artistObj.Name
				if len(artistObj.Genres) > 0 {
					genre = strings.Join(artistObj.Genres, ", ")
				}
			}
		}

		if len(item.Track.Album.Images) > 0 {
			albumCoverURL = item.Track.Album.Images[0].URL
		}

		err = models.InsertRecentlyPlayed(
			item.Track.ID,
			item.Track.Name,
			artist,
			item.Track.Album.Name,
			albumCoverURL,
			genre,
			item.Track.DurationMs,
			item.PlayedAt,
		)
		if err != nil {
			if !strings.Contains(err.Error(), "duplicate") && !strings.Contains(err.Error(), "unique") {
				fmt.Printf("❌ Insert error for %s: %v\n", item.Track.Name, err)
			}
		} else {
			success++
		}

		// Add small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("✅ Recovered %d recently played tracks\n", success)
	if success > 0 {
		fmt.Printf("📊 Date range: %s to %s\n", 
			oldest.Format("2006-01-02 15:04"), 
			newest.Format("2006-01-02 15:04"))
	}
}

func recoverRecentlyLiked(accessToken string, startDate time.Time) {
	fmt.Println("🔍 Fetching all saved/liked tracks from Spotify...")
	
	success := 0
	total := 0
	recoveredFromPeriod := 0
	var oldest, newest time.Time
	
	offset := 0
	limit := 50

	for {
		page, err := services.GetUserSavedTracksPage(accessToken, offset, limit)
		if err != nil {
			fmt.Printf("❌ Failed to fetch saved tracks: %v\n", err)
			break
		}

		if len(page.Items) == 0 {
			break
		}

		fmt.Printf("📄 Processing page %d (offset %d)...\n", (offset/limit)+1, offset)

		for _, item := range page.Items {
			total++
			
			parsedAddedAt, err := time.Parse(time.RFC3339, item.AddedAt)
			if err != nil {
				continue
			}

			// Check if this track is from our recovery period
			isFromRecoveryPeriod := parsedAddedAt.After(startDate) || parsedAddedAt.Equal(startDate)
			if isFromRecoveryPeriod {
				recoveredFromPeriod++
			}

			track := item.Track
			if len(track.Artists) == 0 || len(track.Album.Images) == 0 {
				continue
			}

			artist := track.Artists[0]
			album := track.Album
			image := album.Images[0]

			err = models.InsertRecentlyLiked(
				track.ID,
				track.Name,
				fmt.Sprintf("%d", track.Popularity),
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
					fmt.Printf("❌ Insert error: %v\n", err)
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
		
		// Add delay to avoid rate limiting
		time.Sleep(300 * time.Millisecond)

		// Progress indicator
		if total%100 == 0 {
			fmt.Printf("🔄 Processed %d total tracks so far...\n", total)
		}
	}

	fmt.Printf("✅ Recovery complete!\n")
	fmt.Printf("📊 Total tracks processed: %d\n", total)
	fmt.Printf("💾 Successfully saved: %d\n", success) 
	fmt.Printf("📅 From recovery period (June 21+): %d\n", recoveredFromPeriod)
	if success > 0 {
		fmt.Printf("📊 Date range in DB: %s to %s\n", 
			oldest.Format("2006-01-02 15:04"), 
			newest.Format("2006-01-02 15:04"))
	}
}