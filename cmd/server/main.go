package main

import (
	"crypto/subtle"
	"fmt"
	"os"
	"time"

	"example.com/spotifydb/internal/handlers"
	"example.com/spotifydb/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// endpoints i need
// tracks

func main() {
	router := gin.Default()

	// Rate Limiting: 100 requests per minute per IP
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  100,
	}
	store := memory.NewStore()
	rateLimiterMiddleware := mgin.NewMiddleware(limiter.New(store, rate))
	router.Use(rateLimiterMiddleware)

	// API Key Authentication Middleware
	router.Use(func(c *gin.Context) {
		// For public portfolio: Allow all GET requests (read-only)
		if c.Request.Method == "GET" {
			c.Next()
			return
		}

		// Require API key for write operations (POST, PATCH, DELETE),
		// including /save-refresh. Fail closed if API_KEY is not configured.
		apiKey := c.GetHeader("X-API-Key")
		expectedKey := os.Getenv("API_KEY")

		if expectedKey == "" || subtle.ConstantTimeCompare([]byte(apiKey), []byte(expectedKey)) != 1 {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		c.Next()
	})

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		// Allow localhost for development and any production domain
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"https://mtejeda.co",
			"https://www.mtejeda.co",
			// Add your portfolio domain here when ready
		}

		// Check if origin is allowed
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

		// If no match, allow all (since this is public data)
		if c.Writer.Header().Get("Access-Control-Allow-Origin") == "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	repository.InitDB()

	/* -------- API routes -------- */
	// depracating
	// router.GET("/mostPlayedTracks", handlers.GetMostPlayedTracks)
	router.GET("/recently-played-tracks", handlers.RecentlyPlayedTracks)
	router.GET("/now-listening-to", handlers.NowListeningToTrack)
	router.GET("/recently-liked", handlers.RecentlyLiked)

	// need endpiint for genre
	router.GET("/genre/:genre", handlers.GetUserGenre)

	// router.POST("/mostPlayedTracks", handlers.CreateTrack)
	router.PATCH("/mostPlayedTracks/track/:spotify_song_id", handlers.UpdateTrack)

	/* NEW: endpoint to store (or rotate) refresh_token */
	router.POST("/save-refresh", handlers.SaveRefresh)

	/* AI song analysis (song-analyzer enrichment, cached in Postgres) */
	router.GET("/track-analysis", handlers.TrackAnalysis)

	/* Track detail endpoints */
	router.GET("/tracks/:id/streak", handlers.GetTrackStreak)
	router.GET("/tracks/:id/stats", handlers.GetTrackStats)
	router.GET("/tracks/:id/daily", handlers.GetTrackDaily)
	router.GET("/top-tracks", handlers.GetTopTracks)
	router.GET("/calendar", handlers.GetCalendar)

	/* Analytics endpoints */
	router.GET("/collection-stats", handlers.GetCollectionStats)
	router.GET("/listening-stats", handlers.GetListeningStats)
	router.POST("/backfill-duration", handlers.BackfillDurationHandler)

	/* Embedded-friendly endpoints (Raspberry Pi / Arduino / ESP32) */
	router.GET("/health", handlers.Health)
	router.GET("/embedded/now", handlers.EmbeddedNow)
	router.GET("/events", handlers.NowPlayingEvents)

	// DISABLE_CRON=true runs the API only: no Spotify polling, no token
	// refreshes, no background writes. Use it for local testing.
	if os.Getenv("DISABLE_CRON") == "true" {
		fmt.Println("⏸️  DISABLE_CRON=true: background cron and now-playing poller are off")
	} else {
		/* NEW: start the background cron in its own goroutine */
		go handlers.StartSpotifyCron()

		/* Poller that keeps the now-playing cache fresh for /embedded/now and /events */
		go handlers.StartNowPlayingPoller()
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	router.Run(":" + port)
}
