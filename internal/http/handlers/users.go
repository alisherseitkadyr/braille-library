package handlers

import (
	"net/http"

	"github.com/alisherseitkadyr/braille-library/internal/http/middleware"
	"github.com/alisherseitkadyr/braille-library/internal/service"
)

type UserHandler struct {
	users *service.UserService
}

func NewUserHandler(users *service.UserService) *UserHandler {
	return &UserHandler{users: users}
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	u, err := h.users.GetByID(r.Context(), middleware.UserIDFrom(r.Context()))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	var in service.UpdateProfileInput
	if !decodeJSON(w, r, &in) {
		return
	}
	u, err := h.users.UpdateProfile(r.Context(), middleware.UserIDFrom(r.Context()), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}
