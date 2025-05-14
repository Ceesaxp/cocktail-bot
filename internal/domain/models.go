package domain

import "time"

const (
	ErrUserNotFound        = "User email is not found in database"
	ErrDatabaseUnavailable = "Database is not available, try later"
)

type User struct {
	ID              string
	Email           string
	DateAdded       time.Time
	AlreadyConsumed *time.Time
}
