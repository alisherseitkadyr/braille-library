// Command evaluate is an offline quality check of the recommendations.
//
// For every user, the last 20% of the liked items (by time of the last
// interaction) are hidden. Similarities and popularity are computed from the
// remaining data only, then each method recommends 10 items the user has not
// seen in the training data:
//
//   - cf:      item-based collaborative filtering (as the API does), filled up
//     with popular items when it has fewer than 10 candidates
//   - popular: the 10 most liked items (baseline)
//
// precision@10 = hidden items found / 10, recall@10 = hidden items found / hidden
// items, averaged over users. One row per method is appended to the CSV file.
package main

import (
	"bytes"
	"cmp"
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"slices"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/alisherseitkadyr/braille-library/internal/config"
	"github.com/alisherseitkadyr/braille-library/internal/recommend"
	"github.com/alisherseitkadyr/braille-library/internal/repository/mongo"
)

const k = 10

func main() {
	size := flag.String("size", "demo", "label of the data set, written to the CSV")
	out := flag.String("out", "perf/quality.csv", "CSV file to append to")
	testShare := flag.Float64("test", 0.2, "share of each user's last liked items to hide")
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

	cur, err := db.Collection(mongo.InteractionsCollection).Aggregate(ctx, mongo.AllScoresPipeline())
	if err != nil {
		log.Fatal(err)
	}
	var all []recommend.UserItemScore
	if err := cur.All(ctx, &all); err != nil {
		log.Fatal(err)
	}

	byUser := map[bson.ObjectID][]recommend.UserItemScore{}
	for _, s := range all {
		byUser[s.UserID] = append(byUser[s.UserID], s)
	}

	// Split: hide the most recent liked items of each user.
	type split struct {
		train []recommend.UserItemScore
		test  map[bson.ObjectID]bool
	}
	var splits []split
	var train []recommend.UserItemScore
	for _, history := range byUser {
		var liked []recommend.UserItemScore
		for _, h := range history {
			if h.Score > 0 {
				liked = append(liked, h)
			}
		}
		slices.SortFunc(liked, func(a, b recommend.UserItemScore) int { return b.LastAt.Compare(a.LastAt) })
		nTest := int(math.Round(*testShare * float64(len(liked))))
		if nTest < 1 || nTest >= len(liked) {
			train = append(train, history...) // too few liked items to evaluate
			continue
		}
		s := split{test: map[bson.ObjectID]bool{}}
		for _, h := range liked[:nTest] {
			s.test[h.ItemID] = true
		}
		for _, h := range history {
			if !s.test[h.ItemID] {
				s.train = append(s.train, h)
			}
		}
		train = append(train, s.train...)
		splits = append(splits, s)
	}

	sims := recommend.Similarities(train)
	popular := popularItems(train)

	var cfP, cfR, popP, popR float64
	for _, s := range splits {
		seen := map[bson.ObjectID]bool{}
		for _, h := range s.train {
			seen[h.ItemID] = true
		}

		var cf []bson.ObjectID
		for _, c := range recommend.Recommend(s.train, sims, k) {
			cf = append(cf, c.ItemID)
		}
		cf = fill(cf, popular, seen)
		pop := fill(nil, popular, seen)

		p, r := precisionRecall(cf, s.test)
		cfP, cfR = cfP+p, cfR+r
		p, r = precisionRecall(pop, s.test)
		popP, popR = popP+p, popR+r
	}
	if len(splits) == 0 {
		log.Fatal("no user has enough liked items to evaluate")
	}
	n := float64(len(splits))

	newFile := false
	if _, err := os.Stat(*out); os.IsNotExist(err) {
		newFile = true
	}
	f, err := os.OpenFile(*out, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if newFile {
		fmt.Fprintln(f, "size,method,users,precision@10,recall@10")
	}
	rows := [][]any{{"cf", cfP / n, cfR / n}, {"popular", popP / n, popR / n}}
	for _, r := range rows {
		line := fmt.Sprintf("%s,%s,%d,%.4f,%.4f", *size, r[0], len(splits), r[1], r[2])
		fmt.Fprintln(f, line)
		fmt.Println(line)
	}
}

// popularItems returns items by the number of users who liked them in the training data.
func popularItems(train []recommend.UserItemScore) []bson.ObjectID {
	likes := map[bson.ObjectID]int{}
	for _, s := range train {
		if s.Score > 0 {
			likes[s.ItemID]++
		}
	}
	ids := make([]bson.ObjectID, 0, len(likes))
	for id := range likes {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b bson.ObjectID) int {
		if c := cmp.Compare(likes[b], likes[a]); c != 0 {
			return c
		}
		return bytes.Compare(a[:], b[:])
	})
	return ids
}

// fill adds popular items that are not seen and not already in list, up to k.
func fill(list, popular []bson.ObjectID, seen map[bson.ObjectID]bool) []bson.ObjectID {
	for _, id := range popular {
		if len(list) >= k {
			break
		}
		if !seen[id] && !slices.Contains(list, id) {
			list = append(list, id)
		}
	}
	return list
}

func precisionRecall(list []bson.ObjectID, test map[bson.ObjectID]bool) (float64, float64) {
	hits := 0
	for _, id := range list {
		if test[id] {
			hits++
		}
	}
	return float64(hits) / k, float64(hits) / float64(len(test))
}
