package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"slices"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/alisherseitkadyr/braille-library/internal/auth"
	"github.com/alisherseitkadyr/braille-library/internal/config"
	"github.com/alisherseitkadyr/braille-library/internal/domain"
	"github.com/alisherseitkadyr/braille-library/internal/repository/mongo"
)

const (
	randomSeed   = 42
	demoPassword = "Demo12345"
	userCount    = 50
	itemCount    = 200
)

var (
	levels = []domain.Level{domain.LevelBeginner, domain.LevelIntermediate, domain.LevelAdvanced}
	now    = time.Now().UTC().Truncate(time.Second)
	f      = gofakeit.New(randomSeed)
)

func main() {
	reset := flag.Bool("reset", false, "delete existing users, items and interactions first")
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

	// The seed may run before the API ever started, so make sure the
	// unique indexes exist before inserting data.
	if err := mongo.EnsureIndexes(ctx, db); err != nil {
		log.Fatalf("indexes: %v", err)
	}

	names := []string{mongo.UsersCollection, mongo.ItemsCollection, mongo.InteractionsCollection}
	if *reset {
		// DeleteMany instead of Drop, so the indexes stay in place.
		for _, name := range names {
			if _, err := db.Collection(name).DeleteMany(ctx, bson.M{}); err != nil {
				log.Fatalf("reset %s: %v", name, err)
			}
		}
	} else if n, err := db.Collection(mongo.UsersCollection).CountDocuments(ctx, bson.M{}); err != nil {
		log.Fatalf("count users: %v", err)
	} else if n > 0 {
		log.Fatal("database is not empty, run with --reset to replace the data")
	}

	users := makeUsers()
	items := makeItems()
	interactions := makeInteractions(users, items)

	if _, err := db.Collection(mongo.UsersCollection).InsertMany(ctx, users); err != nil {
		log.Fatalf("insert users: %v", err)
	}
	if _, err := db.Collection(mongo.ItemsCollection).InsertMany(ctx, items); err != nil {
		log.Fatalf("insert items: %v", err)
	}
	if _, err := db.Collection(mongo.InteractionsCollection).InsertMany(ctx, interactions); err != nil {
		log.Fatalf("insert interactions: %v", err)
	}

	for _, name := range names {
		n, _ := db.Collection(name).CountDocuments(ctx, bson.M{})
		fmt.Printf("%-13s %d\n", name, n)
	}
	fmt.Printf("demo accounts: admin@demo.kz, reader@demo.kz (password %s)\n", demoPassword)
}

func makeUsers() []domain.User {
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
	for i := 0; len(users) < userCount; i++ {
		p := people[i]
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

func makeItems() []domain.Item {
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
		items = append(items, domain.Item{
			ID: bson.NewObjectID(), Title: title, Description: desc, Category: category,
			Level: level, Language: lang, BrailleGrade: grade, Pages: pages, Author: author,
			Tags: tags[:f.Number(2, 3)], HasAudio: f.Float64() < audioChance,
			RecognitionAccuracy: math.Round(accuracy*1000) / 1000,
			CreatedAt:           created, UpdatedAt: created,
		})
	}
	return items
}

// makeInteractions simulates reading sessions. Each user picks items with weights based on
// their interests, language and reading level, then views, reads, listens, rates, etc.
func makeInteractions(users []domain.User, items []domain.Item) []domain.Interaction {
	var result []domain.Interaction
	ratings := map[bson.ObjectID][]int{}

	for _, u := range users {
		budget := f.Number(35, 85)
		if u.Role == domain.RoleAdmin {
			budget = 5
		}
		weights := itemWeights(u, items)
		rated := map[bson.ObjectID]bool{}
		favorited := map[bson.ObjectID]bool{}

		count := 0
		for count < budget {
			it := items[weightedIndex(weights)]
			start := it.CreatedAt
			if u.CreatedAt.After(start) {
				start = u.CreatedAt
			}
			t := f.DateRange(start, now)
			add := func(typ domain.InteractionType, value *int, c *domain.Correction) {
				result = append(result, domain.Interaction{
					ID: bson.NewObjectID(), UserID: u.ID, ItemID: it.ID,
					Type: typ, Value: value, Correction: c, CreatedAt: t,
				})
				t = t.Add(time.Duration(f.Number(1, 40)) * time.Minute)
				count++
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

			if progress >= 50 && !rated[it.ID] && f.Float64() < 0.4 {
				r := rating(u, it)
				rated[it.ID] = true
				ratings[it.ID] = append(ratings[it.ID], r)
				add(domain.InteractionRate, intPtr(r), nil)
				if r >= 4 && !favorited[it.ID] && f.Float64() < 0.35 {
					favorited[it.ID] = true
					add(domain.InteractionFavorite, nil, nil)
				}
			}

			// Lower recognition accuracy means more errors for readers to correct.
			if progress > 0 && f.Float64() < (1-it.RecognitionAccuracy)*3 {
				pairs := corrections[it.Language]
				c := pairs[f.IntN(len(pairs))]
				add(domain.InteractionCorrection, nil, &domain.Correction{
					Position: f.Number(0, it.Pages*1200), Original: c[0], Suggested: c[1],
				})
			}
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

func weightedIndex(weights []float64) int {
	total := 0.0
	for _, w := range weights {
		total += w
	}
	r := f.Float64() * total
	for i, w := range weights {
		if r -= w; r < 0 {
			return i
		}
	}
	return len(weights) - 1
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
