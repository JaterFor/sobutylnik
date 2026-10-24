package match

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrNotParticipant = errors.New("user is not a participant of this match")

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

type Match struct {
	ID             string
	UserAID        string
	UserBID        string
	Status         string
	CreatedAt      time.Time
	LastActivityAt time.Time
}

// orderedPair returns (a, b) such that a < b, matching the matches table's
// user_a_id < user_b_id constraint.
func orderedPair(x, y string) (string, string) {
	if x < y {
		return x, y
	}
	return y, x
}

// CreateIfNotExists creates an active match between the two users if one
// does not already exist. Returns the match ID either way.
func (r *Repo) CreateIfNotExists(ctx context.Context, userX, userY string) (matchID string, created bool, err error) {
	a, b := orderedPair(userX, userY)

	row := r.pool.QueryRow(ctx, `
		INSERT INTO matches (user_a_id, user_b_id)
		VALUES ($1, $2)
		ON CONFLICT (user_a_id, user_b_id) DO UPDATE SET user_a_id = matches.user_a_id
		RETURNING id, (xmax = 0) AS inserted`, a, b)

	err = row.Scan(&matchID, &created)
	return matchID, created, err
}

func (r *Repo) ListForUser(ctx context.Context, userID string) ([]Match, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_a_id, user_b_id, status, created_at, last_activity_at
		FROM matches
		WHERE (user_a_id = $1 OR user_b_id = $1) AND status = 'active'
		ORDER BY last_activity_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []Match
	for rows.Next() {
		var m Match
		if err := rows.Scan(&m.ID, &m.UserAID, &m.UserBID, &m.Status, &m.CreatedAt, &m.LastActivityAt); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

func (r *Repo) Get(ctx context.Context, matchID string) (*Match, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_a_id, user_b_id, status, created_at, last_activity_at
		FROM matches WHERE id = $1`, matchID)
	var m Match
	err := row.Scan(&m.ID, &m.UserAID, &m.UserBID, &m.Status, &m.CreatedAt, &m.LastActivityAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &m, err
}

func (r *Repo) Unmatch(ctx context.Context, matchID, requesterID string) error {
	m, err := r.Get(ctx, matchID)
	if err != nil {
		return err
	}
	if m.UserAID != requesterID && m.UserBID != requesterID {
		return ErrNotParticipant
	}

	_, err = r.pool.Exec(ctx, `
		UPDATE matches SET status = 'unmatched', unmatched_at = now(), unmatched_by = $2
		WHERE id = $1`, matchID, requesterID)
	return err
}

func (r *Repo) TouchActivity(ctx context.Context, matchID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE matches SET last_activity_at = now() WHERE id = $1`, matchID)
	return err
}

// FindActiveBetween returns the active match ID between two users, if any.
func (r *Repo) FindActiveBetween(ctx context.Context, userX, userY string) (string, error) {
	a, b := orderedPair(userX, userY)
	var id string
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM matches WHERE user_a_id = $1 AND user_b_id = $2 AND status = 'active'`, a, b).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return id, err
}
