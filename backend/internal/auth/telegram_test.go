package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const testBotToken = "123456:TEST-BOT-TOKEN-abcdef"

func signTelegramData(data TelegramAuthData, botToken string) string {
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
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyTelegramAuth_ValidSignature(t *testing.T) {
	data := TelegramAuthData{
		ID:        42,
		FirstName: "Ivan",
		Username:  "ivan_tg",
		AuthDate:  time.Now().Unix(),
	}
	data.Hash = signTelegramData(data, testBotToken)

	if err := VerifyTelegramAuth(data, testBotToken); err != nil {
		t.Fatalf("expected valid signature, got error: %v", err)
	}
}

func TestVerifyTelegramAuth_InvalidSignature(t *testing.T) {
	data := TelegramAuthData{
		ID:        42,
		FirstName: "Ivan",
		AuthDate:  time.Now().Unix(),
		Hash:      "deadbeef00000000000000000000000000000000000000000000000000000000",
	}

	err := VerifyTelegramAuth(data, testBotToken)
	if err != ErrInvalidTelegramSignature {
		t.Fatalf("expected ErrInvalidTelegramSignature, got: %v", err)
	}
}

func TestVerifyTelegramAuth_ExpiredAuthDate(t *testing.T) {
	data := TelegramAuthData{
		ID:        42,
		FirstName: "Ivan",
		AuthDate:  time.Now().Add(-48 * time.Hour).Unix(),
	}
	data.Hash = signTelegramData(data, testBotToken)

	err := VerifyTelegramAuth(data, testBotToken)
	if err != ErrTelegramAuthExpired {
		t.Fatalf("expected ErrTelegramAuthExpired, got: %v", err)
	}
}
