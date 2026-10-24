package feed

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/JaterFor/sobutylnik/backend/internal/auth"
)

// matcher is the subset of match.Repo this package needs to create a match
// on mutual like, without importing match directly (avoids a cycle risk and
// keeps feed's dependency surface explicit).
type matcher interface {
	CreateIfNotExists(ctx context.Context, userX, userY string) (matchID string, created bool, err error)
}

// matchNotifier is called after a new match is created, so the caller (main)
// can wire in push notifications without feed depending on notify directly.
type matchNotifier interface {
	NotifyNewMatch(ctx context.Context, userA, userB, matchID string)
}

// blockChecker lets feed reject a swipe between blocked users even when the
// client bypasses the feed listing and calls swipe directly with a known ID.
type blockChecker interface {
	IsBlocked(ctx context.Context, userA, userB string) (bool, error)
}

// photoURLResolver turns a stored object key into a publicly fetchable URL
// (backed by profile.R2Storage), so candidate photos render the same way
// GET /profile's do.
type photoURLResolver interface {
	PublicURL(objectKey string) string
}

type Service struct {
	repo     *Repo
	matches  matcher
	notifier matchNotifier
	blocks   blockChecker
	photos   photoURLResolver
}

func NewService(repo *Repo, matches matcher, notifier matchNotifier, blocks blockChecker, photos photoURLResolver) *Service {
	return &Service{repo: repo, matches: matches, notifier: notifier, blocks: blocks, photos: photos}
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

type candidateResponse struct {
	UserID          string   `json:"user_id"`
	Name            string   `json:"name"`
	Age             int      `json:"age"`
	Gender          string   `json:"gender"`
	Bio             string   `json:"bio"`
	ReadinessStatus string   `json:"readiness_status"`
	DistanceKm      float64  `json:"distance_km"`
	Photos          []string `json:"photos"`
}

// GET /feed?radius_km=5&gender=female&readiness=now
func (s *Service) HandleGetFeed(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	radiusKm := DefaultRadiusKm
	if v := r.URL.Query().Get("radius_km"); v != "" {
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil || parsed < MinRadiusKm || parsed > MaxRadiusKm {
			writeError(w, http.StatusBadRequest, "radius_km must be between 0.5 and 50")
			return
		}
		radiusKm = parsed
	}

	var genderFilter, readinessFilter *string
	if v := r.URL.Query().Get("gender"); v != "" {
		genderFilter = &v
	}
	if v := r.URL.Query().Get("readiness"); v != "" {
		readinessFilter = &v
	}

	candidates, err := s.repo.GetCandidates(r.Context(), userID, FeedFilter{
		RadiusKm:  radiusKm,
		Gender:    genderFilter,
		Readiness: readinessFilter,
	})
	if errors.Is(err, ErrNoLocation) {
		writeError(w, http.StatusBadRequest, "set your location before browsing the feed")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load feed")
		return
	}

	resp := make([]candidateResponse, 0, len(candidates))
	for _, c := range candidates {
		photoURLs := make([]string, 0, len(c.Photos))
		for _, objectKey := range c.Photos {
			photoURLs = append(photoURLs, s.photos.PublicURL(objectKey))
		}
		resp = append(resp, candidateResponse{
			UserID:          c.UserID,
			Name:            c.Name,
			Age:             c.Age,
			Gender:          c.Gender,
			Bio:             c.Bio,
			ReadinessStatus: c.ReadinessStatus,
			DistanceKm:      c.DistanceMeters / 1000,
			Photos:          photoURLs,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

type swipeRequest struct {
	TargetUserID string `json:"target_user_id"`
	Liked        bool   `json:"liked"`
}

type swipeResponse struct {
	Matched bool   `json:"matched"`
	MatchID string `json:"match_id,omitempty"`
}

// POST /feed/swipe
func (s *Service) HandleSwipe(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	var req swipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TargetUserID == "" {
		writeError(w, http.StatusBadRequest, "target_user_id is required")
		return
	}
	if req.TargetUserID == userID {
		writeError(w, http.StatusBadRequest, "cannot swipe on yourself")
		return
	}

	if s.blocks != nil {
		blocked, err := s.blocks.IsBlocked(r.Context(), userID, req.TargetUserID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not check block status")
			return
		}
		if blocked {
			// Reject silently without revealing the block to the swiper.
			writeError(w, http.StatusNotFound, "candidate not found")
			return
		}
	}

	err := s.repo.RecordSwipe(r.Context(), userID, req.TargetUserID, req.Liked)
	if errors.Is(err, ErrAlreadySwiped) {
		writeError(w, http.StatusConflict, "already swiped on this candidate")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save swipe")
		return
	}

	if !req.Liked {
		writeJSON(w, http.StatusOK, swipeResponse{Matched: false})
		return
	}

	mutual, err := s.repo.HasMutualLike(r.Context(), userID, req.TargetUserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not check mutual like")
		return
	}
	if !mutual {
		writeJSON(w, http.StatusOK, swipeResponse{Matched: false})
		return
	}

	matchID, created, err := s.matches.CreateIfNotExists(r.Context(), userID, req.TargetUserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create match")
		return
	}
	if created && s.notifier != nil {
		s.notifier.NotifyNewMatch(r.Context(), userID, req.TargetUserID, matchID)
	}

	writeJSON(w, http.StatusOK, swipeResponse{Matched: true, MatchID: matchID})
}
