package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/alisherseitkadyr/braille-library/internal/auth"
	"github.com/alisherseitkadyr/braille-library/internal/config"
	"github.com/alisherseitkadyr/braille-library/internal/domain"
	"github.com/alisherseitkadyr/braille-library/internal/repository/mongo"
)

const (
	randomSeed   = 42
	demoPassword = "Demo12345"
	insertBatch  = 10000
)

var (
	levels = []domain.Level{domain.LevelBeginner, domain.LevelIntermediate, domain.LevelAdvanced}
	now    = time.Now().UTC().Truncate(time.Second)
	f      = gofakeit.New(randomSeed)
	// The item texts use their own generator, so adding them did not change the rest of the data.
	contentRand = gofakeit.New(randomSeed + 1)
)

func main() {
	reset := flag.Bool("reset", false, "delete existing users, items, interactions and similarities first")
	ifEmpty := flag.Bool("if-empty", false, "do nothing (exit 0) if the database already has users; used by docker compose")
	userCount := flag.Int("users", 50, "number of users (including the two demo accounts)")
	itemCount := flag.Int("items", 200, "number of items")
	interactionCount := flag.Int("interactions", 0,
		"exact number of interactions for performance tests; 0 = demo data (35-85 per user).\n"+
			"With a value > 0, generated users get the emails user1@perf.kz, user2@perf.kz, ...")
	flag.Parse()
	if *userCount < 2 || *itemCount < 1 || *interactionCount < 0 {
		log.Fatal("need --users >= 2, --items >= 1 and --interactions >= 0")
	}
	started := time.Now()

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

	// The seed may run before the API ever started, so make sure the
	// unique indexes exist before inserting data.
	if err := mongo.EnsureIndexes(ctx, db); err != nil {
		log.Fatalf("indexes: %v", err)
	}

	names := []string{mongo.UsersCollection, mongo.ItemsCollection, mongo.InteractionsCollection}
	if *reset {
		// DeleteMany instead of Drop, so the indexes stay in place.
		// Old similarities point to deleted items, so they go too.
		for _, name := range append(names, mongo.ItemSimilaritiesCollection) {
			if _, err := db.Collection(name).DeleteMany(ctx, bson.M{}); err != nil {
				log.Fatalf("reset %s: %v", name, err)
			}
		}
	} else if n, err := db.Collection(mongo.UsersCollection).CountDocuments(ctx, bson.M{}); err != nil {
		log.Fatalf("count users: %v", err)
	} else if n > 0 {
		if *ifEmpty {
			fmt.Println("database already has data, seed skipped")
			return
		}
		log.Fatal("database is not empty, run with --reset to replace the data")
	}

	perf := *interactionCount > 0
	users := makeUsers(*userCount, perf)
	items, textErrors := makeItems(*itemCount)
	interactions := makeInteractions(users, items, textErrors, *interactionCount)

	insertAll(ctx, db.Collection(mongo.UsersCollection), users)
	insertAll(ctx, db.Collection(mongo.ItemsCollection), items)
	insertAll(ctx, db.Collection(mongo.InteractionsCollection), interactions)

	for _, name := range names {
		n, _ := db.Collection(name).CountDocuments(ctx, bson.M{})
		fmt.Printf("%-13s %d\n", name, n)
	}
	fmt.Printf("demo accounts: admin@demo.kz, reader@demo.kz (password %s)\n", demoPassword)
	if perf {
		fmt.Printf("generated users: user1@perf.kz ... user%d@perf.kz (same password)\n", len(users)-2)
	}
	fmt.Printf("done in %s\n", time.Since(started).Round(time.Millisecond))
}

// insertAll inserts documents in unordered bulk batches.
func insertAll[T any](ctx context.Context, coll *mongodriver.Collection, docs []T) {
	for start := 0; start < len(docs); start += insertBatch {
		batch := docs[start:min(start+insertBatch, len(docs))]
		if _, err := coll.InsertMany(ctx, batch, options.InsertMany().SetOrdered(false)); err != nil {
			log.Fatalf("insert into %s: %v", coll.Name(), err)
		}
	}
}

