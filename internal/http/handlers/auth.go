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

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

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

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var in loginRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	token, u, err := h.auth.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, loginResponse{Token: token, User: u})
}
