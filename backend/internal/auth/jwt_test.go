package auth

import (
	"testing"
	"time"
)

func TestJWTService_AccessTokenRoundTrip(t *testing.T) {
	svc := NewJWTService("access-secret-xxxxxxxxxxxxxxxx", "refresh-secret-xxxxxxxxxxxxxxxx", time.Minute, time.Hour)

	token, err := svc.IssueAccessToken("user-123")
	if err != nil {
		t.Fatalf("issue error: %v", err)
	}

	userID, err := svc.VerifyAccessToken(token)
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}
	if userID != "user-123" {
		t.Fatalf("got userID %q, want %q", userID, "user-123")
	}
}

func TestJWTService_ExpiredAccessToken(t *testing.T) {
	svc := NewJWTService("access-secret-xxxxxxxxxxxxxxxx", "refresh-secret-xxxxxxxxxxxxxxxx", -time.Minute, time.Hour)

	token, err := svc.IssueAccessToken("user-123")
	if err != nil {
		t.Fatalf("issue error: %v", err)
	}

	if _, err := svc.VerifyAccessToken(token); err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken for expired token, got: %v", err)
	}
}

func TestJWTService_InvalidAccessToken(t *testing.T) {
	svc := NewJWTService("access-secret-xxxxxxxxxxxxxxxx", "refresh-secret-xxxxxxxxxxxxxxxx", time.Minute, time.Hour)

	if _, err := svc.VerifyAccessToken("not-a-real-token"); err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got: %v", err)
	}
}

func TestJWTService_RefreshTokenHashIsDeterministic(t *testing.T) {
	svc := NewJWTService("access-secret-xxxxxxxxxxxxxxxx", "refresh-secret-xxxxxxxxxxxxxxxx", time.Minute, time.Hour)

	token, hash1, expiresAt, err := svc.IssueRefreshToken()
	if err != nil {
		t.Fatalf("issue error: %v", err)
	}
	if expiresAt.Before(time.Now()) {
		t.Fatal("expected refresh token expiry in the future")
	}

	hash2 := HashRefreshToken(token)
	if hash1 != hash2 {
		t.Fatal("expected deterministic hash for the same token")
	}
}
