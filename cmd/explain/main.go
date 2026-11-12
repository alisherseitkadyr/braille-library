// Command explain runs explain("executionStats") on the main queries of the
// recommendations, saves each plan as JSON and prints a summary table
// (also saved as summary.csv) showing COLLSCAN vs IXSCAN and work done.
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/alisherseitkadyr/braille-library/internal/config"
	"github.com/alisherseitkadyr/braille-library/internal/domain"
	"github.com/alisherseitkadyr/braille-library/internal/recommend"
	"github.com/alisherseitkadyr/braille-library/internal/repository/mongo"
)

type query struct {
	name, collection string
	command          bson.D // the command inside {explain: ...}
}

func main() {
	out := flag.String("out", "perf/explain", "directory for the JSON plans and summary.csv")
	email := flag.String("email", "user1@perf.kz", "user whose recommendation queries are explained")
	full := flag.Bool("rebuild", true, "also explain the rebuild job's aggregation over all interactions")
	resummarize := flag.Bool("resummarize", false,
		"do not query MongoDB: rebuild --out/summary.csv from the saved JSON plans (keeps time_ms)")
	flag.Parse()

	if *resummarize {
		resummarizeDir(*out)
		return
	}

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

	queries, err := mainQueries(ctx, db, *email, *full)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}

	rows := [][]string{{"query", "collection", "time_ms", "plan", "indexes", "keys_examined", "docs_examined", "returned"}}
	for _, q := range queries {
		start := time.Now()
		var plan bson.M
		cmd := bson.D{{Key: "explain", Value: q.command}, {Key: "verbosity", Value: "executionStats"}}
		if err := db.RunCommand(ctx, cmd).Decode(&plan); err != nil {
			log.Fatalf("explain %s: %v", q.name, err)
		}
		elapsed := time.Since(start)

		js, err := bson.MarshalExtJSONIndent(plan, false, false, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(*out, q.name+".json"), js, 0o644); err != nil {
			log.Fatal(err)
		}

		rows = append(rows, summaryRow(q.name, q.collection, strconv.FormatInt(elapsed.Milliseconds(), 10), plan))
	}
	writeSummary(*out, rows)
}

func summaryRow(name, collection, timeMs string, plan bson.M) []string {
	s := summarize(plan)
	return []string{
		name, collection, timeMs,
		strings.Join(s.stages, "+"), strings.Join(s.indexes, "+"),
		strconv.FormatInt(s.keys, 10), strconv.FormatInt(s.docs, 10), s.returned,
	}
}

func writeSummary(dir string, rows [][]string) {
	f, err := os.Create(filepath.Join(dir, "summary.csv"))
	if err != nil {
		log.Fatal(err)
	}
	w := csv.NewWriter(f)
	_ = w.WriteAll(rows)
	f.Close()

	for _, r := range rows {
		fmt.Printf("%-22s %-17s %8s  %-28s %-34s %10s %10s %9s\n", r[0], r[1], r[2], r[3], r[4], r[5], r[6], r[7])
	}
}

// resummarizeDir rewrites dir/summary.csv from the JSON plans in dir, in the
// same row order, keeping the collection and time_ms columns of the old file.
func resummarizeDir(dir string) {
	f, err := os.Open(filepath.Join(dir, "summary.csv"))
	if err != nil {
		log.Fatal(err)
	}
	old, err := csv.NewReader(f).ReadAll()
	f.Close()
	if err != nil {
		log.Fatal(err)
	}

	rows := [][]string{old[0]}
	for _, r := range old[1:] {
		js, err := os.ReadFile(filepath.Join(dir, r[0]+".json"))
		if err != nil {
			log.Fatal(err)
		}
		var plan bson.M
		if err := bson.UnmarshalExtJSON(js, false, &plan); err != nil {
			log.Fatalf("%s: %v", r[0], err)
		}
		rows = append(rows, summaryRow(r[0], r[1], r[2], plan))
	}
	writeSummary(dir, rows)
}

