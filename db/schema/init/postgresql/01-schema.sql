-- PostgreSQL schema for Cocktail Bot
-- Creates the users table and indexes

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(100) PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    date_added TIMESTAMP WITH TIME ZONE NOT NULL,
    redeemed TIMESTAMP WITH TIME ZONE NULL
);

-- Add unique constraint for email
CREATE UNIQUE INDEX IF NOT EXISTS idx_email ON users (email);

-- Add index for efficient queries on redeemed status
CREATE INDEX IF NOT EXISTS idx_redeemed ON users (redeemed);

-- Add index for date_added for reporting
CREATE INDEX IF NOT EXISTS idx_date_added ON users (date_added);