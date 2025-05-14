package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Ceesaxp/cocktail-bot/internal/domain"
	"github.com/Ceesaxp/cocktail-bot/internal/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoRepository implements a MongoDB-backed repository
type MongoRepository struct {
	client     *mongo.Client
	collection *mongo.Collection
	logger     *logger.Logger
}

// MongoUser is a MongoDB representation of User
type MongoUser struct {
	ID              string     `bson:"_id"`
	Email           string     `bson:"email"`
	DateAdded       time.Time  `bson:"date_added"`
	AlreadyConsumed *time.Time `bson:"already_consumed,omitempty"`
}

// NewMongoRepository creates a new MongoDB repository
func NewMongoRepository(ctx context.Context, connStr string, logger *logger.Logger) (*MongoRepository, error) {
	// Set client options
	clientOptions := options.Client().ApplyURI(connStr)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Check the connection
	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(ctxPing, readpref.Primary()); err != nil {
		client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// Get database and collection names from connection string
	parts := strings.Split(connStr, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid MongoDB connection string")
	}

	dbName := strings.Split(parts[len(parts)-1], "?")[0]
	if dbName == "" {
		dbName = "cocktail_bot"
	}

	// Get collection
	collection := client.Database(dbName).Collection("users")

	// Create indexes
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	ctxIndex, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, err = collection.Indexes().CreateOne(ctxIndex, indexModel)
	if err != nil {
		client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	return &MongoRepository{
		client:     client,
		collection: collection,
		logger:     logger,
	}, nil
}

// FindByEmail finds a user by email (case-insensitive)
func (r *MongoRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	// Normalize email
	email = strings.ToLower(email)

	// Create filter
	filter := bson.M{"email": bson.M{"$regex": fmt.Sprintf("^%s$", email), "$options": "i"}}

	// Use context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Find user
	var mongoUser MongoUser
	err := r.collection.FindOne(ctxWithTimeout, filter).Decode(&mongoUser)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrUserNotFound
		}

		// Check for connection issues
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "connection") {
			r.logger.Error("Database connection issue", "error", err)
			return nil, domain.ErrDatabaseUnavailable
		}

		r.logger.Error("Error finding user", "email", email, "error", err)
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Convert to domain user
	user := &domain.User{
		ID:              mongoUser.ID,
		Email:           mongoUser.Email,
		DateAdded:       mongoUser.DateAdded,
		AlreadyConsumed: mongoUser.AlreadyConsumed,
	}

	return user, nil
}

// UpdateUser updates a user in the repository
func (r *MongoRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	// Normalize email
	email := strings.ToLower(user.Email)

	// Create filter
	filter := bson.M{"email": bson.M{"$regex": fmt.Sprintf("^%s$", email), "$options": "i"}}

	// Create update
	update := bson.M{"$set": bson.M{"already_consumed": user.AlreadyConsumed}}

	// Use context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Update user
	result, err := r.collection.UpdateOne(ctxWithTimeout, filter, update)
	if err != nil {
		// Check for connection issues
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "connection") {
			r.logger.Error("Database connection issue", "error", err)
			return domain.ErrDatabaseUnavailable
		}

		r.logger.Error("Error updating user", "email", email, "error", err)
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Check if user was found
	if result.MatchedCount == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

// Close closes the repository
func (r *MongoRepository) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.client.Disconnect(ctx)
}