// mainQueries builds the same queries the API runs for one recommendation
// request of the given user (and, optionally, the rebuild job).
func mainQueries(ctx context.Context, db *mongodriver.Database, email string, full bool) ([]query, error) {
	var user domain.User
	if err := db.Collection(mongo.UsersCollection).FindOne(ctx, bson.M{"email": email}).Decode(&user); err != nil {
		return nil, fmt.Errorf("user %s: %w", email, err)
	}
	repo := mongo.NewRecommendationRepository(db)
	history, err := repo.UserScores(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	var liked, seen []bson.ObjectID
	for _, h := range recommend.LikedHistory(history) {
		liked = append(liked, h.ItemID)
	}
	for _, h := range history {
		seen = append(seen, h.ItemID)
	}

	// The norms query of live mode gets the liked items plus all their co-liked items.
	cur, err := db.Collection(mongo.InteractionsCollection).Aggregate(ctx, mongo.CooccurrencePipeline(liked))
	if err != nil {
		return nil, err
	}
	var pairs []recommend.PairStat
	if err := cur.All(ctx, &pairs); err != nil {
		return nil, err
	}
	involved := slices.Clone(liked)
	for _, p := range pairs {
		if !slices.Contains(involved, p.ItemJ) {
			involved = append(involved, p.ItemJ)
		}
	}
	candidates := involved[min(len(liked), len(involved)):min(len(involved), len(liked)+10)]

	aggregate := func(name, coll string, p mongodriver.Pipeline) query {
		return query{name, coll, bson.D{{Key: "aggregate", Value: coll}, {Key: "pipeline", Value: p}, {Key: "cursor", Value: bson.D{}}}}
	}
	find := func(name, coll string, filter any, sort bson.D, limit int) query {
		cmd := bson.D{{Key: "find", Value: coll}, {Key: "filter", Value: filter}}
		if sort != nil {
			cmd = append(cmd, bson.E{Key: "sort", Value: sort})
		}
		if limit > 0 {
			cmd = append(cmd, bson.E{Key: "limit", Value: limit})
		}
		return query{name, coll, cmd}
	}

	profile := domain.PopularFilter{Categories: user.Interests, Level: user.ReadingLevel, Language: user.PreferredLanguage}
	profileFilter, popularSort := mongo.PopularQuery(profile, seen)
	globalFilter, _ := mongo.PopularQuery(domain.PopularFilter{}, seen)

	queries := []query{
		aggregate("user_scores", mongo.InteractionsCollection, mongo.UserScoresPipeline(user.ID)),
		find("similarities_by_item", mongo.ItemSimilaritiesCollection, bson.M{"itemId": bson.M{"$in": liked}}, nil, 0),
		find("items_by_ids", mongo.ItemsCollection, bson.M{"_id": bson.M{"$in": candidates}}, nil, 0),
		find("popular_by_profile", mongo.ItemsCollection, profileFilter, popularSort, 10),
		find("popular_global", mongo.ItemsCollection, globalFilter, popularSort, 10),
		aggregate("live_cooccurrence", mongo.InteractionsCollection, mongo.CooccurrencePipeline(liked)),
		aggregate("live_norms", mongo.InteractionsCollection, mongo.NormsPipeline(involved)),
		find("history_page", mongo.InteractionsCollection, bson.M{"userId": user.ID},
			bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}, 20),
	}
	if full {
		queries = append(queries, aggregate("rebuild_scores", mongo.InteractionsCollection, mongo.LikedScoresPipeline()))
	}
	return queries, nil
}

type summary struct {
	stages, indexes []string
	keys, docs      int64
	returned        string // executionStats.nReturned, "-" when the plan has none at the top
}

// summarize walks an explain document of any shape (find, classic or SBE
// aggregate, $lookup stages) and collects the winning plan's stages, index names and
// the total keys and documents examined.
func summarize(plan bson.M) summary {
	s := summary{returned: "-"}
	switch es := plan["executionStats"].(type) {
	case bson.D:
		for _, e := range es {
			if e.Key == "nReturned" {
				s.returned = strconv.FormatInt(toInt(e.Value), 10)
			}
		}
	case bson.M:
		s.returned = strconv.FormatInt(toInt(es["nReturned"]), 10)
	}
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case bson.M:
			for k, val := range x {
				if k == "rejectedPlans" {
					continue // only the winning plan matters
				}
				switch k {
				case "stage":
					if name, ok := val.(string); ok && isAccessStage(name) && !slices.Contains(s.stages, name) {
						s.stages = append(s.stages, name)
					}
				case "indexName":
					if name, ok := val.(string); ok && !slices.Contains(s.indexes, name) {
						s.indexes = append(s.indexes, name)
					}
				case "indexesUsed": // $lookup stage stats
					if arr, ok := val.(bson.A); ok {
						for _, n := range arr {
							if name, ok := n.(string); ok && !slices.Contains(s.indexes, name) {
								s.indexes = append(s.indexes, name)
							}
						}
					}
				case "collectionScans": // $lookup stage stats
					if toInt(val) > 0 && !slices.Contains(s.stages, "COLLSCAN(lookup)") {
						s.stages = append(s.stages, "COLLSCAN(lookup)")
					}
				case "totalKeysExamined":
					s.keys += toInt(val)
				case "totalDocsExamined":
					s.docs += toInt(val)
				}
				walk(val)
			}
		case bson.D:
			m := bson.M{}
			for _, e := range x {
				m[e.Key] = e.Value
			}
			walk(m)
		case bson.A:
			for _, e := range x {
				walk(e)
			}
		}
	}
	walk(plan)
	slices.Sort(s.stages)
	return s
}

// isAccessStage keeps the stages that say how data is read.
func isAccessStage(name string) bool {
	switch name {
	case "COLLSCAN", "IXSCAN", "FETCH", "SORT", "SORT_MERGE", "EXPRESS_IXSCAN", "IDHACK", "EXPRESS_CLUSTERED_IXSCAN":
		return true
	}
	return false
}

func toInt(v any) int64 {
	switch n := v.(type) {
	case int32:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	}
	return 0
}
