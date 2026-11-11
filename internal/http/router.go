package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/alisher/braille-library/internal/http/handlers"
)

func NewRouter(client *mongo.Client) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	health := handlers.NewHealthHandler(client)
	r.Get("/health", health.Health)

	return r
}
