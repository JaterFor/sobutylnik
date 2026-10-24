package moderation

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/JaterFor/sobutylnik/backend/internal/auth"
)

// matchUnmatcher is the subset of match.Repo this package needs, so blocking
// a user can also tear down any active match between them.
type matchUnmatcher interface {
	FindActiveBetween(ctx context.Context, userX, userY string) (string, error)
	Unmatch(ctx context.Context, matchID, requesterID string) error
}

type Service struct {
	repo    *Repo
	matches matchUnmatcher
}

func NewService(repo *Repo, matches matchUnmatcher) *Service {
	return &Service{repo: repo, matches: matches}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func requireUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return "", false
	}
	return userID, true
}

type reportRequest struct {
	UserID string `json:"user_id"`
	Reason string `json:"reason"`
}

// POST /moderation/report
func (s *Service) HandleReport(w http.ResponseWriter, r *http.Request) {
	reporterID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	var req reportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" || req.Reason == "" {
		writeError(w, http.StatusBadRequest, "user_id and reason are required")
		return
	}
	if req.UserID == reporterID {
		writeError(w, http.StatusBadRequest, "cannot report yourself")
		return
	}

	err := s.repo.CreateReport(r.Context(), reporterID, req.UserID, req.Reason)
	if errors.Is(err, ErrAlreadyReported) {
		writeError(w, http.StatusConflict, "already reported this user")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save report")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type blockRequest struct {
	UserID string `json:"user_id"`
}

// POST /moderation/block
func (s *Service) HandleBlock(w http.ResponseWriter, r *http.Request) {
	blockerID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	var req blockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if req.UserID == blockerID {
		writeError(w, http.StatusBadRequest, "cannot block yourself")
		return
	}

	if err := s.repo.CreateBlock(r.Context(), blockerID, req.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save block")
		return
	}

	if matchID, err := s.matches.FindActiveBetween(r.Context(), blockerID, req.UserID); err == nil {
		_ = s.matches.Unmatch(r.Context(), matchID, blockerID)
	}

	w.WriteHeader(http.StatusNoContent)
}
