package main

import (
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
		// Allow /save-refresh to bypass (for initial setup)
		if c.Request.URL.Path == "/save-refresh" {
			c.Next()
			return
		}

		// For public portfolio: Allow all GET requests (read-only)
		if c.Request.Method == "GET" {
			c.Next()
			return
		}

		// Require API key for write operations (POST, PATCH, DELETE)
		apiKey := c.GetHeader("X-API-Key")
		expectedKey := os.Getenv("API_KEY")

		if apiKey != expectedKey {
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

	/* Track detail endpoints */
	router.GET("/tracks/:id/streak", handlers.GetTrackStreak)
	router.GET("/tracks/:id/stats", handlers.GetTrackStats)
	router.GET("/tracks/:id/daily", handlers.GetTrackDaily)
	router.GET("/top-tracks", handlers.GetTopTracks)

	/* Analytics endpoints */
	router.GET("/collection-stats", handlers.GetCollectionStats)
	router.GET("/listening-stats", handlers.GetListeningStats)
	router.POST("/backfill-duration", handlers.BackfillDurationHandler)

	/* NEW: start the background cron in its own goroutine */
	go handlers.StartSpotifyCron()

	router.Run(":8080")
}
