package spotify

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
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Genres []string `json:"genres"`
	Images []struct {
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

	// fmt.Println("artist-???", artist)

	return &artist, nil

}

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
