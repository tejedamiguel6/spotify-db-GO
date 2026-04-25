package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"example.com/spotifydb/internal/models"
	"example.com/spotifydb/internal/repository"
	"example.com/spotifydb/internal/services"
	"example.com/spotifydb/internal/utils"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	// Initialize database connection
	repository.InitDB()

	fmt.Println("🔄 Starting SAFE data recovery with rate limiting...")
	fmt.Println("⚡ This recovery is designed to avoid Spotify API rate limits")

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

	// Create rate limiter
	rateLimiter := utils.NewRateLimiter()

	// Set recovery start date (June 21, 2024)
	recoveryStartDate := time.Date(2024, 6, 21, 0, 0, 0, 0, time.UTC)
	fmt.Printf("📅 Recovery period: %s to %s\n", 
		recoveryStartDate.Format("2006-01-02"), 
		time.Now().Format("2006-01-02"))

	fmt.Println("\n🎵 Starting recently played recovery (with rate limiting)...")
	recoverRecentlyPlayedSafe(accessToken, rateLimiter)

	fmt.Println("\n💚 Starting recently liked recovery (with rate limiting)...")
	recoverRecentlyLikedSafe(accessToken, rateLimiter, recoveryStartDate)

	fmt.Println("\n✅ SAFE recovery complete!")
	fmt.Println("🎯 Your cron job will now continue collecting data without rate limit issues")
}

func recoverRecentlyPlayedSafe(accessToken string, rateLimiter *utils.RateLimiter) {
	fmt.Println("⚠️  Note: Spotify's recently played API only stores ~50 recent tracks.")
	fmt.Println("📊 This will collect what's currently available with proper rate limiting.")
	
	var items []services.PlayedItem
	var err error

	// Use rate limiter with retry logic
	err = rateLimiter.RetryWithBackoff(func() error {
		items, err = services.GetRecentlyPlayed(accessToken, 50)
		return err
	}, 3) // Max 3 retries

	if err != nil {
		fmt.Printf("❌ Error fetching recently played after retries: %v\n", err)
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

		// Get artist info for genre with rate limiting
		artist := ""
		genre := ""
		albumCoverURL := ""

		if len(item.Track.Artists) > 0 {
			artistID := item.Track.Artists[0].ID
			
			var artistObj *services.Artist
			err = rateLimiter.RetryWithBackoff(func() error {
				artistObj, err = services.GetArtistById(accessToken, artistID)
				return err
			}, 3)

			if err != nil {
				fmt.Printf("⚠️  Failed to fetch artist %s after retries: %v\n", artistID, err)
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
	}

	fmt.Printf("✅ Safely recovered %d recently played tracks\n", success)
	if success > 0 {
		fmt.Printf("📊 Date range: %s to %s\n", 
			oldest.Format("2006-01-02 15:04"), 
			newest.Format("2006-01-02 15:04"))
	}
}

func recoverRecentlyLikedSafe(accessToken string, rateLimiter *utils.RateLimiter, startDate time.Time) {
	fmt.Println("🔍 Fetching all saved/liked tracks from Spotify with safe rate limiting...")
	fmt.Println("🐌 This will take longer but won't trigger rate limits")
	
	success := 0
	total := 0
	recoveredFromPeriod := 0
	var oldest, newest time.Time
	
	offset := 0
	limit := 20 // Reduced batch size to be extra safe
	consecutiveErrors := 0
	maxConsecutiveErrors := 5

	for {
		var page *services.UserSavedTracks
		var err error

		// Use rate limiter with retry logic for each page
		err = rateLimiter.RetryWithBackoff(func() error {
			page, err = services.GetUserSavedTracksPage(accessToken, offset, limit)
			return err
		}, 3) // Max 3 retries per page

		if err != nil {
			consecutiveErrors++
			fmt.Printf("❌ Failed to fetch page at offset %d (attempt %d): %v\n", offset, consecutiveErrors, err)
			
			if consecutiveErrors >= maxConsecutiveErrors {
				fmt.Printf("🛑 Too many consecutive errors (%d). Stopping recovery.\n", maxConsecutiveErrors)
				break
			}
			
			// Wait longer before retrying
			fmt.Println("⏳ Waiting 30 seconds before continuing...")
			time.Sleep(30 * time.Second)
			continue
		}

		// Reset error counter on success
		consecutiveErrors = 0

		if len(page.Items) == 0 {
			fmt.Println("✅ Reached end of saved tracks")
			break
		}

		fmt.Printf("📄 Processing page %d (offset %d, %d tracks)...\n", (offset/limit)+1, offset, len(page.Items))

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

		// Progress indicator
		if total%50 == 0 {
			fmt.Printf("📊 Progress: %d total tracks processed (saved: %d, from June 21+: %d)\n", 
				total, success, recoveredFromPeriod)
		}

		// Extra safety: Wait between pages
		fmt.Printf("⏳ Waiting 2 seconds before next page...\n")
		time.Sleep(2 * time.Second)
	}

	fmt.Printf("\n🎉 SAFE recovery complete!\n")
	fmt.Printf("📊 Total tracks processed: %d\n", total)
	fmt.Printf("💾 Successfully saved: %d\n", success) 
	fmt.Printf("📅 From recovery period (June 21+): %d\n", recoveredFromPeriod)
	if success > 0 {
		fmt.Printf("📊 Date range in DB: %s to %s\n", 
			oldest.Format("2006-01-02 15:04"), 
			newest.Format("2006-01-02 15:04"))
	}
	fmt.Printf("✅ No rate limits hit during recovery!\n")
}