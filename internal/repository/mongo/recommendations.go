package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/alisherseitkadyr/braille-library/internal/recommend"
)

const similarityBatchSize = 1000

type RecommendationRepository struct {
	interactions *mongo.Collection
	similarities *mongo.Collection
}

func NewRecommendationRepository(db *mongo.Database) *RecommendationRepository {
	return &RecommendationRepository{
		interactions: db.Collection(InteractionsCollection),
		similarities: db.Collection(ItemSimilaritiesCollection),
	}
}

// LikedScores returns s(u,i) > 0 for all users and items.
func (r *RecommendationRepository) LikedScores(ctx context.Context) ([]recommend.UserItemScore, error) {
	pipeline := append(scoresPipeline(bson.M{}),
		bson.D{{Key: "$match", Value: bson.M{"score": bson.M{"$gt": 0}}}})
	return r.scores(ctx, pipeline)
}

// UserScores returns s(u,i) of one user for every item the user interacted with.
func (r *RecommendationRepository) UserScores(ctx context.Context, userID bson.ObjectID) ([]recommend.UserItemScore, error) {
	return r.scores(ctx, scoresPipeline(bson.M{"userId": userID}))
}

// scoresPipeline groups interactions by (user, item): score is the rating
// weight if the user rated the item, else the max weight of the other
// interactions; lastAt is the time of the last interaction.
func scoresPipeline(match bson.M) mongo.Pipeline {
	group := recommend.ScoreGroupFields()
	group["_id"] = bson.M{"userId": "$userId", "itemId": "$itemId"}
	group["lastAt"] = bson.M{"$max": "$createdAt"}
	return mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: group}},
		{{Key: "$project", Value: bson.M{
			"_id": 0, "userId": "$_id.userId", "itemId": "$_id.itemId",
			"score": recommend.ScoreExpr(), "lastAt": 1,
		}}},
	}
}

func (r *RecommendationRepository) scores(ctx context.Context, pipeline mongo.Pipeline) ([]recommend.UserItemScore, error) {
	cur, err := r.interactions.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result []recommend.UserItemScore
	if err := cur.All(ctx, &result); err != nil {
		return nil, err
	}
	return result, nil
}

type similarityDoc struct {
	ItemID    bson.ObjectID        `bson:"itemId"`
	Neighbors []recommend.Neighbor `bson:"neighbors"`
	UpdatedAt time.Time            `bson:"updatedAt"`
}

// SaveSimilarities upserts one document per item in bulk, then deletes the
// documents of items that got no neighbors in this run.
func (r *RecommendationRepository) SaveSimilarities(ctx context.Context, sims map[bson.ObjectID][]recommend.Neighbor, at time.Time) error {
	at = at.UTC().Truncate(time.Millisecond) // MongoDB stores milliseconds
	models := make([]mongo.WriteModel, 0, similarityBatchSize)
	flush := func() error {
		if len(models) == 0 {
			return nil
		}
		_, err := r.similarities.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
		models = models[:0]
		return err
	}

	for itemID, neighbors := range sims {
		models = append(models, mongo.NewReplaceOneModel().
			SetFilter(bson.M{"itemId": itemID}).
			SetReplacement(similarityDoc{ItemID: itemID, Neighbors: neighbors, UpdatedAt: at}).
			SetUpsert(true))
		if len(models) == similarityBatchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := flush(); err != nil {
		return err
	}
	_, err := r.similarities.DeleteMany(ctx, bson.M{"updatedAt": bson.M{"$lt": at}})
	return err
}

// Neighbors returns the stored neighbors of the given items.
func (r *RecommendationRepository) Neighbors(ctx context.Context, itemIDs []bson.ObjectID) (map[bson.ObjectID][]recommend.Neighbor, error) {
	if len(itemIDs) == 0 {
		return map[bson.ObjectID][]recommend.Neighbor{}, nil
	}
	cur, err := r.similarities.Find(ctx, bson.M{"itemId": bson.M{"$in": itemIDs}})
	if err != nil {
		return nil, err
	}
	var docs []similarityDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	result := make(map[bson.ObjectID][]recommend.Neighbor, len(docs))
	for _, d := range docs {
		result[d.ItemID] = d.Neighbors
	}
	return result, nil
}
