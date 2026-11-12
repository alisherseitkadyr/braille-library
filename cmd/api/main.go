package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/alisherseitkadyr/braille-library/docs"
	"github.com/alisherseitkadyr/braille-library/internal/auth"
	"github.com/alisherseitkadyr/braille-library/internal/config"
	httpapi "github.com/alisherseitkadyr/braille-library/internal/http"
	"github.com/alisherseitkadyr/braille-library/internal/repository/mongo"
	"github.com/alisherseitkadyr/braille-library/internal/service"
)

// @title       Braille Library API
// @version     1.0
// @description Digital library of Kazakh Braille texts digitized by a recognition model.
// @description Readers can rate texts and correct recognition errors; corrections become training data.
// @BasePath    /
//
// @securityDefinitions.apikey BearerAuth
// @in   header
// @name Authorization
// @description Enter "Bearer <token>" with the token from POST /auth/login.
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	client, db, err := mongo.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("mongo: %v", err)
	}
	defer client.Disconnect(ctx)

	if err := mongo.EnsureIndexes(ctx, db); err != nil {
		log.Fatalf("indexes: %v", err)
	}
	log.Println("indexes ready")

	tokens := auth.NewTokenManager(cfg.JWTSecret, time.Duration(cfg.JWTTTLHours)*time.Hour)
	userRepo := mongo.NewUserRepository(db)
	itemRepo := mongo.NewItemRepository(db)

	router := httpapi.NewRouter(httpapi.Deps{
		Mongo:        client,
		CORSOrigins:  cfg.CORSOrigins,
		Tokens:       tokens,
		Auth:         service.NewAuthService(userRepo, tokens),
		Users:        service.NewUserService(userRepo),
		Items:        service.NewItemService(itemRepo),
		Interactions: service.NewInteractionService(mongo.NewInteractionRepository(db), itemRepo),
		Recommendations: service.NewRecommendationService(
			mongo.NewRecommendationRepository(db), itemRepo, userRepo),
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("API listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Println("server stopped")
}
