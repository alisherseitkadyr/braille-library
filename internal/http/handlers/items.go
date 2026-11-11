package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/alisherseitkadyr/braille-library/internal/domain"
	"github.com/alisherseitkadyr/braille-library/internal/service"
)

type ItemHandler struct {
	items *service.ItemService
}

func NewItemHandler(items *service.ItemService) *ItemHandler {
	return &ItemHandler{items: items}
}

func (h *ItemHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	q := r.URL.Query()
	result, err := h.items.List(r.Context(), domain.ItemFilter{
		Query:    q.Get("q"),
		Category: domain.Category(q.Get("category")),
		Level:    domain.Level(q.Get("level")),
		Language: domain.Language(q.Get("language")),
		Sort:     q.Get("sort"),
		Page:     page,
		Limit:    limit,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *ItemHandler) Get(w http.ResponseWriter, r *http.Request) {
	item, err := h.items.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in service.ItemInput
	if !decodeJSON(w, r, &in) {
		return
	}
	item, err := h.items.Create(r.Context(), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *ItemHandler) Update(w http.ResponseWriter, r *http.Request) {
	var in service.ItemInput
	if !decodeJSON(w, r, &in) {
		return
	}
	item, err := h.items.Update(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *ItemHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.items.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