func makeUsers(count int, perf bool) []domain.User {
	// Same hashing as registration, so demo accounts can log in through the API.
	hash, err := auth.HashPassword(demoPassword)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	users := []domain.User{
		newUser("admin@demo.kz", "Demo Admin", domain.RoleAdmin, domain.LangKazakh, domain.LevelAdvanced, slices.Clone(domain.Categories), false),
		newUser("reader@demo.kz", "Demo Reader", domain.RoleReader, domain.LangKazakh, domain.LevelIntermediate,
			[]domain.Category{domain.CategoryFiction, domain.CategoryNews}, false),
	}

	people := append(slices.Clone(kkPeople), ruPeople...)
	f.ShuffleAnySlice(people)
	for i := 0; len(users) < count; i++ {
		p := people[i%len(people)] // names repeat for big data sets
		lang := domain.LangKazakh
		if slices.Contains(ruPeople, p) {
			lang = domain.LangRussian
		}
		role := pick(map[domain.Role]int{domain.RoleReader: 50, domain.RoleLearner: 35, domain.RoleTeacher: 15})

		var level domain.Level
		var interests []domain.Category
		switch role {
		case domain.RoleLearner:
			level = pick(map[domain.Level]int{domain.LevelBeginner: 75, domain.LevelIntermediate: 20, domain.LevelAdvanced: 5})
			interests = []domain.Category{pick(map[domain.Category]int{domain.CategoryExercises: 60, domain.CategoryChildren: 40})}
		case domain.RoleTeacher:
			level = pick(map[domain.Level]int{domain.LevelIntermediate: 40, domain.LevelAdvanced: 60})
			interests = []domain.Category{pick(map[domain.Category]int{domain.CategoryTextbook: 60, domain.CategoryExercises: 40})}
		default:
			level = pick(map[domain.Level]int{domain.LevelBeginner: 15, domain.LevelIntermediate: 45, domain.LevelAdvanced: 40})
		}
		target := f.Number(1, 3)
		for len(interests) < target {
			if c := randomOf(domain.Categories); !slices.Contains(interests, c) {
				interests = append(interests, c)
			}
		}

		email := fmt.Sprintf("%s@example.kz", p.login)
		if perf {
			email = fmt.Sprintf("user%d@perf.kz", len(users)-1)
		}
		users = append(users, newUser(email, p.name, role, lang, level, interests, f.Float64() < 0.3))
	}

	for i := range users {
		users[i].PasswordHash = hash
	}
	return users
}

func newUser(email, name string, role domain.Role, lang domain.Language, level domain.Level, interests []domain.Category, audio bool) domain.User {
	created := f.DateRange(now.AddDate(-1, 0, 0), now.AddDate(0, -2, 0))
	return domain.User{
		ID: bson.NewObjectID(), Email: email, Name: name, Role: role,
		PreferredLanguage: lang, ReadingLevel: level, Interests: interests, PrefersAudio: audio,
		CreatedAt: created, UpdatedAt: created,
	}
}

