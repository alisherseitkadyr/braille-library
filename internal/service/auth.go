package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/alisherseitkadyr/braille-library/internal/auth"
	"github.com/alisherseitkadyr/braille-library/internal/domain"
)

const minPasswordLength = 8

type AuthService struct {
	users  UserRepository
	tokens *auth.TokenManager
}

func NewAuthService(users UserRepository, tokens *auth.TokenManager) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

type RegisterInput struct {
	Email             string            `json:"email"`
	Password          string            `json:"password"`
	Name              string            `json:"name"`
	Role              domain.Role       `json:"role"`
	PreferredLanguage domain.Language   `json:"preferredLanguage"`
	ReadingLevel      domain.Level      `json:"readingLevel"`
	Interests         []domain.Category `json:"interests"`
	PrefersAudio      bool              `json:"prefersAudio"`
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*domain.User, error) {
	email := normalizeEmail(in.Email)
	if _, err := mail.ParseAddress(email); err != nil || !strings.Contains(email, "@") {
		return nil, domain.Invalid("invalid email")
	}
	if len(in.Password) < minPasswordLength {
		return nil, domain.Invalid("password must be at least 8 characters")
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, domain.Invalid("name is required")
	}

	// Self-registration can never create an admin.
	if in.Role == "" {
		in.Role = domain.RoleReader
	}
	if in.Role != domain.RoleReader && in.Role != domain.RoleLearner && in.Role != domain.RoleTeacher {
		return nil, domain.Invalid("role must be reader, learner or teacher")
	}
	if in.PreferredLanguage == "" {
		in.PreferredLanguage = domain.LangKazakh
	}
	if !in.PreferredLanguage.Valid() {
		return nil, domain.Invalid("preferredLanguage must be kk or ru")
	}
	if in.ReadingLevel == "" {
		in.ReadingLevel = domain.LevelBeginner
	}
	if !in.ReadingLevel.Valid() {
		return nil, domain.Invalid("readingLevel must be beginner, intermediate or advanced")
	}
	if in.Interests == nil {
		in.Interests = []domain.Category{}
	}
	if err := validateInterests(in.Interests); err != nil {
		return nil, err
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	u := &domain.User{
		Email:             email,
		PasswordHash:      hash,
		Name:              name,
		Role:              in.Role,
		PreferredLanguage: in.PreferredLanguage,
		ReadingLevel:      in.ReadingLevel,
		Interests:         in.Interests,
		PrefersAudio:      in.PrefersAudio,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// Login returns a JWT. Unknown email and wrong password give the same error.
func (s *AuthService) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	u, err := s.users.GetByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, domain.ErrNotFound) {
		return "", nil, domain.ErrInvalidCredentials
	}
	if err != nil {
		return "", nil, err
	}
	if !auth.CheckPassword(u.PasswordHash, password) {
		return "", nil, domain.ErrInvalidCredentials
	}

	token, err := s.tokens.Generate(u.ID.Hex(), u.Role)
	if err != nil {
		return "", nil, err
	}
	return token, u, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
