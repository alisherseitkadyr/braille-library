package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/alisherseitkadyr/braille-library/internal/domain"
)

type ItemRepository interface {
	Create(ctx context.Context, item *domain.Item) error
	GetByID(ctx context.Context, id bson.ObjectID) (*domain.Item, error)
	Update(ctx context.Context, item *domain.Item) error
	Delete(ctx context.Context, id bson.ObjectID) error
	List(ctx context.Context, f domain.ItemFilter) ([]domain.Item, int64, error)
}

type ItemService struct {
	items ItemRepository
}

func NewItemService(items ItemRepository) *ItemService {
	return &ItemService{items: items}
}

// ItemInput is used for both create and update (PUT replaces these fields).
// avgRating and ratingsCount are computed from ratings, so they are not here.
type ItemInput struct {
	Title               string          `json:"title"`
	Description         string          `json:"description"`
	Category            domain.Category `json:"category"`
	Level               domain.Level    `json:"level"`
	Language            domain.Language `json:"language"`
	BrailleGrade        int             `json:"brailleGrade"`
	Pages               int             `json:"pages"`
	Author              string          `json:"author"`
	Tags                []string        `json:"tags"`
	HasAudio            bool            `json:"hasAudio"`
	RecognitionAccuracy float64         `json:"recognitionAccuracy"`
	// Content is the recognized text, paragraphs separated by a blank line (optional).
	Content string `json:"content"`
}

type ItemPage struct {
	Items []domain.Item `json:"items"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
	Total int64         `json:"total"`
}

func (s *ItemService) Create(ctx context.Context, in ItemInput) (*domain.Item, error) {
	if err := validateItem(&in); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	item := &domain.Item{CreatedAt: now}
	applyItemInput(item, in, now)
	if err := s.items.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ItemService) Get(ctx context.Context, id string) (*domain.Item, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return s.items.GetByID(ctx, oid)
}

func (s *ItemService) Update(ctx context.Context, id string, in ItemInput) (*domain.Item, error) {
	if err := validateItem(&in); err != nil {
		return nil, err
	}
	item, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	applyItemInput(item, in, time.Now().UTC())
	if err := s.items.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ItemService) Delete(ctx context.Context, id string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return domain.ErrNotFound
	}
	return s.items.Delete(ctx, oid)
}

func (s *ItemService) List(ctx context.Context, f domain.ItemFilter) (*ItemPage, error) {
	f.Query = strings.TrimSpace(f.Query)
	if !utf8.ValidString(f.Query) {
		return nil, domain.Invalid("q must be valid UTF-8")
	}
	if f.Category != "" && !f.Category.Valid() {
		return nil, domain.Invalid("unknown category")
	}
	if f.Level != "" && !f.Level.Valid() {
		return nil, domain.Invalid("unknown level")
	}
	if f.Language != "" && !f.Language.Valid() {
		return nil, domain.Invalid("language must be kk or ru")
	}
	if f.Sort == "" {
		f.Sort = domain.SortNew
	}
	if f.Sort != domain.SortNew && f.Sort != domain.SortRating {
		return nil, domain.Invalid("sort must be new or rating")
	}

	items, total, err := s.items.List(ctx, f)
	if err != nil {
		return nil, err
	}
	return &ItemPage{Items: items, Page: f.Page, Limit: f.Limit, Total: total}, nil
}

func validateItem(in *ItemInput) error {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.Author = strings.TrimSpace(in.Author)
	in.Content = strings.TrimSpace(in.Content)

	tags := []string{}
	for _, t := range in.Tags {
		if t = strings.ToLower(strings.TrimSpace(t)); t != "" {
			tags = append(tags, t)
		}
	}
	in.Tags = tags

	switch {
	case in.Title == "":
		return domain.Invalid("title is required")
	case !in.Category.Valid():
		return domain.Invalid("category must be fiction, textbook, children, news or exercises")
	case !in.Level.Valid():
		return domain.Invalid("level must be beginner, intermediate or advanced")
	case !in.Language.Valid():
		return domain.Invalid("language must be kk or ru")
	case in.BrailleGrade != 1 && in.BrailleGrade != 2:
		return domain.Invalid("brailleGrade must be 1 or 2")
	case in.Pages < 1:
		return domain.Invalid("pages must be at least 1")
	case in.RecognitionAccuracy < 0 || in.RecognitionAccuracy > 1:
		return domain.Invalid("recognitionAccuracy must be between 0 and 1")
	}
	return nil
}

func applyItemInput(item *domain.Item, in ItemInput, now time.Time) {
	item.Title = in.Title
	item.Description = in.Description
	item.Category = in.Category
	item.Level = in.Level
	item.Language = in.Language
	item.BrailleGrade = in.BrailleGrade
	item.Pages = in.Pages
	item.Author = in.Author
	item.Tags = in.Tags
	item.HasAudio = in.HasAudio
	item.RecognitionAccuracy = in.RecognitionAccuracy
	item.Content = in.Content
	item.UpdatedAt = now
}
