package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type InteractionType string

const (
	InteractionView       InteractionType = "view"
	InteractionRead       InteractionType = "read"
	InteractionListen     InteractionType = "listen"
	InteractionRate       InteractionType = "rate"
	InteractionFavorite   InteractionType = "favorite"
	InteractionCorrection InteractionType = "correction"
)

// Correction is a user fix of a recognition error; it becomes training data.
type Correction struct {
	Position  int    `bson:"position" json:"position"`
	Original  string `bson:"original" json:"original"`
	Suggested string `bson:"suggested" json:"suggested"`
}

// Interaction lives in its own collection: it grows without bound,
// so embedding it in users or items would hit the 16 MB document limit.
type Interaction struct {
	ID         bson.ObjectID   `bson:"_id,omitempty" json:"id"`
	UserID     bson.ObjectID   `bson:"userId" json:"userId"`
	ItemID     bson.ObjectID   `bson:"itemId" json:"itemId"`
	Type       InteractionType `bson:"type" json:"type"`
	Value      *int            `bson:"value,omitempty" json:"value,omitempty"`
	Correction *Correction     `bson:"correction,omitempty" json:"correction,omitempty"`
	CreatedAt  time.Time       `bson:"createdAt" json:"createdAt"`
}

// HistoryEntry is an interaction with the item title joined via $lookup.
// ItemTitle is empty if the item was deleted.
type HistoryEntry struct {
	Interaction `bson:",inline"`
	ItemTitle   string `bson:"itemTitle" json:"itemTitle"`
}

func (t InteractionType) Valid() bool {
	switch t {
	case InteractionView, InteractionRead, InteractionListen,
		InteractionRate, InteractionFavorite, InteractionCorrection:
		return true
	}
	return false
}
