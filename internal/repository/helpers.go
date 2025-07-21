package repository

import (
	"context"
)

// read the single row
func GetRefreshToken() (string, error) {
	var tok string
	err := Pool.QueryRow(context.Background(),
		`SELECT refresh_token FROM spotify_auth WHERE id = 1`).Scan(&tok)
	return tok, err
}

// upsert into that same row
func SaveOrUpdateRefreshToken(tok string) error {
	_, err := Pool.Exec(context.Background(), `
        INSERT INTO spotify_auth (id, refresh_token)
        VALUES (1, $1)
        ON CONFLICT (id) DO UPDATE
          SET refresh_token = EXCLUDED.refresh_token,
              updated_at    = NOW();`,
		tok)

	return err
}
