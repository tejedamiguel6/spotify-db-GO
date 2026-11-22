package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"example.com/spotifydb/internal/repository"
	"example.com/spotifydb/internal/services"

	"github.com/gin-gonic/gin"
)

/* ---------- embedded-friendly now-playing cache ----------
   One background poller hits Spotify; devices poll /embedded/now or
   subscribe to /events without ever touching Spotify's rate limits. */

// NowPlayingState is the flat payload served to embedded clients.
type NowPlayingState struct {
	Playing    bool   `json:"playing"`
	TrackID    string `json:"track_id,omitempty"`
	Track      string `json:"track,omitempty"`
	Artist     string `json:"artist,omitempty"`
	Album      string `json:"album,omitempty"`
	ProgressMS int    `json:"progress_ms"`
	DurationMS int    `json:"duration_ms"`
	Art        string `json:"art,omitempty"`       // smallest album image (usually 64px)
	ArtLarge   string `json:"art_large,omitempty"` // largest album image (usually 640px)
	FetchedAt  int64  `json:"fetched_at"`          // unix ms of the last Spotify poll
}

var nowPlaying struct {
	mu    sync.RWMutex
	state NowPlayingState
}

/* ---------- cached access token (one refresh per ~50 min, not per request) ---------- */

var npToken struct {
	mu     sync.Mutex
	token  string
	expiry time.Time
}

func getCachedAccessToken() (string, error) {
	npToken.mu.Lock()
	defer npToken.mu.Unlock()

	if npToken.token != "" && time.Now().Before(npToken.expiry) {
		return npToken.token, nil
	}

	refreshTok, err := repository.GetRefreshToken()
	if err != nil || refreshTok == "" {
		return "", fmt.Errorf("no refresh token stored yet")
	}

	accessTok, newRefresh, err := services.RefreshAccessToken(refreshTok)
	if err != nil {
		return "", err
	}
	if newRefresh != nil && *newRefresh != refreshTok {
		_ = repository.SaveOrUpdateRefreshToken(*newRefresh)
	}

	npToken.token = accessTok
	npToken.expiry = time.Now().Add(50 * time.Minute) // Spotify tokens last 60 min
	return accessTok, nil
}

/* ---------- SSE hub ---------- */

var sseHub = struct {
	mu   sync.Mutex
	subs map[chan []byte]struct{}
}{subs: make(map[chan []byte]struct{})}

func sseSubscribe() chan []byte {
	ch := make(chan []byte, 8)
	sseHub.mu.Lock()
	sseHub.subs[ch] = struct{}{}
	sseHub.mu.Unlock()
	return ch
}

func sseUnsubscribe(ch chan []byte) {
	sseHub.mu.Lock()
	delete(sseHub.subs, ch)
	sseHub.mu.Unlock()
}

func sseBroadcast(payload []byte) {
	sseHub.mu.Lock()
	defer sseHub.mu.Unlock()
	for ch := range sseHub.subs {
		select {
		case ch <- payload:
		default: // slow client, drop the event rather than block the poller
		}
	}
}

/* ---------- background poller ---------- */

// StartNowPlayingPoller polls Spotify's currently-playing endpoint and keeps
// the in-memory cache fresh. Broadcasts an SSE event when the track or
// play/pause state changes.
func StartNowPlayingPoller() {
	fmt.Println("🎧 Starting now-playing poller for embedded clients")

	for {
		pollNowPlaying()

		hour := time.Now().Hour()
		if hour >= 6 && hour <= 23 {
			time.Sleep(20 * time.Second)
		} else {
			time.Sleep(90 * time.Second)
		}
	}
}

