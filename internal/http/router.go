package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/alisherseitkadyr/braille-library/internal/auth"
	"github.com/alisherseitkadyr/braille-library/internal/domain"
	"github.com/alisherseitkadyr/braille-library/internal/http/handlers"
	"github.com/alisherseitkadyr/braille-library/internal/http/middleware"
	"github.com/alisherseitkadyr/braille-library/internal/service"
)

type Deps struct {
	Mongo       *mongo.Client
	CORSOrigins []string
	Tokens      *auth.TokenManager
	Auth        *service.AuthService
	Users       *service.UserService
	Items       *service.ItemService

	Interactions    *service.InteractionService
	Recommendations *service.RecommendationService
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.CORS(d.CORSOrigins))

	health := handlers.NewHealthHandler(d.Mongo)
	authH := handlers.NewAuthHandler(d.Auth)
	usersH := handlers.NewUserHandler(d.Users)
	itemsH := handlers.NewItemHandler(d.Items)
	interactionsH := handlers.NewInteractionHandler(d.Interactions)
	recsH := handlers.NewRecommendationHandler(d.Recommendations)

	r.Get("/health", health.Health)
	// The API has no page of its own at "/": send visitors to the API docs.
	// (The web UI is a separate service on port 3000.)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/index.html", http.StatusFound)
	})
	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/index.html", http.StatusMovedPermanently)
	})
	r.Get("/docs/*", httpSwagger.Handler(httpSwagger.URL("/docs/doc.json")))

	r.Post("/auth/register", authH.Register)
	r.Post("/auth/login", authH.Login)

	r.Get("/items", itemsH.List)
	r.Get("/items/{id}", itemsH.Get)
	r.Get("/items/{id}/similar", recsH.Similar)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(d.Tokens))
		r.Get("/users/me", usersH.GetMe)
		r.Put("/users/me", usersH.UpdateMe)
		r.Get("/users/me/history", interactionsH.History)
		r.Get("/users/me/recommendations", recsH.ForMe)
		r.Post("/interactions", interactionsH.Create)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole(domain.RoleAdmin))
			r.Post("/items", itemsH.Create)
			r.Put("/items/{id}", itemsH.Update)
			r.Delete("/items/{id}", itemsH.Delete)
			r.Post("/admin/recommendations/rebuild", recsH.Rebuild)
		})
	})

	return r
}
