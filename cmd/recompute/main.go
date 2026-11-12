// Command recompute rebuilds the item_similarities collection.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alisherseitkadyr/braille-library/internal/config"
	"github.com/alisherseitkadyr/braille-library/internal/repository/mongo"
	"github.com/alisherseitkadyr/braille-library/internal/service"
)

func main() {
	size := flag.String("size", "", "data set label; with --results, a row size,items,duration_ms is appended")
	results := flag.String("results", "", "CSV file to append the duration to (e.g. perf/rebuild.csv)")
	flag.Parse()

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
		mongo.NewItemRepository(db), mongo.NewUserRepository(db), service.ModePrecomputed, 0)
	result, err := recs.Rebuild(ctx)
	if err != nil {
		log.Fatalf("rebuild: %v", err)
	}
	fmt.Printf("rebuilt similarities for %d items in %s\n",
		result.Items, time.Duration(result.DurationMs)*time.Millisecond)

	if *results != "" {
		if err := appendRow(*results, fmt.Sprintf("%s,%d,%d", *size, result.Items, result.DurationMs)); err != nil {
			log.Fatalf("results: %v", err)
		}
	}
}

// appendRow appends a line to a CSV file, writing the header first if the file is new.
func appendRow(path, row string) error {
	_, statErr := os.Stat(path)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if os.IsNotExist(statErr) {
		fmt.Fprintln(f, "size,items,duration_ms")
	}
	_, err = fmt.Fprintln(f, row)
	return err
}
