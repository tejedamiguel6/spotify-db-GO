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
		ID      string                  `json:"id"`
		Name    string                  `json:"name"`
		Album   struct{ Name string }   `json:"album"`
		Artists []struct{ Name string } `json:"artists"`
	} `json:"track"`
	PlayedAt time.Time `json:"played_at"`
}

type RecentlyPlayedResponse struct {
	Items   []PlayedItem `json: "items"`
	Next    *string      `json: "next"`
	Cursors struct {
		After  *string `json:"after"`
		Before *string `json:"before"`
	} `json:"cursors"`
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

// New function with cursor support
func GetRecentlyPlayedWithCursor(accessToken string, limit int, before string) (*RecentlyPlayedResponse, error) {
	baseURL := "https://api.spotify.com/v1/me/player/recently-played"
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))

	if before != "" {
		params.Set("before", before)
	}

	fullURL := baseURL + "?" + params.Encode()
	fmt.Printf("Making request to: %s\n", fullURL)

	req, _ := http.NewRequest("GET", fullURL, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("spotify: " + res.Status)
	}

	var response RecentlyPlayedResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Function to get all tracks from a specific time period
func GetRecentlyPlayedSince(accessToken string, since time.Time) ([]PlayedItem, error) {
	var allItems []PlayedItem
	limit := 50 // max allowed by Spotify
	cursor := ""
	pageCount := 0

	fmt.Printf("Fetching tracks since: %s\n", since.Format(time.RFC3339))

	for {
		pageCount++
		fmt.Printf("Fetching page %d with cursor: '%s'\n", pageCount, cursor)

		response, err := GetRecentlyPlayedWithCursor(accessToken, limit, cursor)
		if err != nil {
			return nil, fmt.Errorf("error fetching page: %v", err)
		}

		// Debug: Print the response structure
		fmt.Printf("API Response - Items count: %d\n", len(response.Items))
		if response.Cursors.Before != nil {
			fmt.Printf("API Response - Before cursor: '%s'\n", *response.Cursors.Before)
		} else {
			fmt.Printf("API Response - Before cursor: nil\n")
		}
		if response.Cursors.After != nil {
			fmt.Printf("API Response - After cursor: '%s'\n", *response.Cursors.After)
		} else {
			fmt.Printf("API Response - After cursor: nil\n")
		}

		if len(response.Items) == 0 {
			fmt.Println("No more items returned")
			break
		}

		// Check if we've gone back far enough
		oldestInThisBatch := response.Items[len(response.Items)-1].PlayedAt
		fmt.Printf("Oldest in this batch: %s (target: %s)\n",
			oldestInThisBatch.Format(time.RFC3339), since.Format(time.RFC3339))

		if oldestInThisBatch.Before(since) {
			// Filter out items older than our target date
			for _, item := range response.Items {
				if item.PlayedAt.After(since) || item.PlayedAt.Equal(since) {
					allItems = append(allItems, item)
				}
			}
			fmt.Printf("Reached target date. Total items collected: %d\n", len(allItems))
			break
		}

		// Add all items from this batch
		allItems = append(allItems, response.Items...)
		fmt.Printf("Collected %d items (total: %d). Oldest in batch: %s\n",
			len(response.Items), len(allItems), oldestInThisBatch.Format(time.RFC3339))

		// Check if there's a next page
		if response.Cursors.Before == nil || *response.Cursors.Before == "" {
			fmt.Printf("No more pages available (Before cursor is %v)\n", response.Cursors.Before)
			break
		}

		// Safety check to prevent infinite loops
		if pageCount > 1000 {
			fmt.Printf("Reached maximum page limit (1000), stopping\n")
			break
		}

		cursor = *response.Cursors.Before

		// Add a small delay to be respectful to the API
		time.Sleep(200 * time.Millisecond)
	}

	fmt.Printf("Final total: %d tracks collected across %d pages\n", len(allItems), pageCount)
	return allItems, nil
}
