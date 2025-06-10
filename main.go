package main

import (
	"time"

	"example.com/spotifydb/db"
	"example.com/spotifydb/handlers"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// endpoints i need
// tracks

func main() {
	router := gin.Default()

	// Add CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	db.InitDB()

	/* -------- API routes -------- */
	router.GET("/mostPlayedTracks", handlers.GetMostPlayedTracks)
	router.POST("/mostPlayedTracks", handlers.CreateTrack)
	router.PATCH("/mostPlayedTracks/track/:spotify_song_id", handlers.UpdateTrack)

	/* NEW: endpoint to store (or rotate) refresh_token */
	router.POST("/save-refresh", handlers.SaveRefresh)

	/* Analytics endpoints */
	router.GET("/collection-stats", handlers.GetCollectionStats)

	/* NEW: start the background cron in its own goroutine */
	go handlers.StartSpotifyCron()

	router.Run(":8080")
}
