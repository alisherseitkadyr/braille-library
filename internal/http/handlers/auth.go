package handlers

import (
	"net/http"

	"github.com/alisherseitkadyr/braille-library/internal/domain"
	"github.com/alisherseitkadyr/braille-library/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type LoginRequest struct {
	Email    string `json:"email" example:"reader@demo.kz"`
	Password string `json:"password" example:"Demo12345"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

// Register godoc
// @Summary  Register a new user
// @Description Role can be reader, learner or teacher (default reader); admin is not allowed.
// @Description Password needs at least 8 characters. Defaults: preferredLanguage kk, readingLevel beginner.
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body service.RegisterInput true "New user"
// @Success  201 {object} domain.User
// @Failure  400 {object} ErrorResponse
// @Failure  409 {object} ErrorResponse "email already registered"
// @Router   /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var in service.RegisterInput
	if !decodeJSON(w, r, &in) {
		return
	}
	u, err := h.auth.Register(r.Context(), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

// Login godoc
// @Summary  Log in
// @Description Returns a JWT. Use it in the Authorize dialog as "Bearer <token>".
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body LoginRequest true "Credentials"
// @Success  200 {object} LoginResponse
// @Failure  400 {object} ErrorResponse
// @Failure  401 {object} ErrorResponse "wrong email or password"
// @Router   /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var in LoginRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	token, u, err := h.auth.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, LoginResponse{Token: token, User: u})
}
