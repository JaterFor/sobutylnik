package match

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/JaterFor/sobutylnik/backend/internal/auth"
)

// profileCards is the subset of profile.Repo + profile.R2Storage this
// package needs to show "who is this" (name, primary photo) in the match
// list, per matching spec: "базовая информация о собеседнике (фото, имя)".
type profileCards interface {
	GetCard(ctx context.Context, userID string) (name string, primaryPhotoKey *string, err error)
}

type photoURLResolver interface {
	PublicURL(objectKey string) string
}

type Service struct {
	repo     *Repo
	profiles profileCards
	photos   photoURLResolver
}

func NewService(repo *Repo, profiles profileCards, photos photoURLResolver) *Service {
	return &Service{repo: repo, profiles: profiles, photos: photos}
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
	ID             string  `json:"id"`
	OtherUserID    string  `json:"other_user_id"`
	OtherUserName  string  `json:"other_user_name"`
	OtherUserPhoto *string `json:"other_user_photo,omitempty"`
	LastActivityAt string  `json:"last_activity_at"`
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

		name, photoKey, err := s.profiles.GetCard(r.Context(), other)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load match participant")
			return
		}

		var photoURL *string
		if photoKey != nil {
			u := s.photos.PublicURL(*photoKey)
			photoURL = &u
		}

		resp = append(resp, matchResponse{
			ID:             m.ID,
			OtherUserID:    other,
			OtherUserName:  name,
			OtherUserPhoto: photoURL,
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
