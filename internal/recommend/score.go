// Package recommend holds the item-based collaborative filtering algorithm.
// It has no database code, so it can be unit tested on plain data.
package recommend

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/alisherseitkadyr/braille-library/internal/domain"
)

// Weights of interactions in the user-item score.
const (
	ViewWeight         = 1.0
	ListenWeight       = 2.0
	CorrectionWeight   = 1.0
	ReadWeight         = 1.5
	ReadFinishedWeight = 3.0 // read with progress >= ReadFinishedProgress
	FavoriteWeight     = 4.0

	ReadFinishedProgress = 80
)

// RateWeights maps a rating 1-5 to its weight (index 0 is unused).
var RateWeights = [6]float64{0, -2, -1, 1, 3, 5}

// Weight returns the weight of one interaction.
func Weight(t domain.InteractionType, value *int) float64 {
	switch t {
	case domain.InteractionView:
		return ViewWeight
	case domain.InteractionListen:
		return ListenWeight
	case domain.InteractionCorrection:
		return CorrectionWeight
	case domain.InteractionFavorite:
		return FavoriteWeight
	case domain.InteractionRead:
		if value != nil && *value >= ReadFinishedProgress {
			return ReadFinishedWeight
		}
		return ReadWeight
	case domain.InteractionRate:
		if value != nil && *value >= 1 && *value <= 5 {
			return RateWeights[*value]
		}
	}
	return 0
}

// Score is s(u,i) for a user's interactions with one item. A rating is
// explicit feedback and overrides everything else: if the user rated the
// item, the score is the rating weight. Otherwise it is the maximum weight
// of the other interactions. Only a score > 0 means the user liked the item.
func Score(interactions []domain.Interaction) float64 {
	rated := false
	var rating, implicit float64
	for _, in := range interactions {
		w := Weight(in.Type, in.Value)
		if in.Type == domain.InteractionRate {
			rated, rating = true, w
		} else {
			implicit = max(implicit, w)
		}
	}
	if rated {
		return rating
	}
	return implicit
}

// ScoreGroupFields are $group accumulators that, together with ScoreExpr,
// compute the same s(u,i) as Score: "rating" is the rating weight (null if
// not rated, $max ignores nulls), "implicit" the max weight of the rest.
func ScoreGroupFields() bson.M {
	isRate := bson.M{"$eq": bson.A{"$type", domain.InteractionRate}}
	return bson.M{
		"rating":   bson.M{"$max": bson.M{"$cond": bson.A{isRate, WeightExpr(), nil}}},
		"implicit": bson.M{"$max": bson.M{"$cond": bson.A{isRate, nil, WeightExpr()}}},
	}
}

// ScoreExpr picks the rating if there is one, else the implicit score.
func ScoreExpr() bson.M {
	return bson.M{"$ifNull": bson.A{"$rating", "$implicit"}}
}

// WeightExpr is the same rule as Weight, written as a MongoDB aggregation
// expression.
func WeightExpr() bson.M {
	rate := bson.A{}
	for v := 1; v <= 5; v++ {
		rate = append(rate, bson.M{"case": bson.M{"$eq": bson.A{"$value", v}}, "then": RateWeights[v]})
	}
	return bson.M{"$switch": bson.M{
		"branches": bson.A{
			bson.M{"case": bson.M{"$eq": bson.A{"$type", domain.InteractionView}}, "then": ViewWeight},
			bson.M{"case": bson.M{"$eq": bson.A{"$type", domain.InteractionListen}}, "then": ListenWeight},
			bson.M{"case": bson.M{"$eq": bson.A{"$type", domain.InteractionCorrection}}, "then": CorrectionWeight},
			bson.M{"case": bson.M{"$eq": bson.A{"$type", domain.InteractionFavorite}}, "then": FavoriteWeight},
			bson.M{"case": bson.M{"$eq": bson.A{"$type", domain.InteractionRead}}, "then": bson.M{
				"$cond": bson.A{bson.M{"$gte": bson.A{"$value", ReadFinishedProgress}}, ReadFinishedWeight, ReadWeight},
			}},
			bson.M{"case": bson.M{"$eq": bson.A{"$type", domain.InteractionRate}}, "then": bson.M{
				"$switch": bson.M{"branches": rate, "default": 0},
			}},
		},
		"default": 0,
	}}
}
