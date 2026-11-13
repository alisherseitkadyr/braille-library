package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Category string

const (
	CategoryFiction   Category = "fiction"
	CategoryTextbook  Category = "textbook"
	CategoryChildren  Category = "children"
	CategoryNews      Category = "news"
	CategoryExercises Category = "exercises"
)

var Categories = []Category{CategoryFiction, CategoryTextbook, CategoryChildren, CategoryNews, CategoryExercises}

// Item is a Braille text digitized by the recognition model.
type Item struct {
	ID                  bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Title               string        `bson:"title" json:"title"`
	Description         string        `bson:"description" json:"description"`
	Category            Category      `bson:"category" json:"category"`
	Level               Level         `bson:"level" json:"level"`
	Language            Language      `bson:"language" json:"language"`
	BrailleGrade        int           `bson:"brailleGrade" json:"brailleGrade"`
	Pages               int           `bson:"pages" json:"pages"`
	Author              string        `bson:"author" json:"author"`
	Tags                []string      `bson:"tags" json:"tags"`
	HasAudio            bool          `bson:"hasAudio" json:"hasAudio"`
	RecognitionAccuracy float64       `bson:"recognitionAccuracy" json:"recognitionAccuracy"`
	// Content is the recognized text: paragraphs separated by a blank line.
	// It is returned by GET /items/{id} only; lists leave it out.
	Content      string    `bson:"content,omitempty" json:"content,omitempty" example:"Күз келіп, дала сарғая бастады..."`
	AvgRating    float64   `bson:"avgRating" json:"avgRating"`
	RatingsCount int       `bson:"ratingsCount" json:"ratingsCount"`
	CreatedAt    time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time `bson:"updatedAt" json:"updatedAt"`
}

const (
	SortNew    = "new"
	SortRating = "rating"
)

// ItemFilter describes a search in the catalog. Empty fields mean "any".
type ItemFilter struct {
	Query    string
	Category Category
	Level    Level
	Language Language
	Sort     string
	Page     int
	Limit    int
}

func (c Category) Valid() bool {
	for _, v := range Categories {
		if c == v {
			return true
		}
	}
	return false
}

// PopularFilter narrows a popularity query. Empty fields mean "any".
type PopularFilter struct {
	Categories []Category
	Level      Level
	Language   Language
}
