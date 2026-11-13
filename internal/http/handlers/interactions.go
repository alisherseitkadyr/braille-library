package handlers

import (
	"net/http"

	"github.com/alisherseitkadyr/braille-library/internal/domain"
	"github.com/alisherseitkadyr/braille-library/internal/http/middleware"
	"github.com/alisherseitkadyr/braille-library/internal/service"
)

type InteractionHandler struct {
	interactions *service.InteractionService
}

func NewInteractionHandler(interactions *service.InteractionService) *InteractionHandler {
	return &InteractionHandler{interactions: interactions}
}

// Create godoc
// @Summary  Record an interaction
// @Description value: read = progress 0-100, rate = 1-5, other types must not have it.
// @Description correction {position, original, suggested} is required for type correction only.
// @Description position is the character index (Unicode code point) in the item's content where original starts;
// @Description original must be exactly the text there (400 otherwise). Items without content accept any position.
// @Description rate is an upsert (one rating per user and item) and updates the item's avgRating.
// @Description favorite is stored once per user and item.
// @Tags     interactions
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body body service.InteractionInput true "Interaction"
// @Success  201 {object} domain.Interaction "created"
// @Success  200 {object} domain.Interaction "rating updated or favorite already existed"
// @Failure  400 {object} ErrorResponse
// @Failure  401 {object} ErrorResponse
// @Failure  404 {object} ErrorResponse "item not found"
// @Router   /interactions [post]
func (h *InteractionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in service.InteractionInput
	if !decodeJSON(w, r, &in) {
		return
	}
	it, created, err := h.interactions.Record(r.Context(), middleware.UserIDFrom(r.Context()), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, it)
}

// History godoc
// @Summary  My interaction history
// @Description Newest first, with the item title (empty if the item was deleted).
// @Tags     users
// @Produce  json
// @Security BearerAuth
// @Param    type  query string false "Interaction type" Enums(view, read, listen, rate, favorite, correction)
// @Param    page  query int    false "Page" minimum(1) default(1)
// @Param    limit query int    false "Page size (max 100)" minimum(1) maximum(100) default(20)
// @Success  200 {object} service.HistoryPage
// @Failure  400 {object} ErrorResponse
// @Failure  401 {object} ErrorResponse
// @Router   /users/me/history [get]
func (h *InteractionHandler) History(w http.ResponseWriter, r *http.Request) {
	page, limit, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	t := domain.InteractionType(r.URL.Query().Get("type"))
	result, err := h.interactions.History(r.Context(), middleware.UserIDFrom(r.Context()), t, page, limit)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
