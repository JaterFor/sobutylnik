package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type Service struct {
	repo       *Repo
	jwt        *JWTService
	botToken   string
	loginLimit *loginRateLimiter
}

func NewService(repo *Repo, jwt *JWTService, botToken string) *Service {
	return &Service{
		repo:       repo,
		jwt:        jwt,
		botToken:   botToken,
		loginLimit: newLoginRateLimiter(5, time.Minute),
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (s *Service) issueTokens(w http.ResponseWriter, r *http.Request, userID string) {
	access, err := s.jwt.IssueAccessToken(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue access token")
		return
	}
	refresh, refreshHash, expiresAt, err := s.jwt.IssueRefreshToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue refresh token")
		return
	}
	if err := s.repo.StoreRefreshToken(r.Context(), userID, refreshHash, expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "could not store refresh token")
		return
	}
	writeJSON(w, http.StatusOK, tokenResponse{AccessToken: access, RefreshToken: refresh})
}

// POST /auth/telegram
func (s *Service) HandleTelegramLogin(w http.ResponseWriter, r *http.Request) {
	var data TelegramAuthData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := VerifyTelegramAuth(data, s.botToken); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid telegram authentication")
		return
	}

	user, err := s.repo.FindByTelegramID(r.Context(), data.ID)
	if errors.Is(err, ErrNotFound) {
		user, err = s.repo.CreateWithTelegramID(r.Context(), data.ID)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create or find user")
		return
	}

	s.issueTokens(w, r, user.ID)
}

type emailPasswordRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isValidEmail(email string) bool {
	at := strings.IndexByte(email, '@')
	if at <= 0 || at == len(email)-1 {
		return false
	}
	domain := email[at+1:]
	return strings.Contains(domain, ".")
}

// POST /auth/register
func (s *Service) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req emailPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	email := normalizeEmail(req.Email)
	if !isValidEmail(email) {
		writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}
	if err := ValidatePasswordStrength(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not hash password")
		return
	}

	user, err := s.repo.CreateWithEmailPassword(r.Context(), email, hash)
	if errors.Is(err, ErrAlreadyExists) {
		writeError(w, http.StatusConflict, "email already registered")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create user")
		return
	}

	s.issueTokens(w, r, user.ID)
}

// POST /auth/login
func (s *Service) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req emailPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	email := normalizeEmail(req.Email)

	if !s.loginLimit.Allow(email) {
		writeError(w, http.StatusTooManyRequests, "too many login attempts, try again later")
		return
	}

	const genericErr = "invalid email or password"

	user, err := s.repo.FindByEmail(r.Context(), email)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusUnauthorized, genericErr)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not look up user")
		return
	}
	if user.PasswordHash == nil || !VerifyPassword(*user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, genericErr)
		return
	}

	s.loginLimit.Reset(email)
	s.issueTokens(w, r, user.ID)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// POST /auth/refresh
func (s *Service) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	hash := HashRefreshToken(req.RefreshToken)
	rt, err := s.repo.FindRefreshTokenByHash(r.Context(), hash)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not look up refresh token")
		return
	}
	if rt.RevokedAt != nil || time.Now().After(rt.ExpiresAt) {
		writeError(w, http.StatusUnauthorized, "refresh token expired or revoked")
		return
	}

	access, err := s.jwt.IssueAccessToken(rt.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue access token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"access_token": access})
}

// POST /auth/logout
func (s *Service) HandleLogout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	hash := HashRefreshToken(req.RefreshToken)
	if err := s.repo.RevokeRefreshToken(r.Context(), hash); err != nil {
		writeError(w, http.StatusInternalServerError, "could not revoke refresh token")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
