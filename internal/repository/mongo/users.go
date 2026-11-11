package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/alisherseitkadyr/braille-library/internal/domain"
)

type UserRepository struct {
	coll *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{coll: db.Collection(UsersCollection)}
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	res, err := r.coll.InsertOne(ctx, u)
	if mongo.IsDuplicateKeyError(err) {
		return domain.ErrConflict
	}
	if err != nil {
		return err
	}
	u.ID = res.InsertedID.(bson.ObjectID)
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.findOne(ctx, bson.M{"email": email})
}

func (r *UserRepository) GetByID(ctx context.Context, id bson.ObjectID) (*domain.User, error) {
	return r.findOne(ctx, bson.M{"_id": id})
}

func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	res, err := r.coll.ReplaceOne(ctx, bson.M{"_id": u.ID}, u)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepository) findOne(ctx context.Context, filter bson.M) (*domain.User, error) {
	var u domain.User
	err := r.coll.FindOne(ctx, filter).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
