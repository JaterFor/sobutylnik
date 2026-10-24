package chat

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) CreateMessage(ctx context.Context, matchID, senderID, body string) (*Message, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO messages (match_id, sender_id, body)
		VALUES ($1, $2, $3)
		RETURNING id, match_id, sender_id, body, read_at, created_at`,
		matchID, senderID, body)

	var m Message
	err := row.Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Body, &m.ReadAt, &m.CreatedAt)
	return &m, err
}

func (r *Repo) ListHistory(ctx context.Context, matchID string) ([]Message, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, match_id, sender_id, body, read_at, created_at
		FROM messages WHERE match_id = $1 ORDER BY created_at ASC`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Body, &m.ReadAt, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// MarkReadByRecipient marks all unread messages in a match as read, for
// messages not sent by the recipient themself. Returns the IDs that were
// just marked read (to notify their senders over the websocket hub).
func (r *Repo) MarkReadByRecipient(ctx context.Context, matchID, recipientID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		UPDATE messages SET read_at = now()
		WHERE match_id = $1 AND sender_id <> $2 AND read_at IS NULL
		RETURNING id`, matchID, recipientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
