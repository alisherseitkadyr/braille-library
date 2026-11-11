package service

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/alisherseitkadyr/braille-library/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id bson.ObjectID) (*domain.User, error)
	Update(ctx context.Context, u *domain.User) error
}

type UserService struct {
	users UserRepository
}

func NewUserService(users UserRepository) *UserService {
	return &UserService{users: users}
}

// UpdateProfileInput holds the fields a user may change. Nil means "keep".
type UpdateProfileInput struct {
	Name              *string            `json:"name"`
	PreferredLanguage *domain.Language   `json:"preferredLanguage"`
	ReadingLevel      *domain.Level      `json:"readingLevel"`
	Interests         *[]domain.Category `json:"interests"`
	PrefersAudio      *bool              `json:"prefersAudio"`
}

func (s *UserService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return s.users.GetByID(ctx, oid)
}

func (s *UserService) UpdateProfile(ctx context.Context, id string, in UpdateProfileInput) (*domain.User, error) {
	u, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, domain.Invalid("name must not be empty")
		}
		u.Name = name
	}
	if in.PreferredLanguage != nil {
		if !in.PreferredLanguage.Valid() {
			return nil, domain.Invalid("preferredLanguage must be kk or ru")
		}
		u.PreferredLanguage = *in.PreferredLanguage
	}
	if in.ReadingLevel != nil {
		if !in.ReadingLevel.Valid() {
			return nil, domain.Invalid("readingLevel must be beginner, intermediate or advanced")
		}
		u.ReadingLevel = *in.ReadingLevel
	}
	if in.Interests != nil {
		if err := validateInterests(*in.Interests); err != nil {
			return nil, err
		}
		u.Interests = *in.Interests
	}
	if in.PrefersAudio != nil {
		u.PrefersAudio = *in.PrefersAudio
	}

	u.UpdatedAt = time.Now().UTC()
	if err := s.users.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func validateInterests(interests []domain.Category) error {
	for _, c := range interests {
		if !c.Valid() {
			return domain.Invalid("unknown interest category: " + string(c))
		}
	}
	return nil
}
