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

// GetMe godoc
// @Summary  Get my profile
// @Tags     users
// @Produce  json
// @Security BearerAuth
// @Success  200 {object} domain.User
// @Failure  401 {object} ErrorResponse
// @Router   /users/me [get]
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	u, err := h.users.GetByID(r.Context(), middleware.UserIDFrom(r.Context()))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// UpdateMe godoc
// @Summary  Update my profile
// @Description Only the fields that are sent are changed. Email, role and password cannot be changed here.
// @Tags     users
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body body service.UpdateProfileInput true "Fields to change"
// @Success  200 {object} domain.User
// @Failure  400 {object} ErrorResponse
// @Failure  401 {object} ErrorResponse
// @Router   /users/me [put]
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
