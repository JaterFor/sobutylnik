package notify

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// UpsertToken registers or refreshes a device's push token for a user,
// re-validating it if it had previously been marked invalid.
func (r *Repo) UpsertToken(ctx context.Context, userID, token string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO push_tokens (user_id, token) VALUES ($1, $2)
		ON CONFLICT (token) DO UPDATE SET user_id = EXCLUDED.user_id, valid = true, updated_at = now()`,
		userID, token)
	return err
}

func (r *Repo) TokensForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT token FROM push_tokens WHERE user_id = $1 AND valid`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

func (r *Repo) MarkInvalid(ctx context.Context, token string) error {
	_, err := r.pool.Exec(ctx, `UPDATE push_tokens SET valid = false, updated_at = now() WHERE token = $1`, token)
	return err
}
