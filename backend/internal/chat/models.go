package chat

import "time"

type Message struct {
	ID        string
	MatchID   string
	SenderID  string
	Body      string
	ReadAt    *time.Time
	CreatedAt time.Time
}

const MaxMessageLength = 2000
