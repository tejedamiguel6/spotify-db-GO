package services

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// refreshes access_token; may return a new refresh_token
func RefreshAccessToken(refreshToken string) (accessToken string, newRefreshTok *string, err error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, _ := http.NewRequest("POST", "https://accounts.spotify.com/api/token",
		strings.NewReader(data.Encode()))
	basic := base64.StdEncoding.EncodeToString([]byte(
		os.Getenv("SPOTIFY_CLIENT_ID") + ":" + os.Getenv("SPOTIFY_CLIENT_SECRET")))
	req.Header.Set("Authorization", "Basic "+basic)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = errors.New("refresh failed: " + res.Status)
		return
	}
	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token,omitempty"`
	}
	if err = json.NewDecoder(res.Body).Decode(&body); err != nil {
		return
	}

	accessToken = body.AccessToken
	if body.RefreshToken != "" {
		newRefreshTok = &body.RefreshToken
	}
	return
}

/* ─── recently‑played ─────────────────────────────────────────── */

type PlayedItem struct {
	Track struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Album struct {
			Name   string       `json:"name"`
			Images []AlbumImage `json:"images"`
		} `json:"album"`
		Artists []struct {
			ID   string
			Name string
		} `json:"artists"`
	} `json:"track"`
	PlayedAt time.Time `json:"played_at"`
}

type AlbumImage struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}

type RecentlyPlayedResponse struct {
	Items   []PlayedItem `json:"items"`
	Next    *string      `json:"next"`
	Cursors struct {
		After  *string `json:"after"`
		Before *string `json:"before"`
	} `json:"cursors"`
}

// this is to get the genre of the artist
type Artist struct {
	ID     string     `json:"id"`
	Name   string     `json:"name"`
	Genres []string   `json:"genres"`
	Images []struct { //todo: reference  album images
		URL    string `json:"url"`
		Height int    `json:"height"`
		Width  int    `json:"width"`
	}
}

type TrackDetails struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Artists []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"artists"`
	Album struct {
		Name   string       `json:"name"`
		Images []AlbumImage `json:"images"`
	} `json:"album"`
}

// user saved tracks
type UserSavedTracks struct {
	Items  []UserSavedItems `json:"items"`
	Next   string           `json:"next"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
	Total  int              `json:"total"`
}

type UserSavedItems struct {
	AddedAt string `json:"added_at"`
	Track   Track  `json:"track"`
}

type Track struct {
	ID    string `json:"id"`
	Album Album  `json:"album"`

	Name       string `json:"name"`
	Popularity int    `json:"popularity"`

	Artists []SimplifiedArtist
}

type Album struct {
	AlbumType            string             `json:"album_type"`
	TotalTracks          int                `json:"total_tracks"`
	Images               []AlbumImage       `json:"images"`
	Name                 string             `json:"name"`
	ReleaseDate          string             `json:"release_date"`
	ReleaseDatePrecision string             `json:"release_date_precision"`
	Type                 string             `json:"type"`
	Artists              []SimplifiedArtist `json:"artists"`
}

type SimplifiedArtist struct {
	Href string `json:"href"`
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // should be "artist"
	URI  string `json:"uri"`
}

// struct for currently playing
type CurrentlyPlaying struct {
	ID         string      `json:"id"`
	Timestamp  int         `json:"timestamp"`
	ProgressMS int         `json:"progress_ms"`
	Item       TrackObject `json:"item"`
}

type TrackObject struct {
	Album Album `json:"album"`
}

func GetRecentlyPlayed(accessToken string, limit int) ([]PlayedItem, error) {
	req, _ := http.NewRequest("GET",
		"https://api.spotify.com/v1/me/player/recently-played?limit="+strconv.Itoa(limit),
		nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("spotify: " + res.Status)
	}
	var body RecentlyPlayedResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body.Items, nil
}

// gets the artist by ID
func GetArtistById(accessToken, artistID string) (*Artist, error) {
	req, _ := http.NewRequest("GET",
		"https://api.spotify.com/v1/artists/"+artistID, nil)

	req.Header.Set("Authorization", "Bearer "+accessToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify failed to get artist %s: %s", artistID, res.Status)
	}

	var artist Artist
	if err := json.NewDecoder(res.Body).Decode(&artist); err != nil {
		return nil, err
	}

	return &artist, nil

}

// gets single track
func GetTrack(accessToken, trackID string) (*TrackDetails, error) {
	req, _ := http.NewRequest("GET",
		"https://api.spotify.com/v1/tracks/"+trackID, nil)

	req.Header.Set("Authorization", "Bearer "+accessToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify failed to get track id %s: %s", trackID, res.Status)

	}

	var track TrackDetails
	if err := json.NewDecoder(res.Body).Decode(&track); err != nil {
		return nil, err
	}

	fmt.Println("these are the @@@TRACKD, ", track)
	return &track, nil

}

// get User saved tracks

func GetUserSavedTracksPage(accessToken string, offset, limit int) (*UserSavedTracks, error) {
	url := fmt.Sprintf("https://api.spotify.com/v1/me/tracks?offset=%d&limit=%d", offset, limit)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusTooManyRequests {
		retryAfter := res.Header.Get("Retry-After")
		if retryAfter != "" {
			retrySeconds, err := strconv.Atoi(retryAfter)
			if err == nil {
				fmt.Printf("Rate limited. Retrying after %d seconds...\n", retrySeconds)
				time.Sleep(time.Duration(retrySeconds+1) * time.Second)
				return GetUserSavedTracksPage(accessToken, offset, limit)
			}
		}
		return nil, fmt.Errorf("spotify: 429 Too Many Requests (no Retry-After)")
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify failed to get saved tracks at offset %d", offset)
	}

	var page UserSavedTracks
	if err := json.NewDecoder(res.Body).Decode(&page); err != nil {
		return nil, err
	}

	return &page, nil
}

// function to get currently listening
func GetCurrentlyListening(accessToken string) (*CurrentlyPlaying, error) {
	req, _ := http.NewRequest("GET",
		"https://api.spotify.com/v1/me/player/currently-playing", nil)

	req.Header.Set("Authorization", "Bearer "+accessToken)

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		fmt.Printf("Error getting currently listening to")
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get currently listening to tracks")
	}

	var currentlyPlaying CurrentlyPlaying
	if err := json.NewDecoder(res.Body).Decode(&currentlyPlaying); err != nil {
		return nil, err
	}

	return &currentlyPlaying, nil

}
