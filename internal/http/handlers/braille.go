package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/alisherseitkadyr/braille-library/internal/braille"
	"github.com/alisherseitkadyr/braille-library/internal/service"
)

type BrailleHandler struct {
	items *service.ItemService
}

func NewBrailleHandler(items *service.ItemService) *BrailleHandler {
	return &BrailleHandler{items: items}
}

type ConvertRequest struct {
	Text string `json:"text" example:"Абай жолы 2026!"`
}

type ConvertResponse struct {
	Braille string   `json:"braille" example:"⠁⠃⠁⠯ ⠚⠕⠇⠮ ⠼⠃⠚⠃⠋⠖"`
	Skipped []string `json:"skipped" example:"«,»"`
}

type BrailleTable struct {
	Entries []braille.Entry `json:"entries"`
}

// Convert godoc
// @Summary  Convert text to Braille
// @Description Uncontracted Kazakh Braille (the Russian letters use the same cells) as Unicode Braille patterns.
// @Description Letters are lowercased: there is no capital sign, because sources disagree on it.
// @Description Digits get the number sign ⠼ once before each run. Spaces and line breaks stay.
// @Description Any other character is left out and listed once in skipped.
// @Tags     braille
// @Accept   json
// @Produce  json
// @Param    body body ConvertRequest true "Text"
// @Success  200 {object} ConvertResponse
// @Failure  400 {object} ErrorResponse
// @Router   /braille/convert [post]
func (h *BrailleHandler) Convert(w http.ResponseWriter, r *http.Request) {
	var in ConvertRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	b, skipped := braille.ToBraille(in.Text)
	writeJSON(w, http.StatusOK, ConvertResponse{Braille: b, Skipped: skipped})
}

// Table godoc
// @Summary  The Braille table
// @Description All cells with their dot numbers: letters in alphabet order, punctuation, the number sign, digits.
// @Tags     braille
// @Produce  json
// @Success  200 {object} BrailleTable
// @Router   /braille/table [get]
func (h *BrailleHandler) Table(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, BrailleTable{Entries: braille.Table()})
}

// Item godoc
// @Summary  An item's text in Braille
// @Description The content converted like POST /braille/convert; paragraphs stay separated by a blank line.
// @Tags     braille
// @Produce  json
// @Param    id path string true "Item ID"
// @Success  200 {object} ConvertResponse
// @Failure  404 {object} ErrorResponse
// @Router   /items/{id}/braille [get]
func (h *BrailleHandler) Item(w http.ResponseWriter, r *http.Request) {
	item, err := h.items.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	b, skipped := braille.ToBraille(item.Content)
	writeJSON(w, http.StatusOK, ConvertResponse{Braille: b, Skipped: skipped})
}
