package chat

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub tracks live WebSocket connections per user, in-process. Sufficient for
// a single backend instance on the MVP server (see design.md: WebSocket hub
// decision). Lost on restart, which is an accepted trade-off at this scale.
type Hub struct {
	mu    sync.RWMutex
	conns map[string]map[*websocket.Conn]bool // userID -> set of connections (multi-device)
}

func NewHub() *Hub {
	return &Hub{conns: make(map[string]map[*websocket.Conn]bool)}
}

func (h *Hub) Register(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[userID] == nil {
		h.conns[userID] = make(map[*websocket.Conn]bool)
	}
	h.conns[userID][conn] = true
}

func (h *Hub) Unregister(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.conns[userID]; ok {
		delete(set, conn)
		if len(set) == 0 {
			delete(h.conns, userID)
		}
	}
}

func (h *Hub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns[userID]) > 0
}

type wsEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Send delivers an event to all of a user's live connections. No-op if the
// user has no open connection (caller decides whether to persist/push instead).
func (h *Hub) Send(userID, eventType string, payload interface{}) {
	h.mu.RLock()
	conns := make([]*websocket.Conn, 0, len(h.conns[userID]))
	for c := range h.conns[userID] {
		conns = append(conns, c)
	}
	h.mu.RUnlock()

	if len(conns) == 0 {
		return
	}

	data, err := json.Marshal(wsEvent{Type: eventType, Payload: payload})
	if err != nil {
		return
	}

	for _, c := range conns {
		_ = c.WriteMessage(websocket.TextMessage, data)
	}
}
