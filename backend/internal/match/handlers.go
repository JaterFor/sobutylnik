package match

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/JaterFor/sobutylnik/backend/internal/auth"
)

type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
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

type matchResponse struct {
	ID             string `json:"id"`
	OtherUserID    string `json:"other_user_id"`
	LastActivityAt string `json:"last_activity_at"`
}

// GET /matches
func (s *Service) HandleList(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	matches, err := s.repo.ListForUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load matches")
		return
	}

	resp := make([]matchResponse, 0, len(matches))
	for _, m := range matches {
		other := m.UserAID
		if other == userID {
			other = m.UserBID
		}
		resp = append(resp, matchResponse{
			ID:             m.ID,
			OtherUserID:    other,
			LastActivityAt: m.LastActivityAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// POST /matches/{matchID}/unmatch
func (s *Service) HandleUnmatch(w http.ResponseWriter, r *http.Request, matchID string) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	err := s.repo.Unmatch(r.Context(), matchID, userID)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "match not found")
		return
	}
	if errors.Is(err, ErrNotParticipant) {
		writeError(w, http.StatusForbidden, "not a participant of this match")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not unmatch")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
