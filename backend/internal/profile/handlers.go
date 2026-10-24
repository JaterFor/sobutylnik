package profile

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/JaterFor/sobutylnik/backend/internal/auth"
)

type Service struct {
	repo    *Repo
	storage *R2Storage
}

func NewService(repo *Repo, storage *R2Storage) *Service {
	return &Service{repo: repo, storage: storage}
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

type photoResponse struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	IsPrimary bool   `json:"is_primary"`
}

type profileResponse struct {
	UserID          string          `json:"user_id"`
	Name            string          `json:"name"`
	Age             int             `json:"age"`
	Gender          string          `json:"gender"`
	Bio             string          `json:"bio"`
	PreferredDrink  *string         `json:"preferred_drink,omitempty"`
	ReadinessStatus string          `json:"readiness_status"`
	HasLocation     bool            `json:"has_location"`
	Photos          []photoResponse `json:"photos"`
}

func (s *Service) toResponse(r *http.Request, p *Profile) (profileResponse, error) {
	photos, err := s.repo.ListPhotos(r.Context(), p.UserID)
	if err != nil {
		return profileResponse{}, err
	}
	photoResp := make([]photoResponse, 0, len(photos))
	for _, ph := range photos {
		photoResp = append(photoResp, photoResponse{
			ID:        ph.ID,
			URL:       s.storage.PublicURL(ph.ObjectKey),
			IsPrimary: ph.IsPrimary,
		})
	}
	return profileResponse{
		UserID:          p.UserID,
		Name:            p.Name,
		Age:             p.Age,
		Gender:          p.Gender,
		Bio:             p.Bio,
		PreferredDrink:  p.PreferredDrink,
		ReadinessStatus: p.ReadinessStatus,
		HasLocation:     p.HasLocation,
		Photos:          photoResp,
	}, nil
}

// GET /profile
func (s *Service) HandleGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	p, err := s.repo.Get(r.Context(), userID)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "profile not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load profile")
		return
	}
	resp, err := s.toResponse(r, p)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load photos")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

type upsertProfileRequest struct {
	Name            string  `json:"name"`
	Age             int     `json:"age"`
	Gender          string  `json:"gender"`
	Bio             string  `json:"bio"`
	PreferredDrink  *string `json:"preferred_drink"`
	ReadinessStatus string  `json:"readiness_status"`
}

// PUT /profile
func (s *Service) HandleUpsert(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	var req upsertProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Age < MinAge {
		writeError(w, http.StatusBadRequest, "age must be at least 18")
		return
	}
	if !IsValidGender(req.Gender) {
		writeError(w, http.StatusBadRequest, "invalid gender")
		return
	}
	if len(req.Bio) > MaxBioLength {
		writeError(w, http.StatusBadRequest, "bio too long")
		return
	}
	if req.ReadinessStatus == "" {
		req.ReadinessStatus = "now"
	}
	if !IsValidReadinessStatus(req.ReadinessStatus) {
		writeError(w, http.StatusBadRequest, "invalid readiness_status")
		return
	}

	photos, err := s.repo.ListPhotos(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not check photos")
		return
	}
	_, existingErr := s.repo.Get(r.Context(), userID)
	isNewProfile := errors.Is(existingErr, ErrNotFound)
	if isNewProfile && len(photos) == 0 {
		writeError(w, http.StatusBadRequest, "at least one photo is required before creating a profile")
		return
	}

	p := &Profile{
		UserID:          userID,
		Name:            req.Name,
		Age:             req.Age,
		Gender:          req.Gender,
		Bio:             req.Bio,
		PreferredDrink:  req.PreferredDrink,
		ReadinessStatus: req.ReadinessStatus,
	}
	if err := s.repo.Upsert(r.Context(), p); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save profile")
		return
	}

	saved, err := s.repo.Get(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not reload profile")
		return
	}
	resp, err := s.toResponse(r, saved)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load photos")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

type locationRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// PUT /profile/location
func (s *Service) HandleUpdateLocation(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	var req locationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		writeError(w, http.StatusBadRequest, "invalid coordinates")
		return
	}
	if err := s.repo.UpdateLocation(r.Context(), userID, req.Latitude, req.Longitude); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update location")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type presignPhotoRequest struct {
	ContentType string `json:"content_type"`
}

// POST /profile/photos/presign
func (s *Service) HandlePresignPhoto(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	var req presignPhotoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ContentType == "" {
		writeError(w, http.StatusBadRequest, "content_type is required")
		return
	}

	photos, err := s.repo.ListPhotos(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not check photo count")
		return
	}
	if len(photos) >= MaxPhotos {
		writeError(w, http.StatusConflict, "photo limit reached (max 3)")
		return
	}

	uploadURL, objectKey, err := s.storage.PresignUpload(r.Context(), userID, req.ContentType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not presign upload")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"upload_url": uploadURL,
		"object_key": objectKey,
	})
}

type confirmPhotoRequest struct {
	ObjectKey string `json:"object_key"`
}

// POST /profile/photos/confirm
func (s *Service) HandleConfirmPhoto(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	var req confirmPhotoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ObjectKey == "" {
		writeError(w, http.StatusBadRequest, "object_key is required")
		return
	}

	photo, err := s.repo.AddPhoto(r.Context(), userID, req.ObjectKey)
	if errors.Is(err, ErrPhotoLimitReached) {
		writeError(w, http.StatusConflict, "photo limit reached (max 3)")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save photo")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":  photo.ID,
		"url": s.storage.PublicURL(photo.ObjectKey),
	})
}

// DELETE /profile/photos/{photoID}
func (s *Service) HandleDeletePhoto(w http.ResponseWriter, r *http.Request, photoID string) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	err := s.repo.DeletePhoto(r.Context(), userID, photoID)
	if errors.Is(err, ErrLastPhoto) {
		writeError(w, http.StatusConflict, "cannot delete the last photo")
		return
	}
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "photo not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete photo")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