// makeItems also returns the recognition errors put into each item's content.
func makeItems(itemCount int) ([]domain.Item, map[bson.ObjectID][]textError) {
	textErrors := map[bson.ObjectID][]textError{}
	seen := map[string]bool{}
	items := make([]domain.Item, 0, itemCount)
	for len(items) < itemCount {
		category := randomOf(domain.Categories)
		lang := pick(map[domain.Language]int{domain.LangKazakh: 60, domain.LangRussian: 40})

		base := randomOf(titles[category][lang])
		title := base
		for part := 2; seen[title]; part++ {
			if lang == domain.LangKazakh {
				title = fmt.Sprintf("%s (%d-бөлім)", base, part)
			} else {
				title = fmt.Sprintf("%s (часть %d)", base, part)
			}
		}
		seen[title] = true

		var level domain.Level
		var pages int
		author := randomOf(editors[lang])
		switch category {
		case domain.CategoryFiction:
			level = pick(map[domain.Level]int{domain.LevelBeginner: 10, domain.LevelIntermediate: 45, domain.LevelAdvanced: 45})
			pages = f.Number(80, 400)
			author = randomOf(authors[lang])
		case domain.CategoryChildren:
			level = pick(map[domain.Level]int{domain.LevelBeginner: 80, domain.LevelIntermediate: 20})
			pages = f.Number(5, 40)
			author = randomOf(authors[lang])
		case domain.CategoryTextbook:
			level = randomOf(levels)
			pages = f.Number(60, 250)
		case domain.CategoryNews:
			level = pick(map[domain.Level]int{domain.LevelBeginner: 20, domain.LevelIntermediate: 60, domain.LevelAdvanced: 20})
			pages = f.Number(2, 12)
		default:
			level = pick(map[domain.Level]int{domain.LevelBeginner: 55, domain.LevelIntermediate: 30, domain.LevelAdvanced: 15})
			pages = f.Number(10, 60)
		}

		// Grade 2 uses contractions, so it is more common at higher levels and harder to recognise.
		grade := 1
		gradeTwoChance := map[domain.Level]float64{domain.LevelBeginner: 0.15, domain.LevelIntermediate: 0.5, domain.LevelAdvanced: 0.8}
		if f.Float64() < gradeTwoChance[level] {
			grade = 2
		}
		accuracy := f.Float64Range(0.88, 0.99)
		if grade == 2 {
			accuracy -= f.Float64Range(0.02, 0.06)
		}

		tags := slices.Clone(tagsByCategory[category][lang])
		f.ShuffleAnySlice(tags)

		var desc string
		if lang == domain.LangKazakh {
			desc = fmt.Sprintf("%s. Брайль жазуынан цифрландырылған мәтін, %d бет.", categoryName[category][lang], pages)
		} else {
			desc = fmt.Sprintf("%s. Текст, оцифрованный со шрифта Брайля, %d стр.", categoryName[category][lang], pages)
		}

		audioChance := 0.3
		if category == domain.CategoryChildren {
			audioChance = 0.55
		}

		created := f.DateRange(now.AddDate(-1, 0, 0), now.AddDate(0, -1, 0))
		id := bson.NewObjectID()
		accuracy = math.Round(accuracy*1000) / 1000
		content, errs := makeContent(contentRand, category, lang, accuracy)
		textErrors[id] = errs
		items = append(items, domain.Item{
			ID: id, Title: title, Description: desc, Category: category,
			Level: level, Language: lang, BrailleGrade: grade, Pages: pages, Author: author,
			Tags: tags[:f.Number(2, 3)], HasAudio: f.Float64() < audioChance,
			RecognitionAccuracy: accuracy, Content: content,
			CreatedAt: created, UpdatedAt: created,
		})
	}
	return items, textErrors
}

