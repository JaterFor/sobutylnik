package notify

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/JaterFor/sobutylnik/backend/internal/auth"
)

type nameLookup interface {
	GetName(ctx context.Context, userID string) (string, error)
}

// presenceChecker reports whether a user currently has a live connection
// (backed by chat.Hub), so match notifications skip push for a participant
// who's actively using the app — mirroring chat's own online/offline split.
type presenceChecker interface {
	IsOnline(userID string) bool
}

type Service struct {
	repo     *Repo
	expo     *ExpoClient
	profiles nameLookup
	presence presenceChecker
}

func NewService(repo *Repo, expo *ExpoClient, profiles nameLookup, presence presenceChecker) *Service {
	return &Service{repo: repo, expo: expo, profiles: profiles, presence: presence}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func requireUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return "", false
	}
	return userID, true
}

type registerTokenRequest struct {
	Token string `json:"token"`
}

// POST /notifications/register-token
func (s *Service) HandleRegisterToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	var req registerTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}
	if err := s.repo.UpsertToken(r.Context(), userID, req.Token); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save push token")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

const sendTimeout = 10 * time.Second

// send pushes to every registered device of a user and marks any token
// Expo reports as permanently invalid. Errors are logged, not surfaced to
// the caller — push delivery is best-effort. It always runs on a detached
// context with its own timeout (see async, below), so a slow or hanging
// Expo call never adds latency to the request that triggered it.
func (s *Service) send(ctx context.Context, userID, title, body string, data map[string]string) {
	tokens, err := s.repo.TokensForUser(ctx, userID)
	if err != nil || len(tokens) == 0 {
		return
	}
	results, err := s.expo.Send(ctx, tokens, title, body, data)
	if err != nil {
		log.Printf("notify: expo send error: %v", err)
		return
	}
	for _, res := range results {
		if res.InvalidNow {
			_ = s.repo.MarkInvalid(ctx, res.Token)
		}
	}
}

// async runs send on a detached context so it survives the triggering
// request's context being cancelled when that request returns.
func (s *Service) async(userID, title, body string, data map[string]string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
		defer cancel()
		s.send(ctx, userID, title, body, data)
	}()
}

// NotifyNewMatch implements feed's matchNotifier interface. Only pushes to a
// participant with no live connection (see notifications spec: push is for
// the participant not currently in the app).
func (s *Service) NotifyNewMatch(ctx context.Context, userA, userB, matchID string) {
	nameA, _ := s.profiles.GetName(ctx, userA)
	nameB, _ := s.profiles.GetName(ctx, userB)

	if s.presence == nil || !s.presence.IsOnline(userA) {
		s.async(userA, "Новый мэтч!", nameB+" тоже хочет встретиться", map[string]string{"match_id": matchID, "type": "new_match"})
	}
	if s.presence == nil || !s.presence.IsOnline(userB) {
		s.async(userB, "Новый мэтч!", nameA+" тоже хочет встретиться", map[string]string{"match_id": matchID, "type": "new_match"})
	}
}

// NotifyNewMessage implements chat's messageNotifier interface, called only
// when the recipient has no live WebSocket connection.
func (s *Service) NotifyNewMessage(ctx context.Context, recipientID, senderID, matchID string) {
	senderName, _ := s.profiles.GetName(ctx, senderID)
	s.async(recipientID, senderName, "Новое сообщение", map[string]string{"match_id": matchID, "type": "new_message"})
}
