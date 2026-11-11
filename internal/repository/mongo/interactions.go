package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/alisherseitkadyr/braille-library/internal/domain"
)

type InteractionRepository struct {
	coll *mongo.Collection
}

func NewInteractionRepository(db *mongo.Database) *InteractionRepository {
	return &InteractionRepository{coll: db.Collection(InteractionsCollection)}
}

func (r *InteractionRepository) Insert(ctx context.Context, in *domain.Interaction) error {
	res, err := r.coll.InsertOne(ctx, in)
	if err != nil {
		return err
	}
	in.ID = res.InsertedID.(bson.ObjectID)
	return nil
}

// Upsert keeps one document per (userId, itemId, type), backed by the
// unique partial index. With overwrite=true (rate) value and createdAt
// are replaced; with overwrite=false (favorite) an existing document is
// left as is. It returns true if a new document was created.
func (r *InteractionRepository) Upsert(ctx context.Context, in *domain.Interaction, overwrite bool) (bool, error) {
	filter := bson.M{"userId": in.UserID, "itemId": in.ItemID, "type": in.Type}

	fields := bson.M{"createdAt": in.CreatedAt}
	if in.Value != nil {
		fields["value"] = *in.Value
	}
	update := bson.M{"$setOnInsert": fields}
	if overwrite {
		update = bson.M{"$set": fields}
	}

	res, err := r.coll.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return false, err
	}
	if err := r.coll.FindOne(ctx, filter).Decode(in); err != nil {
		return false, err
	}
	return res.UpsertedCount == 1, nil
}

// RatingStats returns the average and the number of ratings of an item.
func (r *InteractionRepository) RatingStats(ctx context.Context, itemID bson.ObjectID) (float64, int, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"itemId": itemID, "type": domain.InteractionRate}}},
		{{Key: "$group", Value: bson.M{
			"_id":   nil,
			"avg":   bson.M{"$avg": "$value"},
			"count": bson.M{"$sum": 1},
		}}},
	}
	cur, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, 0, err
	}
	var rows []struct {
		Avg   float64 `bson:"avg"`
		Count int     `bson:"count"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return 0, 0, err
	}
	if len(rows) == 0 {
		return 0, 0, nil
	}
	return rows[0].Avg, rows[0].Count, nil
}

// History returns a user's interactions, newest first, with item titles.
func (r *InteractionRepository) History(ctx context.Context, userID bson.ObjectID, t domain.InteractionType, page, limit int) ([]domain.HistoryEntry, int64, error) {
	match := bson.M{"userId": userID}
	if t != "" {
		match["type"] = t
	}

	total, err := r.coll.CountDocuments(ctx, match)
	if err != nil {
		return nil, 0, err
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$sort", Value: bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}}},
		{{Key: "$skip", Value: (page - 1) * limit}},
		{{Key: "$limit", Value: limit}},
		{{Key: "$lookup", Value: bson.M{
			"from":         ItemsCollection,
			"localField":   "itemId",
			"foreignField": "_id",
			"pipeline":     bson.A{bson.M{"$project": bson.M{"title": 1}}},
			"as":           "item",
		}}},
		{{Key: "$set", Value: bson.M{"itemTitle": bson.M{"$ifNull": bson.A{bson.M{"$first": "$item.title"}, ""}}}}},
		{{Key: "$unset", Value: "item"}},
	}
	cur, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	entries := []domain.HistoryEntry{}
	if err := cur.All(ctx, &entries); err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}
