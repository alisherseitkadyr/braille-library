package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Role string

const (
	RoleReader  Role = "reader"
	RoleLearner Role = "learner"
	RoleTeacher Role = "teacher"
	RoleAdmin   Role = "admin"
)

type Language string

const (
	LangKazakh  Language = "kk"
	LangRussian Language = "ru"
)

type Level string

const (
	LevelBeginner     Level = "beginner"
	LevelIntermediate Level = "intermediate"
	LevelAdvanced     Level = "advanced"
)

// User stores only reading preferences, never disability or health information.
type User struct {
	ID                bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Email             string        `bson:"email" json:"email"`
	PasswordHash      string        `bson:"passwordHash" json:"-"`
	Name              string        `bson:"name" json:"name"`
	Role              Role          `bson:"role" json:"role"`
	PreferredLanguage Language      `bson:"preferredLanguage" json:"preferredLanguage"`
	ReadingLevel      Level         `bson:"readingLevel" json:"readingLevel"`
	Interests         []Category    `bson:"interests" json:"interests"`
	PrefersAudio      bool          `bson:"prefersAudio" json:"prefersAudio"`
	CreatedAt         time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time     `bson:"updatedAt" json:"updatedAt"`
}

func (r Role) Valid() bool {
	switch r {
	case RoleReader, RoleLearner, RoleTeacher, RoleAdmin:
		return true
	}
	return false
}

func (l Language) Valid() bool {
	return l == LangKazakh || l == LangRussian
}

func (l Level) Valid() bool {
	switch l {
	case LevelBeginner, LevelIntermediate, LevelAdvanced:
		return true
	}
	return false
}
