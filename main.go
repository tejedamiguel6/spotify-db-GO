package main

import (
	"fmt"
	"net/http"
	"time"

	"example.com/spotifydb/db"
	"example.com/spotifydb/models"
	"example.com/spotifydb/spotify"
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
	router.GET("/mostPlayedTracks", getMostPlayedTracks)
	router.POST("/mostPlayedTracks", createTrack)
	router.PATCH("/mostPlayedTracks/track/:spotify_song_id", updateTrack)

	/* NEW: endpoint to store (or rotate) refresh_token */
	router.POST("/save-refresh", saveRefresh)

	/* NEW: start the background cron in its own goroutine */
	go startSpotifyCron()

	router.Run(":8080")
}

/* ---------- background ticker ---------- */
func startSpotifyCron() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		refreshTok, err := db.GetRefreshToken()
		if err != nil || refreshTok == "" {
			fmt.Println("cron: no refresh token stored yet")
			continue
		}

		accessTok, newRefresh, err := spotify.RefreshAccessToken(refreshTok)
		if err != nil {
			fmt.Println("cron: refresh error:", err)
			continue
		}
		if newRefresh != nil && *newRefresh != refreshTok {
			_ = db.SaveOrUpdateRefreshToken(*newRefresh)
		}

		items, err := spotify.GetRecentlyPlayed(accessTok, 50)
		if err != nil {
			fmt.Println("cron: recently-played error:", err)
			continue
		}

		success := 0

		for _, it := range items {
			artist := ""
			if len(it.Track.Artists) > 0 {
				artist = it.Track.Artists[0].Name
			}
			err = models.InsertRecentlyPlayed(
				it.Track.ID,
				it.Track.Name,
				artist,
				it.Track.Album.Name,
				it.PlayedAt,
			)
			if err != nil {
				fmt.Println("upsert error:", err) // <‑‑ temporary log
			} else {
				success++
			}

		}
		fmt.Printf("cron: stored %d plays (%d new/updated) %s\n",
			len(items), success, time.Now().Format(time.Kitchen))
	}
}

/* ---------- route to accept refresh token from Next.js ---------- */
func saveRefresh(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.RefreshToken == "" {
		c.JSON(400, gin.H{"msg": "no token"})
		return
	}
	if err := db.SaveOrUpdateRefreshToken(body.RefreshToken); err != nil {
		c.JSON(500, gin.H{"msg": "db error"})
		return
	}
	c.JSON(200, gin.H{"msg": "saved"})
}

func getMostPlayedTracks(context *gin.Context) {
	tracks := models.GetAllTracksonRepeat(db.Pool)
	context.JSON(http.StatusOK, tracks)
}

// saves to database
func createTrack(context *gin.Context) {
	var track models.Track

	err := context.ShouldBindJSON(&track)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "could not parse data"})
		return
	}

	// this saves into database
	err = track.SaveToDatabase(db.Pool)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "could not save to database"})
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "created track", "track": track})

}

// update tracks

func updateTrack(context *gin.Context) {
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
	existingTrack, err := models.GetSingleTrack(db.Pool, spotifyID)
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"message:": "track not found!"})
		return
	}

	// update the fields
	existingTrack.PlayCount = updateData.PlayCount

	// Save back to DB
	if err := existingTrack.UpdateTrackDB(db.Pool); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "could not update track"})
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "track updated"})

}
