package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidTelegramSignature = errors.New("invalid telegram signature")
var ErrTelegramAuthExpired = errors.New("telegram auth data expired")

// TelegramAuthData is the payload sent by the Telegram Login Widget.
// Fields map to the widget's documented field names.
type TelegramAuthData struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	PhotoURL  string `json:"photo_url"`
	AuthDate  int64  `json:"auth_date"`
	Hash      string `json:"hash"`
}

const telegramAuthMaxAge = 24 * time.Hour

// VerifyTelegramAuth validates the HMAC signature of Telegram Login Widget
// data against the bot token, and checks the auth_date is recent.
// https://core.telegram.org/widgets/login#checking-authorization
func VerifyTelegramAuth(data TelegramAuthData, botToken string) error {
	fields := map[string]string{
		"id":         strconv.FormatInt(data.ID, 10),
		"first_name": data.FirstName,
		"last_name":  data.LastName,
		"username":   data.Username,
		"photo_url":  data.PhotoURL,
		"auth_date":  strconv.FormatInt(data.AuthDate, 10),
	}

	var keys []string
	for k, v := range fields {
		if v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, k+"="+fields[k])
	}
	checkString := strings.Join(pairs, "\n")

	secretKey := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secretKey[:])
	mac.Write([]byte(checkString))
	expectedHash := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedHash), []byte(strings.ToLower(data.Hash))) {
		return ErrInvalidTelegramSignature
	}

	authTime := time.Unix(data.AuthDate, 0)
	if time.Since(authTime) > telegramAuthMaxAge {
		return ErrTelegramAuthExpired
	}

	return nil
}
