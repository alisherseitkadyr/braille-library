package mongo

import (
	"context"
	"math"
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
	return r.scores(ctx, LikedScoresPipeline())
}

// UserScores returns s(u,i) of one user for every item the user interacted with.
func (r *RecommendationRepository) UserScores(ctx context.Context, userID bson.ObjectID) ([]recommend.UserItemScore, error) {
	return r.scores(ctx, UserScoresPipeline(userID))
}

// LikedScoresPipeline returns s(u,i) > 0 for all pairs (input of the rebuild job).
func LikedScoresPipeline() mongo.Pipeline {
	return append(scoresPipeline(bson.M{}),
		bson.D{{Key: "$match", Value: bson.M{"score": bson.M{"$gt": 0}}}})
}

// UserScoresPipeline returns s(u,i) of one user for every item.
func UserScoresPipeline(userID bson.ObjectID) mongo.Pipeline {
	return scoresPipeline(bson.M{"userId": userID})
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

// LiveNeighbors computes the neighbors of the given items on request, with no
// precomputed data: one aggregation for the co-occurrence of each item with
// all other items, one for the norms of all items involved.
func (r *RecommendationRepository) LiveNeighbors(ctx context.Context, itemIDs []bson.ObjectID) (map[bson.ObjectID][]recommend.Neighbor, error) {
	if len(itemIDs) == 0 {
		return map[bson.ObjectID][]recommend.Neighbor{}, nil
	}
	cur, err := r.interactions.Aggregate(ctx, CooccurrencePipeline(itemIDs))
	if err != nil {
		return nil, err
	}
	var pairs []recommend.PairStat
	if err := cur.All(ctx, &pairs); err != nil {
		return nil, err
	}

	involved := map[bson.ObjectID]bool{}
	for _, id := range itemIDs {
		involved[id] = true
	}
	for _, p := range pairs {
		involved[p.ItemJ] = true
	}
	ids := make([]bson.ObjectID, 0, len(involved))
	for id := range involved {
		ids = append(ids, id)
	}

	cur, err = r.interactions.Aggregate(ctx, NormsPipeline(ids))
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ItemID bson.ObjectID `bson:"_id"`
		SumSq  float64       `bson:"sumSq"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	norms := make(map[bson.ObjectID]float64, len(rows))
	for _, row := range rows {
		norms[row.ItemID] = math.Sqrt(row.SumSq)
	}
	return recommend.NeighborsFromStats(pairs, norms), nil
}

// likedScoresStages groups interactions by (userId, itemId) into
// {u, i, s} and keeps only liked pairs (s > 0).
func likedScoresStages() mongo.Pipeline {
	group := recommend.ScoreGroupFields()
	group["_id"] = bson.M{"u": "$userId", "i": "$itemId"}
	return mongo.Pipeline{
		{{Key: "$group", Value: group}},
		{{Key: "$project", Value: bson.M{"_id": 0, "u": "$_id.u", "i": "$_id.i", "s": recommend.ScoreExpr()}}},
		{{Key: "$match", Value: bson.M{"s": bson.M{"$gt": 0}}}},
	}
}

// CooccurrencePipeline returns, for each given item i and every other item j
// liked by the same users, {i, j, dot: sum s(u,i)*s(u,j), n: common users}.
func CooccurrencePipeline(itemIDs []bson.ObjectID) mongo.Pipeline {
	pipeline := mongo.Pipeline{{{Key: "$match", Value: bson.M{"itemId": bson.M{"$in": itemIDs}}}}}
	pipeline = append(pipeline, likedScoresStages()...)

	// All items liked by the same user, looked up once per user.
	otherItems := bson.A{}
	for _, stage := range likedScoresStages() {
		otherItems = append(otherItems, stage)
	}
	return append(pipeline,
		bson.D{{Key: "$group", Value: bson.M{"_id": "$u", "liked": bson.M{"$push": bson.M{"i": "$i", "s": "$s"}}}}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from": InteractionsCollection, "localField": "_id", "foreignField": "userId",
			"pipeline": otherItems, "as": "other",
		}}},
		bson.D{{Key: "$unwind", Value: "$liked"}},
		bson.D{{Key: "$unwind", Value: "$other"}},
		bson.D{{Key: "$match", Value: bson.M{"$expr": bson.M{"$ne": bson.A{"$liked.i", "$other.i"}}}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id": bson.M{"i": "$liked.i", "j": "$other.i"},
			"dot": bson.M{"$sum": bson.M{"$multiply": bson.A{"$liked.s", "$other.s"}}},
			"n":   bson.M{"$sum": 1},
		}}},
		bson.D{{Key: "$project", Value: bson.M{"_id": 0, "i": "$_id.i", "j": "$_id.j", "dot": 1, "n": 1}}},
	)
}

// NormsPipeline returns {_id: itemId, sumSq: sum of s(u,i)^2 over liking users}.
func NormsPipeline(itemIDs []bson.ObjectID) mongo.Pipeline {
	pipeline := mongo.Pipeline{{{Key: "$match", Value: bson.M{"itemId": bson.M{"$in": itemIDs}}}}}
	pipeline = append(pipeline, likedScoresStages()...)
	return append(pipeline, bson.D{{Key: "$group", Value: bson.M{
		"_id":   "$i",
		"sumSq": bson.M{"$sum": bson.M{"$multiply": bson.A{"$s", "$s"}}},
	}}})
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
