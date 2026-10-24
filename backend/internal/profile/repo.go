package profile

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrPhotoLimitReached = errors.New("photo limit reached")
var ErrLastPhoto = errors.New("cannot delete the last photo")

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// GetName returns just the display name, for lightweight lookups (e.g.
// notification text) that don't need the full profile.
func (r *Repo) GetName(ctx context.Context, userID string) (string, error) {
	var name string
	err := r.pool.QueryRow(ctx, `SELECT name FROM profiles WHERE user_id = $1`, userID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return name, err
}

func (r *Repo) Get(ctx context.Context, userID string) (*Profile, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT user_id, name, age, gender, bio, preferred_drink, readiness_status,
		       ST_Y(location::geometry), ST_X(location::geometry), location_updated_at,
		       created_at, updated_at
		FROM profiles WHERE user_id = $1`, userID)

	var p Profile
	var lat, lng *float64
	err := row.Scan(&p.UserID, &p.Name, &p.Age, &p.Gender, &p.Bio, &p.PreferredDrink, &p.ReadinessStatus,
		&lat, &lng, &p.LocationUpdatedAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	p.Latitude, p.Longitude = lat, lng
	p.HasLocation = lat != nil && lng != nil
	return &p, nil
}

func (r *Repo) Upsert(ctx context.Context, p *Profile) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO profiles (user_id, name, age, gender, bio, preferred_drink, readiness_status, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now())
		ON CONFLICT (user_id) DO UPDATE SET
			name = EXCLUDED.name,
			age = EXCLUDED.age,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			preferred_drink = EXCLUDED.preferred_drink,
			readiness_status = EXCLUDED.readiness_status,
			updated_at = now()`,
		p.UserID, p.Name, p.Age, p.Gender, p.Bio, p.PreferredDrink, p.ReadinessStatus)
	return err
}

func (r *Repo) UpdateLocation(ctx context.Context, userID string, lat, lng float64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE profiles
		SET location = ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, location_updated_at = now()
		WHERE user_id = $1`, userID, lng, lat)
	return err
}

func (r *Repo) ListPhotos(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, object_key, position, is_primary, created_at
		FROM profile_photos WHERE user_id = $1 ORDER BY position ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []Photo
	for rows.Next() {
		var ph Photo
		if err := rows.Scan(&ph.ID, &ph.UserID, &ph.ObjectKey, &ph.Position, &ph.IsPrimary, &ph.CreatedAt); err != nil {
			return nil, err
		}
		photos = append(photos, ph)
	}
	return photos, rows.Err()
}

func (r *Repo) AddPhoto(ctx context.Context, userID, objectKey string) (*Photo, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM profile_photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return nil, err
	}
	if count >= MaxPhotos {
		return nil, ErrPhotoLimitReached
	}

	isPrimary := count == 0
	var photo Photo
	err = tx.QueryRow(ctx, `
		INSERT INTO profile_photos (user_id, object_key, position, is_primary)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, object_key, position, is_primary, created_at`,
		userID, objectKey, count, isPrimary).Scan(
		&photo.ID, &photo.UserID, &photo.ObjectKey, &photo.Position, &photo.IsPrimary, &photo.CreatedAt)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &photo, nil
}

func (r *Repo) DeletePhoto(ctx context.Context, userID, photoID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM profile_photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return err
	}
	if count <= 1 {
		return ErrLastPhoto
	}

	var wasPrimary bool
	err = tx.QueryRow(ctx, `
		DELETE FROM profile_photos WHERE id = $1 AND user_id = $2
		RETURNING is_primary`, photoID, userID).Scan(&wasPrimary)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if wasPrimary {
		if _, err := tx.Exec(ctx, `
			UPDATE profile_photos SET is_primary = true
			WHERE user_id = $1 AND id = (
				SELECT id FROM profile_photos WHERE user_id = $1 ORDER BY position ASC LIMIT 1
			)`, userID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
