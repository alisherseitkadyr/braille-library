package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/alisherseitkadyr/braille-library/internal/domain"
)

type ItemRepository struct {
	coll *mongo.Collection
}

func NewItemRepository(db *mongo.Database) *ItemRepository {
	return &ItemRepository{coll: db.Collection(ItemsCollection)}
}

func (r *ItemRepository) Create(ctx context.Context, item *domain.Item) error {
	res, err := r.coll.InsertOne(ctx, item)
	if err != nil {
		return err
	}
	item.ID = res.InsertedID.(bson.ObjectID)
	return nil
}

func (r *ItemRepository) GetByID(ctx context.Context, id bson.ObjectID) (*domain.Item, error) {
	var item domain.Item
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&item)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ItemRepository) Update(ctx context.Context, item *domain.Item) error {
	res, err := r.coll.ReplaceOne(ctx, bson.M{"_id": item.ID}, item)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ItemRepository) SetRating(ctx context.Context, id bson.ObjectID, avg float64, count int) error {
	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id},
		bson.M{"$set": bson.M{"avgRating": avg, "ratingsCount": count}})
	return err
}

func (r *ItemRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	res, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ItemRepository) List(ctx context.Context, f domain.ItemFilter) ([]domain.Item, int64, error) {
	filter := bson.M{}
	if f.Query != "" {
		filter["$text"] = bson.M{"$search": f.Query}
	}
	if f.Category != "" {
		filter["category"] = f.Category
	}
	if f.Level != "" {
		filter["level"] = f.Level
	}
	if f.Language != "" {
		filter["language"] = f.Language
	}

	var sort bson.D
	switch {
	case f.Query != "":
		sort = bson.D{{Key: "score", Value: bson.M{"$meta": "textScore"}}, {Key: "_id", Value: 1}}
	case f.Sort == domain.SortRating:
		sort = bson.D{{Key: "avgRating", Value: -1}, {Key: "ratingsCount", Value: -1}, {Key: "_id", Value: 1}}
	default:
		sort = bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}
	}

	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSort(sort).
		SetSkip(int64((f.Page - 1) * f.Limit)).
		SetLimit(int64(f.Limit))
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	items := []domain.Item{}
	if err := cur.All(ctx, &items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetMany returns the items with the given ids, in any order. Missing ids are skipped.
func (r *ItemRepository) GetMany(ctx context.Context, ids []bson.ObjectID) ([]domain.Item, error) {
	items := []domain.Item{}
	if len(ids) == 0 {
		return items, nil
	}
	cur, err := r.coll.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	if err := cur.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// Popular returns items sorted by popularity (number of ratings, then
// average rating), skipping the excluded ids. Empty filter fields mean "any".
func (r *ItemRepository) Popular(ctx context.Context, f domain.PopularFilter, exclude []bson.ObjectID, limit int) ([]domain.Item, error) {
	filter, sort := PopularQuery(f, exclude)
	opts := options.Find().SetSort(sort).SetLimit(int64(limit))
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	items := []domain.Item{}
	if err := cur.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// PopularQuery builds the filter and sort of Popular.
func PopularQuery(f domain.PopularFilter, exclude []bson.ObjectID) (bson.M, bson.D) {
	filter := bson.M{}
	if len(exclude) > 0 {
		filter["_id"] = bson.M{"$nin": exclude}
	}
	if len(f.Categories) > 0 {
		filter["category"] = bson.M{"$in": f.Categories}
	}
	if f.Level != "" {
		filter["level"] = f.Level
	}
	if f.Language != "" {
		filter["language"] = f.Language
	}
	return filter, bson.D{{Key: "ratingsCount", Value: -1}, {Key: "avgRating", Value: -1}, {Key: "_id", Value: 1}}
}
