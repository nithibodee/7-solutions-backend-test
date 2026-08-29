// Package mongo implements the domain user.Repository port on top of the
// official MongoDB Go driver.
package mongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	domain "github.com/nithibodee/7-solutions-backend-test/internal/domain/user"
)

const collectionName = "users"

// userDoc is the BSON representation of a user.
type userDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Name      string        `bson:"name"`
	Email     string        `bson:"email"`
	Password  string        `bson:"password"`
	CreatedAt time.Time     `bson:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at"`
}

func (d userDoc) toDomain() domain.User {
	return domain.User{
		ID:        d.ID.Hex(),
		Name:      d.Name,
		Email:     d.Email,
		Password:  d.Password,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

// UserRepository persists users in a MongoDB collection.
type UserRepository struct {
	coll *mongo.Collection
}

// NewUserRepository returns a repository bound to the "users" collection of db.
func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{coll: db.Collection(collectionName)}
}

// EnsureIndexes creates the unique index on email. Safe to call on every start.
func (r *UserRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("uniq_email"),
	})
	return err
}

// Create inserts a new user and back-fills its generated ID.
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	doc := userDoc{
		Name:      u.Name,
		Email:     u.Email,
		Password:  u.Password,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
	res, err := r.coll.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.ErrEmailAlreadyExists
		}
		return err
	}
	if oid, ok := res.InsertedID.(bson.ObjectID); ok {
		u.ID = oid.Hex()
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return r.findOne(ctx, bson.M{"_id": oid})
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.findOne(ctx, bson.M{"email": email})
}

func (r *UserRepository) findOne(ctx context.Context, filter bson.M) (*domain.User, error) {
	var doc userDoc
	if err := r.coll.FindOne(ctx, filter).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	u := doc.toDomain()
	return &u, nil
}

func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	cur, err := r.coll.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var docs []userDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	users := make([]domain.User, len(docs))
	for i, d := range docs {
		users[i] = d.toDomain()
	}
	return users, nil
}

func (r *UserRepository) Update(ctx context.Context, id string, fields domain.UpdateFields) (*domain.User, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	set := bson.M{"updated_at": time.Now().UTC()}
	if fields.Name != nil {
		set["name"] = *fields.Name
	}
	if fields.Email != nil {
		set["email"] = *fields.Email
	}

	var doc userDoc
	err = r.coll.FindOneAndUpdate(ctx, bson.M{"_id": oid}, bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		if mongo.IsDuplicateKeyError(err) {
			return nil, domain.ErrEmailAlreadyExists
		}
		return nil, err
	}
	u := doc.toDomain()
	return &u, nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return domain.ErrNotFound
	}
	res, err := r.coll.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	return r.coll.CountDocuments(ctx, bson.M{})
}
