//go:build integration

// Package mongo integration tests run against a real MongoDB instance.
// Enable with: go test -tags=integration ./internal/adapter/mongo/...
// Set MONGO_TEST_URI to override the default connection string.
package mongo_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	mongoadapter "github.com/nithibodee/7-solutions-backend-test/internal/adapter/mongo"
	domain "github.com/nithibodee/7-solutions-backend-test/internal/domain/user"
)

func testDB(t *testing.T) *mongo.Database {
	t.Helper()
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err)
	require.NoError(t, client.Ping(ctx, readpref.Primary()))

	db := client.Database("usermgmt_it")
	t.Cleanup(func() {
		_ = db.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})
	return db
}

func TestUserRepository_CRUD(t *testing.T) {
	repo := mongoadapter.NewUserRepository(testDB(t))
	ctx := context.Background()
	require.NoError(t, repo.EnsureIndexes(ctx))

	now := time.Now().UTC().Truncate(time.Millisecond)
	u := &domain.User{Name: "Alice", Email: "alice@example.com", Password: "hash", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, repo.Create(ctx, u))
	require.NotEmpty(t, u.ID)

	got, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", got.Email)

	byEmail, err := repo.GetByEmail(ctx, "alice@example.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID, byEmail.ID)

	newName := "Alice Cooper"
	updated, err := repo.Update(ctx, u.ID, domain.UpdateFields{Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, "Alice Cooper", updated.Name)

	list, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	require.NoError(t, repo.Delete(ctx, u.ID))
	_, err = repo.GetByID(ctx, u.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserRepository_DuplicateEmail(t *testing.T) {
	repo := mongoadapter.NewUserRepository(testDB(t))
	ctx := context.Background()
	require.NoError(t, repo.EnsureIndexes(ctx))

	now := time.Now().UTC()
	require.NoError(t, repo.Create(ctx, &domain.User{Name: "A", Email: "dup@example.com", Password: "h", CreatedAt: now, UpdatedAt: now}))

	err := repo.Create(ctx, &domain.User{Name: "B", Email: "dup@example.com", Password: "h", CreatedAt: now, UpdatedAt: now})
	assert.ErrorIs(t, err, domain.ErrEmailAlreadyExists)
}

func TestUserRepository_GetMissing(t *testing.T) {
	repo := mongoadapter.NewUserRepository(testDB(t))
	_, err := repo.GetByID(context.Background(), "64b8f0c2f1a2b3c4d5e6f7a8")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
