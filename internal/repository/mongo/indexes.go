package mongo

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	UsersCollection        = "users"
	ItemsCollection        = "items"
	InteractionsCollection = "interactions"

	ItemSimilaritiesCollection = "item_similarities"
)

// EnsureIndexes creates all indexes. Creating an existing index with the
// same spec is a no-op, so this is safe to run at every startup.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	indexes := map[string][]mongo.IndexModel{
		UsersCollection: {
			{
				Keys:    bson.D{{Key: "email", Value: 1}},
				Options: options.Index().SetName("email_unique").SetUnique(true),
			},
		},
		ItemsCollection: {
			{
				Keys: bson.D{
					{Key: "title", Value: "text"},
					{Key: "description", Value: "text"},
					{Key: "tags", Value: "text"},
				},
				// MongoDB has no Kazakh stemming, so use "none" for all texts.
				// language_override points to a field we never use, because
				// items have their own "language" field with values like "kk".
				Options: options.Index().SetName("items_text").
					SetDefaultLanguage("none").
					SetLanguageOverride("textSearchLanguage"),
			},
			{
				Keys:    bson.D{{Key: "language", Value: 1}},
				Options: options.Index().SetName("language"),
			},
			{
				Keys:    bson.D{{Key: "avgRating", Value: -1}},
				Options: options.Index().SetName("avgRating_desc"),
			},
			// Popular items (recommendation fallback): the index order is the
			// sort order, so no in-memory SORT and the scan stops after "limit".
			{
				Keys: bson.D{
					{Key: "ratingsCount", Value: -1}, {Key: "avgRating", Value: -1}, {Key: "_id", Value: 1},
				},
				Options: options.Index().SetName("popularity"),
			},
			// Content-based fallback, ESR rule: equality fields (category $in,
			// level, language) first, then the sort fields. With several
			// categories MongoDB merges the sorted ranges (SORT_MERGE).
			{
				Keys: bson.D{
					{Key: "category", Value: 1}, {Key: "level", Value: 1}, {Key: "language", Value: 1},
					{Key: "ratingsCount", Value: -1}, {Key: "avgRating", Value: -1}, {Key: "_id", Value: 1},
				},
				Options: options.Index().SetName("profile_popularity"),
			},
		},
		InteractionsCollection: {
			{
				// History, newest first: _id is the tie-breaker of the sort, so it is
				// in the index too and the sort needs no in-memory SORT stage.
				Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}},
				Options: options.Index().SetName("user_createdAt_id"),
			},
			{
				Keys:    bson.D{{Key: "itemId", Value: 1}, {Key: "type", Value: 1}},
				Options: options.Index().SetName("item_type"),
			},
			{
				// User-item scores: the index holds every field the scores
				// aggregation reads, so it is a covered query (no FETCH).
				Keys: bson.D{
					{Key: "userId", Value: 1}, {Key: "itemId", Value: 1},
					{Key: "type", Value: 1}, {Key: "value", Value: 1}, {Key: "createdAt", Value: 1},
				},
				Options: options.Index().SetName("user_item_scores"),
			},
			{
				// One rating and one favorite per user per item.
				Keys: bson.D{{Key: "userId", Value: 1}, {Key: "itemId", Value: 1}, {Key: "type", Value: 1}},
				Options: options.Index().SetName("one_rate_favorite_per_user_item").
					SetUnique(true).
					SetPartialFilterExpression(bson.M{
						"type": bson.M{"$in": bson.A{"rate", "favorite"}},
					}),
			},
		},
		ItemSimilaritiesCollection: {
			{
				Keys:    bson.D{{Key: "itemId", Value: 1}},
				Options: options.Index().SetName("itemId_unique").SetUnique(true),
			},
		},
	}

	for coll, models := range indexes {
		if _, err := db.Collection(coll).Indexes().CreateMany(ctx, models); err != nil {
			return fmt.Errorf("create indexes on %s: %w", coll, err)
		}
	}

	// Indexes replaced after the index review: user_createdAt by user_createdAt_id,
	// category_level is a prefix of profile_popularity.
	obsolete := map[string][]string{
		ItemsCollection:        {"category_level"},
		InteractionsCollection: {"user_createdAt"},
	}
	for coll, names := range obsolete {
		for _, name := range names {
			err := db.Collection(coll).Indexes().DropOne(ctx, name)
			var cmdErr mongo.CommandError
			if err != nil && !(errors.As(err, &cmdErr) && cmdErr.Name == "IndexNotFound") {
				return fmt.Errorf("drop index %s on %s: %w", name, coll, err)
			}
		}
	}
	return nil
}
