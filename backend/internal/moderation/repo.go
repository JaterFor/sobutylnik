package moderation

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAlreadyReported = errors.New("already reported this user")

type Repo struct {
	pool      *pgxpool.Pool
	threshold int
}

func NewRepo(pool *pgxpool.Pool, banThreshold int) *Repo {
	return &Repo{pool: pool, threshold: banThreshold}
}

// CreateReport stores a report and, if the reported user has now reached
// the ban threshold of unique reporters, marks them restricted.
func (r *Repo) CreateReport(ctx context.Context, reporterID, reportedID, reason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason) VALUES ($1, $2, $3)`,
		reporterID, reportedID, reason)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyReported
		}
		return err
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM reports WHERE reported_id = $1`, reportedID).Scan(&count); err != nil {
		return err
	}
	if count >= r.threshold {
		if _, err := tx.Exec(ctx, `UPDATE users SET restricted = true WHERE id = $1`, reportedID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// CreateBlock blocks a user. If the two users share an active match, the
// caller is responsible for unmatching it (see match package).
func (r *Repo) CreateBlock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2)
		ON CONFLICT (blocker_id, blocked_id) DO NOTHING`, blockerID, blockedID)
	return err
}

// IsBlocked reports whether a block exists between the two users in either direction.
func (r *Repo) IsBlocked(ctx context.Context, userA, userB string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM blocks
			WHERE (blocker_id = $1 AND blocked_id = $2) OR (blocker_id = $2 AND blocked_id = $1)
		)`, userA, userB).Scan(&exists)
	return exists, err
}

func (r *Repo) IsRestricted(ctx context.Context, userID string) (bool, error) {
	var restricted bool
	err := r.pool.QueryRow(ctx, `SELECT restricted FROM users WHERE id = $1`, userID).Scan(&restricted)
	return restricted, err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
