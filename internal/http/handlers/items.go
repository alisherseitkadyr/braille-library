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

// List godoc
// @Summary  Search the catalog
// @Description q is a full-text search over title, description and tags (sorted by text score, sort is ignored).
// @Description Without q, sort=new (default) is newest first and sort=rating is highest rated first.
// @Tags     items
// @Produce  json
// @Param    q        query string false "Search words"
// @Param    category query string false "Category" Enums(fiction, textbook, children, news, exercises)
// @Param    level    query string false "Level" Enums(beginner, intermediate, advanced)
// @Param    language query string false "Language" Enums(kk, ru)
// @Param    sort     query string false "Sort order" Enums(new, rating) default(new)
// @Param    page     query int    false "Page" minimum(1) default(1)
// @Param    limit    query int    false "Page size (max 100)" minimum(1) maximum(100) default(20)
// @Success  200 {object} service.ItemPage
// @Failure  400 {object} ErrorResponse
// @Router   /items [get]
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

// Get godoc
// @Summary  Get an item
// @Tags     items
// @Produce  json
// @Param    id  path string true "Item ID"
// @Success  200 {object} domain.Item
// @Failure  404 {object} ErrorResponse
// @Router   /items/{id} [get]
func (h *ItemHandler) Get(w http.ResponseWriter, r *http.Request) {
	item, err := h.items.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// Create godoc
// @Summary  Create an item (admin)
// @Description avgRating and ratingsCount are computed from ratings and cannot be set.
// @Tags     items
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body body service.ItemInput true "New item"
// @Success  201 {object} domain.Item
// @Failure  400 {object} ErrorResponse
// @Failure  401 {object} ErrorResponse
// @Failure  403 {object} ErrorResponse "not an admin"
// @Router   /items [post]
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

// Update godoc
// @Summary  Update an item (admin)
// @Description Replaces all editable fields; send the whole item.
// @Tags     items
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id   path string           true "Item ID"
// @Param    body body service.ItemInput true "Item"
// @Success  200 {object} domain.Item
// @Failure  400 {object} ErrorResponse
// @Failure  401 {object} ErrorResponse
// @Failure  403 {object} ErrorResponse "not an admin"
// @Failure  404 {object} ErrorResponse
// @Router   /items/{id} [put]
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

// Delete godoc
// @Summary  Delete an item (admin)
// @Tags     items
// @Security BearerAuth
// @Param    id path string true "Item ID"
// @Success  204
// @Failure  401 {object} ErrorResponse
// @Failure  403 {object} ErrorResponse "not an admin"
// @Failure  404 {object} ErrorResponse
// @Router   /items/{id} [delete]
func (h *ItemHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.items.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
