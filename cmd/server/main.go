package main

import (
	"example.com/spotifydb/internal/handlers"
	"example.com/spotifydb/internal/repository"
	"github.com/gin-gonic/gin"
)

// endpoints i need
// tracks

func main() {
	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

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

	/* Analytics endpoints */
	router.GET("/collection-stats", handlers.GetCollectionStats)

	/* NEW: start the background cron in its own goroutine */
	go handlers.StartSpotifyCron()

	router.Run(":8080")
}
