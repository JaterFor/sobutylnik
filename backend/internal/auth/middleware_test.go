package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequireAuth_MissingToken(t *testing.T) {
	svc := &Service{jwt: NewJWTService("access-secret-xxxxxxxxxxxxxxxx", "refresh-secret-xxxxxxxxxxxxxxxx", time.Minute, time.Hour)}

	called := false
	handler := svc.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Fatal("handler should not be called without a token")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	svc := &Service{jwt: NewJWTService("access-secret-xxxxxxxxxxxxxxxx", "refresh-secret-xxxxxxxxxxxxxxxx", time.Minute, time.Hour)}

	handler := svc.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestRequireAuth_ValidToken(t *testing.T) {
	jwtSvc := NewJWTService("access-secret-xxxxxxxxxxxxxxxx", "refresh-secret-xxxxxxxxxxxxxxxx", time.Minute, time.Hour)
	svc := &Service{jwt: jwtSvc}

	token, err := jwtSvc.IssueAccessToken("user-42")
	if err != nil {
		t.Fatalf("issue error: %v", err)
	}

	var gotUserID string
	handler := svc.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, _ = UserIDFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	if gotUserID != "user-42" {
		t.Fatalf("got userID %q, want %q", gotUserID, "user-42")
	}
}
