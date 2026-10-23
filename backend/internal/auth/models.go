package auth

import "time"

type User struct {
	ID           string
	TelegramID   *int64
	Email        *string
	PasswordHash *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}
