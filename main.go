package main

import (
	"net/http"
	"time"

	"example.com/spotifydb/db"
	"example.com/spotifydb/models"
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
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	db.InitDB()

	router.GET("/mostPlayedTracks", getMostPlayedTracks)
	router.POST("/mostPlayedTracks", createTrack)
	router.Run(":8080")
}

func getMostPlayedTracks(context *gin.Context) {
	tracks := models.GetAllTracks(db.Pool)
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
