-- MySQL schema for Cocktail Bot
-- Creates the users table and indexes

USE cocktailbot;

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(100) PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    date_added DATETIME NOT NULL,
    redeemed DATETIME NULL,
    UNIQUE INDEX idx_email (email)
);

-- Add index for efficient queries on redeemed status
CREATE INDEX idx_redeemed ON users (redeemed);

-- Add index for date_added for reporting
CREATE INDEX idx_date_added ON users (date_added);