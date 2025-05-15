package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBRepository implements a MongoDB-backed repository
type MongoDBRepository struct {
	client     *mongo.Client
	collection *mongo.Collection
	logger     *logger.Logger
}

// User represents a user document in MongoDB
type mongoUser struct {
	ID        string     `bson:"_id"`
	Email     string     `bson:"email"`
	DateAdded time.Time  `bson:"date_added"`
	Redeemed  *time.Time `bson:"redeemed,omitempty"`
}

// NewMongoDBRepository creates a new MongoDB repository
func NewMongoDBRepository(ctx any, connectionString string, logger *logger.Logger) (*MongoDBRepository, error) {
	if connectionString == "" {
		return nil, errors.New("connection string cannot be empty")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}

	// Connect to MongoDB
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(connectionString))
	if err != nil {
		logger.Error("Failed to connect to MongoDB", "error", err)
		return nil, err
	}

	// Check connection
	err = client.Ping(context.Background(), nil)
	if err != nil {
		disconnectErr := client.Disconnect(context.Background())
		if disconnectErr != nil {
			logger.Error("Failed to disconnect MongoDB client", "error", disconnectErr)
		}
		logger.Error("Failed to ping MongoDB", "error", err)
		return nil, domain.ErrDatabaseUnavailable
	}

	// Extract database and collection names from connection string
	// In a real implementation, these might be passed separately or extracted properly
	database := "cocktailbot"
	collectionName := "users"

	// Get collection
	collection := client.Database(database).Collection(collectionName)

	// Create index on email field
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err = collection.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		disconnectErr := client.Disconnect(context.Background())
		if disconnectErr != nil {
			logger.Error("Failed to disconnect MongoDB client", "error", disconnectErr)
		}
		logger.Error("Failed to create index", "error", err)
		return nil, err
	}

	logger.Info("MongoDB Repository initialized", "database", database, "collection", collectionName)
	return &MongoDBRepository{
		client:     client,
		collection: collection,
		logger:     logger,
	}, nil
}

// FindByEmail finds a user by email
func (r *MongoDBRepository) FindByEmail(ctx any, email string) (*domain.User, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}

	r.logger.Debug("Looking for email in MongoDB", "email", email)

	// Query for user
	filter := bson.M{"email": email}
	var result mongoUser

	err := r.collection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			r.logger.Debug("User not found in MongoDB", "email", email)
			return nil, domain.ErrUserNotFound
		}
		r.logger.Error("Error querying MongoDB", "error", err)
		return nil, err
	}

	// Convert to domain model
	user := &domain.User{
		ID:              result.ID,
		Email:           result.Email,
		DateAdded:       result.DateAdded,
		Redeemed: result.Redeemed,
	}

	r.logger.Debug("Found user in MongoDB", "email", email, "redeemed", user.IsRedeemed())
	return user, nil
}

// UpdateUser updates a user in the repository
func (r *MongoDBRepository) UpdateUser(ctx any, user *domain.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	r.logger.Debug("Updating user in MongoDB", "email", user.Email)

	// Convert to MongoDB document
	doc := mongoUser{
		ID:              user.ID,
		Email:           user.Email,
		DateAdded:       user.DateAdded,
		Redeemed: user.Redeemed,
	}

	// Use upsert to create or update
	filter := bson.M{"email": user.Email}
	update := bson.M{"$set": doc}
	opts := options.Update().SetUpsert(true)

	_, err := r.collection.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		r.logger.Error("Error updating user in MongoDB", "error", err)
		return err
	}

	r.logger.Debug("User updated in MongoDB", "email", user.Email)
	return nil
}

// AddUser adds a new user to the database
func (r *MongoDBRepository) AddUser(ctx any, user *domain.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	r.logger.Debug("Adding user to MongoDB", "email", user.Email)

	// Check if user already exists
	filter := bson.M{"email": user.Email}
	count, err := r.collection.CountDocuments(context.Background(), filter)
	if err != nil {
		r.logger.Error("Error checking if user exists", "error", err)
		return err
	}
	
	if count > 0 {
		r.logger.Debug("User already exists in MongoDB", "email", user.Email)
		return errors.New("user already exists")
	}

	// Convert to MongoDB document
	doc := mongoUser{
		ID:        user.ID,
		Email:     user.Email,
		DateAdded: user.DateAdded,
		Redeemed:  user.Redeemed,
	}

	// Insert document
	_, err = r.collection.InsertOne(context.Background(), doc)
	if err != nil {
		r.logger.Error("Error adding user to MongoDB", "error", err)
		return err
	}

	r.logger.Debug("User added to MongoDB", "email", user.Email)
	return nil
}

// GetReport retrieves users based on the report parameters
func (r *MongoDBRepository) GetReport(ctx any, params domain.ReportParams) ([]*domain.User, error) {
	r.logger.Debug("Generating report from MongoDB", "type", params.Type, "from", params.From, "to", params.To)

	// Create date filter
	dateFilter := bson.M{
		"date_added": bson.M{
			"$gte": params.From,
			"$lte": params.To,
		},
	}

	// Add report type filter
	var filter bson.M
	switch params.Type {
	case domain.ReportTypeRedeemed:
		// Only get users who have redeemed within the date range
		filter = bson.M{
			"$and": []bson.M{
				dateFilter,
				{"redeemed": bson.M{"$ne": nil}},
			},
		}
	case domain.ReportTypeAdded, domain.ReportTypeAll:
		// Get all users within the date range
		filter = dateFilter
	default:
		return nil, errors.New("invalid report type")
	}

	// Set up options (sorting by date added, newest first)
	findOptions := options.Find().SetSort(bson.M{"date_added": -1})

	// Execute query with timeout
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	cursor, err := r.collection.Find(ctxWithTimeout, filter, findOptions)
	if err != nil {
		r.logger.Error("Failed to execute MongoDB query", "error", err)
		return nil, err
	}
	defer cursor.Close(ctxWithTimeout)

	// Process results
	var mongoUsers []mongoUser
	if err = cursor.All(ctxWithTimeout, &mongoUsers); err != nil {
		r.logger.Error("Failed to decode MongoDB results", "error", err)
		return nil, err
	}

	// Convert to domain objects
	users := make([]*domain.User, len(mongoUsers))
	for i, mongoUser := range mongoUsers {
		users[i] = &domain.User{
			ID:        mongoUser.ID,
			Email:     mongoUser.Email,
			DateAdded: mongoUser.DateAdded,
			Redeemed:  mongoUser.Redeemed,
		}
	}

	r.logger.Info("Report generated from MongoDB", "type", params.Type, "count", len(users))
	return users, nil
}

func (r *MongoDBRepository) Close() error {
	r.logger.Debug("Closing MongoDB repository")
	if r.client != nil {
		return r.client.Disconnect(context.Background())
	}
	return nil
}