// makeInteractions simulates reading sessions. Each user picks items with weights based on
// their interests, language and reading level, then views, reads, listens, rates, etc.
//
// total = 0 gives the demo data (35-85 interactions per user). total > 0 gives exactly
// that many interactions, about total/users per user, for performance tests.
func makeInteractions(users []domain.User, items []domain.Item, textErrors map[bson.ObjectID][]textError, total int) []domain.Interaction {
	result := make([]domain.Interaction, 0, total)
	ratings := map[bson.ObjectID][]int{}
	// One rating and one favorite per user and item (unique index), across all passes.
	type pair struct{ user, item bson.ObjectID }
	rated := map[pair]bool{}
	favorited := map[pair]bool{}
	// Users with the same profile share the same cumulative item weights.
	weightsByProfile := map[string][]float64{}
	avg := max(1, total/len(users))

	for {
		for _, u := range users {
			budget := f.Number(35, 85)
			if u.Role == domain.RoleAdmin {
				budget = 5
			}
			if total > 0 {
				budget = f.Number(max(1, avg/2), avg*3/2)
			}
			profile := fmt.Sprint(u.Role, u.PreferredLanguage, u.ReadingLevel, u.Interests, u.PrefersAudio)
			weights, ok := weightsByProfile[profile]
			if !ok {
				weights = cumulativeWeights(itemWeights(u, items))
				weightsByProfile[profile] = weights
			}

			count := 0
			for count < budget && (total == 0 || len(result) < total) {
				it := items[weightedIndex(weights)]
				key := pair{u.ID, it.ID}
				start := it.CreatedAt
				if u.CreatedAt.After(start) {
					start = u.CreatedAt
				}
				t := f.DateRange(start, now)
				// add returns false once the requested total is reached.
				add := func(typ domain.InteractionType, value *int, c *domain.Correction) bool {
					if total > 0 && len(result) >= total {
						return false
					}
					result = append(result, domain.Interaction{
						ID: bson.NewObjectID(), UserID: u.ID, ItemID: it.ID,
						Type: typ, Value: value, Correction: c, CreatedAt: t,
					})
					t = t.Add(time.Duration(f.Number(1, 40)) * time.Minute)
					count++
					return true
				}

				add(domain.InteractionView, nil, nil)

				progress := 0
				if f.Float64() < 0.6 {
					progress = 100
					if f.Float64() < 0.6 {
						progress = f.Number(5, 95)
					}
					add(domain.InteractionRead, intPtr(progress), nil)
				}

				listenChance := 0.1
				if u.PrefersAudio {
					listenChance = 0.6
				}
				if it.HasAudio && f.Float64() < listenChance {
					add(domain.InteractionListen, nil, nil)
				}

				if progress >= 50 && !rated[key] && f.Float64() < 0.4 {
					r := rating(u, it)
					if add(domain.InteractionRate, intPtr(r), nil) {
						rated[key] = true
						ratings[it.ID] = append(ratings[it.ID], r)
					}
					if r >= 4 && !favorited[key] && f.Float64() < 0.35 && add(domain.InteractionFavorite, nil, nil) {
						favorited[key] = true
					}
				}

				// Lower recognition accuracy means more errors for readers to correct.
				// A correction points at one of the errors in the item's text.
				if errs := textErrors[it.ID]; progress > 0 && len(errs) > 0 && f.Float64() < (1-it.RecognitionAccuracy)*3 {
					e := errs[f.IntN(len(errs))]
					add(domain.InteractionCorrection, nil, &domain.Correction{
						Position: e.Position, Original: e.Wrong, Suggested: e.Correct,
					})
				}
			}
		}
		// The demo data is one pass; a big data set loops until the total is reached.
		if total == 0 || len(result) >= total {
			break
		}
	}

	for i := range items {
		if rs := ratings[items[i].ID]; len(rs) > 0 {
			sum := 0
			for _, r := range rs {
				sum += r
			}
			items[i].RatingsCount = len(rs)
			items[i].AvgRating = math.Round(float64(sum)/float64(len(rs))*100) / 100
		}
	}
	return result
}

func itemWeights(u domain.User, items []domain.Item) []float64 {
	learnerLevelWeight := map[domain.Level]float64{domain.LevelBeginner: 6, domain.LevelIntermediate: 1, domain.LevelAdvanced: 0.2}
	weights := make([]float64, len(items))
	for i, it := range items {
		w := 1.0
		if slices.Contains(u.Interests, it.Category) {
			w *= 8
		}
		if it.Language == u.PreferredLanguage {
			w *= 3
		}
		if u.Role == domain.RoleLearner {
			w *= learnerLevelWeight[it.Level]
		} else if it.Level == u.ReadingLevel {
			w *= 3
		}
		if u.PrefersAudio && it.HasAudio {
			w *= 2
		}
		weights[i] = w
	}
	return weights
}

// rating is higher for items that match the user and were recognised well.
func rating(u domain.User, it domain.Item) int {
	score := 3 + (it.RecognitionAccuracy-0.9)*15 + f.Float64Range(-1.2, 1.2)
	if slices.Contains(u.Interests, it.Category) {
		score += 0.5
	}
	if it.Level == u.ReadingLevel {
		score += 0.3
	}
	return max(1, min(5, int(math.Round(score))))
}

func cumulativeWeights(weights []float64) []float64 {
	cum := make([]float64, len(weights))
	sum := 0.0
	for i, w := range weights {
		sum += w
		cum[i] = sum
	}
	return cum
}

// weightedIndex picks an index with a binary search over cumulative weights,
// so a pick is O(log n) even with thousands of items.
func weightedIndex(cum []float64) int {
	r := f.Float64() * cum[len(cum)-1]
	return min(sort.Search(len(cum), func(i int) bool { return cum[i] > r }), len(cum)-1)
}

// pick returns a key chosen with the given percentage weights, in a stable order.
func pick[T ~string](weights map[T]int) T {
	keys := make([]T, 0, len(weights))
	for k := range weights {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	r := f.IntN(100)
	for _, k := range keys {
		if r -= weights[k]; r < 0 {
			return k
		}
	}
	return keys[len(keys)-1]
}

func randomOf[T any](values []T) T {
	return values[f.IntN(len(values))]
}

func intPtr(v int) *int { return &v }
