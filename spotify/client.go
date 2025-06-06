package spotify

import (
	"encoding/base64"
	"encoding/json"
	"errors"
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
		ID      string                  `json:"id"`
		Name    string                  `json:"name"`
		Album   struct{ Name string }   `json:"album"`
		Artists []struct{ Name string } `json:"artists"`
	} `json:"track"`
	PlayedAt time.Time `json:"played_at"`
}

func GetRecentlyPlayed(accessToken string, limit int) ([]PlayedItem, error) {
	req, _ := http.NewRequest("GET",
		"https://api.spotify.com/v1/me/player/recently-played?after=1735689600000&limit"+strconv.Itoa(limit),
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
	var body struct{ Items []PlayedItem }
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body.Items, nil
}
