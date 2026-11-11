package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	UsersCollection        = "users"
	ItemsCollection        = "items"
	InteractionsCollection = "interactions"
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
				Keys:    bson.D{{Key: "category", Value: 1}, {Key: "level", Value: 1}},
				Options: options.Index().SetName("category_level"),
			},
			{
				Keys:    bson.D{{Key: "language", Value: 1}},
				Options: options.Index().SetName("language"),
			},
			{
				Keys:    bson.D{{Key: "avgRating", Value: -1}},
				Options: options.Index().SetName("avgRating_desc"),
			},
		},
		InteractionsCollection: {
			{
				Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}},
				Options: options.Index().SetName("user_createdAt"),
			},
			{
				Keys:    bson.D{{Key: "itemId", Value: 1}, {Key: "type", Value: 1}},
				Options: options.Index().SetName("item_type"),
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
	}

	for coll, models := range indexes {
		if _, err := db.Collection(coll).Indexes().CreateMany(ctx, models); err != nil {
			return fmt.Errorf("create indexes on %s: %w", coll, err)
		}
	}
	return nil
}