func pollNowPlaying() {
	accessTok, err := getCachedAccessToken()
	if err != nil {
		return
	}

	current, err := services.GetCurrentlyListening(accessTok)
	if err != nil {
		fmt.Printf("now-playing poller: %v\n", err)
		return
	}

	newState := NowPlayingState{FetchedAt: time.Now().UnixMilli()}
	if current != nil && current.Item.Name != "" {
		artists := make([]string, 0, len(current.Item.Artists))
		for _, a := range current.Item.Artists {
			artists = append(artists, a.Name)
		}

		newState.Playing = current.IsPlaying
		newState.TrackID = current.Item.ID
		newState.Track = current.Item.Name
		newState.Artist = strings.Join(artists, ", ")
		newState.Album = current.Item.Album.Name
		newState.ProgressMS = current.ProgressMS
		newState.DurationMS = current.Item.DurationMs

		if n := len(current.Item.Album.Images); n > 0 {
			newState.ArtLarge = current.Item.Album.Images[0].URL
			newState.Art = current.Item.Album.Images[n-1].URL
		}
	}

	nowPlaying.mu.Lock()
	changed := newState.TrackID != nowPlaying.state.TrackID ||
		newState.Playing != nowPlaying.state.Playing
	nowPlaying.state = newState
	nowPlaying.mu.Unlock()

	if changed {
		if payload, err := json.Marshal(newState); err == nil {
			sseBroadcast(payload)
		}
	}
}

// snapshot returns the cached state with progress extrapolated to "now",
// so clients see a live progress bar even between Spotify polls.
func nowPlayingSnapshot() NowPlayingState {
	nowPlaying.mu.RLock()
	s := nowPlaying.state
	nowPlaying.mu.RUnlock()

	if s.Playing && s.DurationMS > 0 {
		elapsed := int(time.Now().UnixMilli() - s.FetchedAt)
		s.ProgressMS += elapsed
		if s.ProgressMS > s.DurationMS {
			s.ProgressMS = s.DurationMS
		}
	}
	return s
}

/* ---------- GET /embedded/now ---------- */

// EmbeddedNow serves the cached now-playing state as flat JSON, or as
// key=value lines with ?format=text for boards without a JSON parser.
// ETag changes only when the track or play state changes, so devices can
// send If-None-Match and sleep on a 304.
func EmbeddedNow(c *gin.Context) {
	s := nowPlayingSnapshot()

	etag := fmt.Sprintf(`"%s-%t"`, s.TrackID, s.Playing)
	c.Header("ETag", etag)
	c.Header("Cache-Control", "no-cache")
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}

	if c.Query("format") == "text" {
		playing := 0
		if s.Playing {
			playing = 1
		}
		body := fmt.Sprintf(
			"playing=%d\ntrack=%s\nartist=%s\nalbum=%s\nprogress_ms=%d\nduration_ms=%d\nfetched_at=%d\n",
			playing, s.Track, s.Artist, s.Album, s.ProgressMS, s.DurationMS, s.FetchedAt)
		c.String(http.StatusOK, body)
		return
	}

	c.JSON(http.StatusOK, s)
}

/* ---------- GET /events (SSE) ---------- */

// NowPlayingEvents streams now-playing changes as Server-Sent Events.
// Sends the current state immediately on connect, then an event whenever
// the track or play state changes, with keepalive comments in between.
func NowPlayingEvents(c *gin.Context) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ch := sseSubscribe()
	defer sseUnsubscribe(ch)

	// initial snapshot so the device can render right away
	if payload, err := json.Marshal(nowPlayingSnapshot()); err == nil {
		fmt.Fprintf(c.Writer, "event: now-playing\ndata: %s\n\n", payload)
		flusher.Flush()
	}

	keepalive := time.NewTicker(25 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case payload := <-ch:
			fmt.Fprintf(c.Writer, "event: now-playing\ndata: %s\n\n", payload)
			flusher.Flush()
		case <-keepalive.C:
			fmt.Fprint(c.Writer, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

/* ---------- GET /health ---------- */

// Health lets devices (and load balancers) cheaply check the API is up.
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UnixMilli()})
}
