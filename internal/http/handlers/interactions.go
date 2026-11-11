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

// Create returns 201 for a new interaction, 200 when an existing rating
// was updated or the favorite already existed.
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
