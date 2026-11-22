package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"example.com/spotifydb/internal/repository"
	"github.com/gin-gonic/gin"
	analyzer "github.com/tejedamiguel6/song-analyzer-golang"
)

// Serializes cache-miss enrichment so a burst of requests for the same
// track results in a single model call.
var enrichMu sync.Mutex

// hasValidAPIKey reports whether the request carries the server's API_KEY.
// It fails closed when API_KEY is not configured.
func hasValidAPIKey(c *gin.Context) bool {
	expected := os.Getenv("API_KEY")
	got := c.GetHeader("X-API-Key")
	return expected != "" && subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

// TrackAnalysis returns the AI enrichment (genre, moods, refined scores,
// palette) for a track, generating and caching it on first request.
// GET /track-analysis?track=...&artist=...
// Cached results are public; a cache miss requires X-API-Key (cost guard).
func TrackAnalysis(c *gin.Context) {
	track := c.Query("track")
	artist := c.Query("artist")
	if track == "" || artist == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "track and artist query params are required"})
		return
	}

	cached, err := repository.GetEnrichment(track, artist)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cached != nil {
		c.JSON(http.StatusOK, gin.H{"data": json.RawMessage(cached), "cached": true})
		return
	}

	// Cost guard: cached analyses are public, but generating a new one calls
	// the paid Anthropic API, so it needs the same X-API-Key as writes.
	// Without this, anyone could run up the bill with made-up track names.
	if !hasValidAPIKey(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "analysis not cached yet; generating one requires X-API-Key"})
		return
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI enrichment not configured (ANTHROPIC_API_KEY missing)"})
		return
	}

	enrichMu.Lock()
	defer enrichMu.Unlock()

	// Another request may have filled the cache while we waited.
	if cached, _ := repository.GetEnrichment(track, artist); cached != nil {
		c.JSON(http.StatusOK, gin.H{"data": json.RawMessage(cached), "cached": true})
		return
	}

	enricher := analyzer.NewEnricher(apiKey, os.Getenv("ANTHROPIC_MODEL"))
	enrichment, err := enricher.EnrichTrack(track, artist, nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "enrichment failed: " + err.Error()})
		return
	}

	payload, err := json.Marshal(enrichment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := repository.SaveEnrichment(track, artist, payload, enrichment.Model); err != nil {
		fmt.Println("TrackAnalysis: failed to cache enrichment:", err)
	}

	c.JSON(http.StatusOK, gin.H{"data": json.RawMessage(payload), "cached": false})
}
