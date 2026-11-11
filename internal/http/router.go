package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/alisherseitkadyr/braille-library/internal/auth"
	"github.com/alisherseitkadyr/braille-library/internal/domain"
	"github.com/alisherseitkadyr/braille-library/internal/http/handlers"
	"github.com/alisherseitkadyr/braille-library/internal/http/middleware"
	"github.com/alisherseitkadyr/braille-library/internal/service"
)

type Deps struct {
	Mongo  *mongo.Client
	Tokens *auth.TokenManager
	Auth   *service.AuthService
	Users  *service.UserService
	Items  *service.ItemService

	Interactions *service.InteractionService
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	health := handlers.NewHealthHandler(d.Mongo)
	authH := handlers.NewAuthHandler(d.Auth)
	usersH := handlers.NewUserHandler(d.Users)
	itemsH := handlers.NewItemHandler(d.Items)
	interactionsH := handlers.NewInteractionHandler(d.Interactions)

	r.Get("/health", health.Health)

	r.Post("/auth/register", authH.Register)
	r.Post("/auth/login", authH.Login)

	r.Get("/items", itemsH.List)
	r.Get("/items/{id}", itemsH.Get)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(d.Tokens))
		r.Get("/users/me", usersH.GetMe)
		r.Put("/users/me", usersH.UpdateMe)
		r.Get("/users/me/history", interactionsH.History)
		r.Post("/interactions", interactionsH.Create)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole(domain.RoleAdmin))
			r.Post("/items", itemsH.Create)
			r.Put("/items/{id}", itemsH.Update)
			r.Delete("/items/{id}", itemsH.Delete)
		})
	})

	return r
}
