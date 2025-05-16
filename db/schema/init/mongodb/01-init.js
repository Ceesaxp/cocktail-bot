// MongoDB initialization script for Cocktail Bot

// Connect to the cocktailbot database
db = db.getSiblingDB('cocktailbot');

// Create a unique index on email to prevent duplicates
db.users.createIndex({ "email": 1 }, { unique: true });

// Create indexes for efficient querying
db.users.createIndex({ "redeemed": 1 });
db.users.createIndex({ "date_added": 1 });

// Display the created indexes
print("MongoDB indexes created for Cocktail Bot:");
db.users.getIndexes().forEach(function(index) {
    print(JSON.stringify(index));
});