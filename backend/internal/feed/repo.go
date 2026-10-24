package feed

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNoLocation = errors.New("viewer has no location set")
var ErrAlreadySwiped = errors.New("already swiped on this candidate")

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

type FeedFilter struct {
	RadiusKm  float64
	Gender    *string
	Readiness *string
}

func (r *Repo) GetCandidates(ctx context.Context, viewerID string, filter FeedFilter) ([]Candidate, error) {
	var hasLocation bool
	if err := r.pool.QueryRow(ctx, `SELECT location IS NOT NULL FROM profiles WHERE user_id = $1`, viewerID).Scan(&hasLocation); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoLocation
		}
		return nil, err
	}
	if !hasLocation {
		return nil, ErrNoLocation
	}

	radiusMeters := filter.RadiusKm * 1000

	rows, err := r.pool.Query(ctx, `
		SELECT p.user_id, p.name, p.age, p.gender, p.bio, p.readiness_status,
		       ST_Distance(p.location, viewer.location) AS distance_m
		FROM profiles p
		JOIN users u ON u.id = p.user_id
		CROSS JOIN (SELECT location FROM profiles WHERE user_id = $1) viewer
		WHERE p.user_id <> $1
		  AND p.location IS NOT NULL
		  AND u.restricted = false
		  AND ST_DWithin(p.location, viewer.location, $2)
		  AND ($3::text IS NULL OR p.gender = $3)
		  AND ($4::text IS NULL OR p.readiness_status = $4)
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $1 AND s.target_id = p.user_id)
		  AND NOT EXISTS (
		      SELECT 1 FROM blocks b
		      WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id)
		         OR (b.blocker_id = p.user_id AND b.blocked_id = $1)
		  )
		ORDER BY distance_m ASC
		LIMIT $5`,
		viewerID, radiusMeters, filter.Gender, filter.Readiness, FeedPageSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []Candidate
	var userIDs []string
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.UserID, &c.Name, &c.Age, &c.Gender, &c.Bio, &c.ReadinessStatus, &c.DistanceMeters); err != nil {
			return nil, err
		}
		candidates = append(candidates, c)
		userIDs = append(userIDs, c.UserID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(userIDs) == 0 {
		return candidates, nil
	}

	photoRows, err := r.pool.Query(ctx, `
		SELECT user_id, object_key FROM profile_photos
		WHERE user_id = ANY($1) ORDER BY user_id, position ASC`, userIDs)
	if err != nil {
		return nil, err
	}
	defer photoRows.Close()

	photosByUser := make(map[string][]string)
	for photoRows.Next() {
		var userID, objectKey string
		if err := photoRows.Scan(&userID, &objectKey); err != nil {
			return nil, err
		}
		photosByUser[userID] = append(photosByUser[userID], objectKey)
	}
	if err := photoRows.Err(); err != nil {
		return nil, err
	}

	for i := range candidates {
		candidates[i].Photos = photosByUser[candidates[i].UserID]
	}

	return candidates, nil
}

type SwipeResult struct {
	Matched bool
	MatchID string
}

// RecordSwipe stores the swiper's decision. Returns ErrAlreadySwiped if this
// pair was already evaluated. Mutual-match detection is handled by the
// caller (match package) since it needs to read the reciprocal swipe.
func (r *Repo) RecordSwipe(ctx context.Context, swiperID, targetID string, liked bool) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO swipes (swiper_id, target_id, liked) VALUES ($1, $2, $3)`,
		swiperID, targetID, liked)
	if err != nil && isUniqueViolation(err) {
		return ErrAlreadySwiped
	}
	return err
}

// HasMutualLike reports whether targetID has already liked swiperID.
func (r *Repo) HasMutualLike(ctx context.Context, swiperID, targetID string) (bool, error) {
	var liked bool
	err := r.pool.QueryRow(ctx, `
		SELECT liked FROM swipes WHERE swiper_id = $1 AND target_id = $2`, targetID, swiperID).Scan(&liked)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return liked, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
