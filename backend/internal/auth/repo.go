package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrAlreadyExists = errors.New("already exists")

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) FindByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, telegram_id, email, password_hash, created_at, updated_at
		FROM users WHERE telegram_id = $1`, telegramID)
	return scanUser(row)
}

func (r *Repo) FindByEmail(ctx context.Context, email string) (*User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, telegram_id, email, password_hash, created_at, updated_at
		FROM users WHERE email = $1`, email)
	return scanUser(row)
}

func (r *Repo) FindByID(ctx context.Context, id string) (*User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, telegram_id, email, password_hash, created_at, updated_at
		FROM users WHERE id = $1`, id)
	return scanUser(row)
}

func (r *Repo) CreateWithTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (telegram_id) VALUES ($1)
		RETURNING id, telegram_id, email, password_hash, created_at, updated_at`, telegramID)
	return scanUser(row)
}

func (r *Repo) CreateWithEmailPassword(ctx context.Context, email, passwordHash string) (*User, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING id, telegram_id, email, password_hash, created_at, updated_at`, email, passwordHash)
	user, err := scanUser(row)
	if err != nil && isUniqueViolation(err) {
		return nil, ErrAlreadyExists
	}
	return user, err
}

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.TelegramID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Refresh tokens

type RefreshTokenRecord struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

func (r *Repo) StoreRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`, userID, tokenHash, expiresAt)
	return err
}

func (r *Repo) FindRefreshTokenByHash(ctx context.Context, tokenHash string) (*RefreshTokenRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at
		FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	var rt RefreshTokenRecord
	err := row.Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.RevokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rt, nil
}

func (r *Repo) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL`, tokenHash)
	return err
}
