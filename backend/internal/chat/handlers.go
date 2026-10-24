package chat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"

	"github.com/JaterFor/sobutylnik/backend/internal/auth"
	matchpkg "github.com/JaterFor/sobutylnik/backend/internal/match"
)

type matchAccessor interface {
	Get(ctx context.Context, matchID string) (*matchpkg.Match, error)
	TouchActivity(ctx context.Context, matchID string) error
}

type blockChecker interface {
	IsBlocked(ctx context.Context, userA, userB string) (bool, error)
	IsRestricted(ctx context.Context, userID string) (bool, error)
}

// messageNotifier is called when a message is sent to a recipient with no
// live WebSocket connection, so main can wire in push delivery without chat
// depending on the notify package directly.
type messageNotifier interface {
	NotifyNewMessage(ctx context.Context, recipientID, senderID, matchID string)
}

type Service struct {
	repo     *Repo
	matches  matchAccessor
	blocks   blockChecker
	hub      *Hub
	jwt      *auth.JWTService
	notifier messageNotifier
}

func NewService(repo *Repo, matches matchAccessor, blocks blockChecker, hub *Hub, jwt *auth.JWTService, notifier messageNotifier) *Service {
	return &Service{repo: repo, matches: matches, blocks: blocks, hub: hub, jwt: jwt, notifier: notifier}
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

var ErrNotParticipant = errors.New("not a participant")
var ErrMatchInactive = errors.New("match is not active")

func (s *Service) checkActiveParticipant(ctx context.Context, matchID, userID string) (otherUserID string, err error) {
	m, err := s.matches.Get(ctx, matchID)
	if err != nil {
		return "", err
	}
	if m.UserAID != userID && m.UserBID != userID {
		return "", ErrNotParticipant
	}
	if m.Status != "active" {
		return "", ErrMatchInactive
	}
	if m.UserAID == userID {
		return m.UserBID, nil
	}
	return m.UserAID, nil
}

type messageResponse struct {
	ID        string  `json:"id"`
	SenderID  string  `json:"sender_id"`
	Body      string  `json:"body"`
	ReadAt    *string `json:"read_at,omitempty"`
	CreatedAt string  `json:"created_at"`
}

func toMessageResponse(m Message) messageResponse {
	resp := messageResponse{
		ID:        m.ID,
		SenderID:  m.SenderID,
		Body:      m.Body,
		CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if m.ReadAt != nil {
		s := m.ReadAt.Format("2006-01-02T15:04:05Z07:00")
		resp.ReadAt = &s
	}
	return resp
}

// GET /chat/{matchID}/messages — also marks unread messages as read by this
// recipient, and notifies the sender over the hub if they're online.
func (s *Service) HandleHistory(w http.ResponseWriter, r *http.Request, matchID string) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	otherUserID, err := s.checkActiveParticipant(r.Context(), matchID, userID)
	if errors.Is(err, matchpkg.ErrNotFound) {
		writeError(w, http.StatusNotFound, "match not found")
		return
	}
	if errors.Is(err, ErrNotParticipant) {
		writeError(w, http.StatusForbidden, "not a participant of this match")
		return
	}
	if errors.Is(err, ErrMatchInactive) {
		writeError(w, http.StatusForbidden, "match is no longer active")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load match")
		return
	}

	messages, err := s.repo.ListHistory(r.Context(), matchID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load messages")
		return
	}

	readIDs, err := s.repo.MarkReadByRecipient(r.Context(), matchID, userID)
	if err == nil && len(readIDs) > 0 {
		s.hub.Send(otherUserID, "messages_read", map[string]interface{}{
			"match_id":    matchID,
			"message_ids": readIDs,
		})
	}

	resp := make([]messageResponse, 0, len(messages))
	for _, m := range messages {
		resp = append(resp, toMessageResponse(m))
	}
	writeJSON(w, http.StatusOK, resp)
}

type sendMessageRequest struct {
	Body string `json:"body"`
}

// POST /chat/{matchID}/messages
func (s *Service) HandleSend(w http.ResponseWriter, r *http.Request, matchID string) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Body == "" || len(req.Body) > MaxMessageLength {
		writeError(w, http.StatusBadRequest, "body is required (max 2000 chars)")
		return
	}

	otherUserID, err := s.checkActiveParticipant(r.Context(), matchID, userID)
	if errors.Is(err, matchpkg.ErrNotFound) {
		writeError(w, http.StatusNotFound, "match not found")
		return
	}
	if errors.Is(err, ErrNotParticipant) {
		writeError(w, http.StatusForbidden, "not a participant of this match")
		return
	}
	if errors.Is(err, ErrMatchInactive) {
		writeError(w, http.StatusForbidden, "match is no longer active")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load match")
		return
	}

	if s.blocks != nil {
		blocked, err := s.blocks.IsBlocked(r.Context(), userID, otherUserID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not check block status")
			return
		}
		if blocked {
			writeError(w, http.StatusForbidden, "cannot send messages in this match")
			return
		}

		// moderation spec: a user auto-restricted after crossing the report
		// threshold loses the ability to send messages, not just feed visibility.
		restricted, err := s.blocks.IsRestricted(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not check restriction status")
			return
		}
		if restricted {
			writeError(w, http.StatusForbidden, "cannot send messages")
			return
		}
	}

	msg, err := s.repo.CreateMessage(r.Context(), matchID, userID, req.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save message")
		return
	}
	_ = s.matches.TouchActivity(r.Context(), matchID)

	resp := toMessageResponse(*msg)
	if s.hub.IsOnline(otherUserID) {
		s.hub.Send(otherUserID, "new_message", resp)
	} else if s.notifier != nil {
		s.notifier.NotifyNewMessage(r.Context(), otherUserID, userID, matchID)
	}

	writeJSON(w, http.StatusCreated, resp)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// GET /chat/ws?token=<access_token>
// Browsers' native WebSocket API cannot set an Authorization header, so the
// access token travels as a query parameter for this handshake only.
func (s *Service) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		if parsed, err := url.ParseQuery(r.URL.RawQuery); err == nil {
			token = parsed.Get("token")
		}
	}
	userID, err := s.jwt.VerifyAccessToken(token)
	if err != nil {
		http.Error(w, "invalid or missing token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.hub.Register(userID, conn)
	defer func() {
		s.hub.Unregister(userID, conn)
		_ = conn.Close()
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
		// Client->server messages are not part of MVP scope; the connection
		// is read-driven only to detect disconnects and keep it alive.
	}
}
