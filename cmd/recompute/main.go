// Command recompute rebuilds the item_similarities collection.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alisherseitkadyr/braille-library/internal/config"
	"github.com/alisherseitkadyr/braille-library/internal/repository/mongo"
	"github.com/alisherseitkadyr/braille-library/internal/service"
)

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

	recs := service.NewRecommendationService(mongo.NewRecommendationRepository(db),
		mongo.NewItemRepository(db), mongo.NewUserRepository(db), service.ModePrecomputed)
	result, err := recs.Rebuild(ctx)
	if err != nil {
		log.Fatalf("rebuild: %v", err)
	}
	fmt.Printf("rebuilt similarities for %d items in %s\n",
		result.Items, time.Duration(result.DurationMs)*time.Millisecond)
}
