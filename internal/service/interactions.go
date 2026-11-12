package service

import (
	"context"
	"math"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/alisherseitkadyr/braille-library/internal/domain"
)

type InteractionRepository interface {
	Insert(ctx context.Context, in *domain.Interaction) error
	Upsert(ctx context.Context, in *domain.Interaction, overwrite bool) (bool, error)
	RatingStats(ctx context.Context, itemID bson.ObjectID) (float64, int, error)
	History(ctx context.Context, userID bson.ObjectID, t domain.InteractionType, page, limit int) ([]domain.HistoryEntry, int64, error)
}

type RatingUpdater interface {
	GetByID(ctx context.Context, id bson.ObjectID) (*domain.Item, error)
	SetRating(ctx context.Context, id bson.ObjectID, avg float64, count int) error
}

type InteractionService struct {
	interactions InteractionRepository
	items        RatingUpdater
	onRecord     func(userID string)
}

// NewInteractionService takes onRecord, called after each recorded interaction
// (used to drop the user's cached recommendations); it may be nil.
func NewInteractionService(interactions InteractionRepository, items RatingUpdater, onRecord func(userID string)) *InteractionService {
	return &InteractionService{interactions: interactions, items: items, onRecord: onRecord}
}

type InteractionInput struct {
	ItemID     string                 `json:"itemId"`
	Type       domain.InteractionType `json:"type"`
	Value      *int                   `json:"value"`
	Correction *domain.Correction     `json:"correction"`
}

type HistoryPage struct {
	Items []domain.HistoryEntry `json:"items"`
	Page  int                   `json:"page"`
	Limit int                   `json:"limit"`
	Total int64                 `json:"total"`
}

// ValidateInteraction checks value and correction against the type:
// read needs progress 0-100, rate needs 1-5, correction needs a correction,
// all other types must have neither.
func ValidateInteraction(in InteractionInput) error {
	if !in.Type.Valid() {
		return domain.Invalid("type must be view, read, listen, rate, favorite or correction")
	}

	switch in.Type {
	case domain.InteractionRead:
		if in.Value == nil || *in.Value < 0 || *in.Value > 100 {
			return domain.Invalid("read needs value (progress) from 0 to 100")
		}
	case domain.InteractionRate:
		if in.Value == nil || *in.Value < 1 || *in.Value > 5 {
			return domain.Invalid("rate needs value from 1 to 5")
		}
	default:
		if in.Value != nil {
			return domain.Invalid(string(in.Type) + " must not have a value")
		}
	}

	if in.Type != domain.InteractionCorrection {
		if in.Correction != nil {
			return domain.Invalid("only correction may have a correction field")
		}
		return nil
	}

	c := in.Correction
	switch {
	case c == nil:
		return domain.Invalid("correction needs a correction field")
	case c.Position < 0:
		return domain.Invalid("correction.position must be 0 or more")
	case strings.TrimSpace(c.Original) == "":
		return domain.Invalid("correction.original is required")
	case strings.TrimSpace(c.Suggested) == "":
		return domain.Invalid("correction.suggested is required")
	case c.Original == c.Suggested:
		return domain.Invalid("correction.suggested must differ from original")
	}
	return nil
}

// Record saves an interaction. It returns true if a new document was created
// (false when an existing rating was updated or a favorite already existed).
func (s *InteractionService) Record(ctx context.Context, userID string, in InteractionInput) (*domain.Interaction, bool, error) {
	it, created, err := s.record(ctx, userID, in)
	if err == nil && s.onRecord != nil {
		s.onRecord(userID)
	}
	return it, created, err
}

func (s *InteractionService) record(ctx context.Context, userID string, in InteractionInput) (*domain.Interaction, bool, error) {
	if err := ValidateInteraction(in); err != nil {
		return nil, false, err
	}
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, false, domain.ErrNotFound
	}
	itemID, err := bson.ObjectIDFromHex(in.ItemID)
	if err != nil {
		return nil, false, domain.Invalid("itemId is not a valid id")
	}
	if _, err := s.items.GetByID(ctx, itemID); err != nil {
		return nil, false, err
	}

	it := &domain.Interaction{
		UserID:     uid,
		ItemID:     itemID,
		Type:       in.Type,
		Value:      in.Value,
		Correction: in.Correction,
		CreatedAt:  time.Now().UTC(),
	}

	switch in.Type {
	case domain.InteractionRate:
		created, err := s.interactions.Upsert(ctx, it, true)
		if err != nil {
			return nil, false, err
		}
		if err := s.updateItemRating(ctx, itemID); err != nil {
			return nil, false, err
		}
		return it, created, nil
	case domain.InteractionFavorite:
		created, err := s.interactions.Upsert(ctx, it, false)
		return it, created, err
	default:
		err := s.interactions.Insert(ctx, it)
		return it, true, err
	}
}

func (s *InteractionService) History(ctx context.Context, userID string, t domain.InteractionType, page, limit int) (*HistoryPage, error) {
	if t != "" && !t.Valid() {
		return nil, domain.Invalid("unknown interaction type")
	}
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	entries, total, err := s.interactions.History(ctx, uid, t, page, limit)
	if err != nil {
		return nil, err
	}
	return &HistoryPage{Items: entries, Page: page, Limit: limit, Total: total}, nil
}

func (s *InteractionService) updateItemRating(ctx context.Context, itemID bson.ObjectID) error {
	avg, count, err := s.interactions.RatingStats(ctx, itemID)
	if err != nil {
		return err
	}
	return s.items.SetRating(ctx, itemID, math.Round(avg*100)/100, count)
}
