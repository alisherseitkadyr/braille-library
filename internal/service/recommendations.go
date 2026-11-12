package service

import (
	"context"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/alisherseitkadyr/braille-library/internal/domain"
	"github.com/alisherseitkadyr/braille-library/internal/recommend"
)

const (
	ReasonInterests = "matches your interests"
	ReasonPopular   = "popular"
)

type RecommendationRepository interface {
	LikedScores(ctx context.Context) ([]recommend.UserItemScore, error)
	UserScores(ctx context.Context, userID bson.ObjectID) ([]recommend.UserItemScore, error)
	SaveSimilarities(ctx context.Context, sims map[bson.ObjectID][]recommend.Neighbor, at time.Time) error
	Neighbors(ctx context.Context, itemIDs []bson.ObjectID) (map[bson.ObjectID][]recommend.Neighbor, error)
}

type RecommendationItems interface {
	GetByID(ctx context.Context, id bson.ObjectID) (*domain.Item, error)
	GetMany(ctx context.Context, ids []bson.ObjectID) ([]domain.Item, error)
	Popular(ctx context.Context, f domain.PopularFilter, exclude []bson.ObjectID, limit int) ([]domain.Item, error)
}

type RecommendationService struct {
	recs  RecommendationRepository
	items RecommendationItems
	users UserRepository
}

func NewRecommendationService(recs RecommendationRepository, items RecommendationItems, users UserRepository) *RecommendationService {
	return &RecommendationService{recs: recs, items: items, users: users}
}

type Recommendation struct {
	Item   domain.Item `json:"item"`
	Score  float64     `json:"score" example:"2.415"`
	Reason string      `json:"reason" example:"because you read Абай жолы"`
}

type RecommendationList struct {
	Items []Recommendation `json:"items"`
}

type SimilarItem struct {
	Item  domain.Item `json:"item"`
	Score float64     `json:"score" example:"0.412"`
}

type SimilarList struct {
	Items []SimilarItem `json:"items"`
}

type RebuildResult struct {
	Items      int   `json:"items" example:"200"`
	DurationMs int64 `json:"durationMs" example:"85"`
}

// Rebuild recomputes item_similarities from all interactions.
// Items is the number of items that got at least one neighbor.
func (s *RecommendationService) Rebuild(ctx context.Context) (*RebuildResult, error) {
	start := time.Now()
	scores, err := s.recs.LikedScores(ctx)
	if err != nil {
		return nil, err
	}
	sims := recommend.Similarities(scores)
	if err := s.recs.SaveSimilarities(ctx, sims, start); err != nil {
		return nil, err
	}
	return &RebuildResult{Items: len(sims), DurationMs: time.Since(start).Milliseconds()}, nil
}

// ForUser returns up to limit recommendations: collaborative filtering
// first, then items matching the user's profile, then popular items.
// Items the user already interacted with are never returned.
func (s *RecommendationService) ForUser(ctx context.Context, userID string, limit int) (*RecommendationList, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	user, err := s.users.GetByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	history, err := s.recs.UserScores(ctx, uid)
	if err != nil {
		return nil, err
	}

	var likedIDs []bson.ObjectID
	for _, h := range recommend.LikedHistory(history) {
		likedIDs = append(likedIDs, h.ItemID)
	}
	neighbors, err := s.recs.Neighbors(ctx, likedIDs)
	if err != nil {
		return nil, err
	}
	candidates := recommend.Recommend(history, neighbors, limit)

	// Load the candidates and the history items named in the reasons.
	ids := make([]bson.ObjectID, 0, 2*len(candidates))
	for _, c := range candidates {
		ids = append(ids, c.ItemID, c.Because)
	}
	items, err := s.items.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[bson.ObjectID]domain.Item, len(items))
	for _, it := range items {
		byID[it.ID] = it
	}

	result := []Recommendation{}
	exclude := make([]bson.ObjectID, 0, len(history)+limit)
	for _, h := range history {
		exclude = append(exclude, h.ItemID)
	}
	for _, c := range candidates {
		item, ok := byID[c.ItemID]
		if !ok {
			continue // deleted since the last rebuild
		}
		reason := "similar to your history"
		if because, ok := byID[c.Because]; ok {
			reason = "because you read " + because.Title
		}
		result = append(result, Recommendation{Item: item, Score: round3(c.Score), Reason: reason})
		exclude = append(exclude, item.ID)
	}

	fill := func(f domain.PopularFilter, reason string) error {
		if len(result) >= limit {
			return nil
		}
		items, err := s.items.Popular(ctx, f, exclude, limit-len(result))
		if err != nil {
			return err
		}
		for _, it := range items {
			result = append(result, Recommendation{Item: it, Reason: reason})
			exclude = append(exclude, it.ID)
		}
		return nil
	}
	if len(user.Interests) > 0 {
		profile := domain.PopularFilter{Categories: user.Interests, Level: user.ReadingLevel, Language: user.PreferredLanguage}
		if err := fill(profile, ReasonInterests); err != nil {
			return nil, err
		}
	}
	if err := fill(domain.PopularFilter{}, ReasonPopular); err != nil {
		return nil, err
	}
	return &RecommendationList{Items: result}, nil
}

// Similar returns up to limit items most similar to the given one.
func (s *RecommendationService) Similar(ctx context.Context, itemID string, limit int) (*SimilarList, error) {
	oid, err := bson.ObjectIDFromHex(itemID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if _, err := s.items.GetByID(ctx, oid); err != nil {
		return nil, err
	}
	all, err := s.recs.Neighbors(ctx, []bson.ObjectID{oid})
	if err != nil {
		return nil, err
	}
	neighbors := all[oid]
	if len(neighbors) > limit {
		neighbors = neighbors[:limit]
	}

	ids := make([]bson.ObjectID, len(neighbors))
	for i, n := range neighbors {
		ids[i] = n.ItemID
	}
	items, err := s.items.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[bson.ObjectID]domain.Item, len(items))
	for _, it := range items {
		byID[it.ID] = it
	}

	result := []SimilarItem{}
	for _, n := range neighbors {
		if item, ok := byID[n.ItemID]; ok {
			result = append(result, SimilarItem{Item: item, Score: round3(n.Score)})
		}
	}
	return &SimilarList{Items: result}, nil
}

func round3(x float64) float64 {
	return math.Round(x*1000) / 1000
}
