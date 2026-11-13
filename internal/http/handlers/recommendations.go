package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/alisherseitkadyr/braille-library/internal/http/middleware"
	"github.com/alisherseitkadyr/braille-library/internal/recommend"
	"github.com/alisherseitkadyr/braille-library/internal/service"
)

const (
	defaultRecsLimit = 10
	maxRecsLimit     = 50
)

type RecommendationHandler struct {
	recs *service.RecommendationService
}

func NewRecommendationHandler(recs *service.RecommendationService) *RecommendationHandler {
	return &RecommendationHandler{recs: recs}
}

// ForMe godoc
// @Summary  My recommendations
// @Description Item-based collaborative filtering over the last 50 liked items.
// @Description If there are not enough results, items matching the profile (interests, level, language)
// @Description and then popular items are added. Items already interacted with are never returned.
// @Description reason is "because you read <title>", "matches your interests" or "popular" (score 0 for the last two).
// @Description Items have no content (the text); get it with GET /items/{id}.
// @Tags     recommendations
// @Produce  json
// @Security BearerAuth
// @Param    limit query int false "Number of items (max 50)" minimum(1) maximum(50) default(10)
// @Success  200 {object} service.RecommendationList
// @Failure  400 {object} ErrorResponse
// @Failure  401 {object} ErrorResponse
// @Router   /users/me/recommendations [get]
func (h *RecommendationHandler) ForMe(w http.ResponseWriter, r *http.Request) {
	limit, err := parseLimit(r, defaultRecsLimit, maxRecsLimit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.recs.ForUser(r.Context(), middleware.UserIDFrom(r.Context()), limit)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// Similar godoc
// @Summary  Similar items
// @Description Neighbors from item_similarities (cosine similarity with shrinkage), most similar first.
// @Description Empty until the first rebuild, or if nobody who liked this item liked anything else.
// @Tags     recommendations
// @Produce  json
// @Param    id    path  string true  "Item ID"
// @Param    limit query int    false "Number of items (max 20)" minimum(1) maximum(20) default(10)
// @Success  200 {object} service.SimilarList
// @Failure  400 {object} ErrorResponse
// @Failure  404 {object} ErrorResponse
// @Router   /items/{id}/similar [get]
func (h *RecommendationHandler) Similar(w http.ResponseWriter, r *http.Request) {
	limit, err := parseLimit(r, defaultRecsLimit, recommend.TopNeighbors)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.recs.Similar(r.Context(), chi.URLParam(r, "id"), limit)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// Rebuild godoc
// @Summary  Rebuild item similarities (admin)
// @Description Recomputes item_similarities from all interactions. Runs synchronously.
// @Tags     recommendations
// @Produce  json
// @Security BearerAuth
// @Success  200 {object} service.RebuildResult
// @Failure  401 {object} ErrorResponse
// @Failure  403 {object} ErrorResponse "not an admin"
// @Router   /admin/recommendations/rebuild [post]
func (h *RecommendationHandler) Rebuild(w http.ResponseWriter, r *http.Request) {
	result, err := h.recs.Rebuild(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// parseLimit reads limit; values above max are capped at max.
func parseLimit(r *http.Request, def, max int) (int, error) {
	s := r.URL.Query().Get("limit")
	if s == "" {
		return def, nil
	}
	limit, err := strconv.Atoi(s)
	if err != nil || limit < 1 {
		return 0, errors.New("limit must be a positive integer")
	}
	return min(limit, max), nil
}
