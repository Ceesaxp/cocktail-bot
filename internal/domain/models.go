package domain

import (
	"time"
)

type User struct {
	ID         string
	Email      string
	DateAdded  time.Time
	Redeemed   *time.Time
}

// IsRedeemed returns true if the user has already redeemed their cocktail
func (u *User) IsRedeemed() bool {
	return u.Redeemed != nil
}

// Redeem marks the user as having redeemed their cocktail with the current time
func (u *User) Redeem() {
	now := time.Now()
	u.Redeemed = &now
}

// Repository is the interface that all database implementations must satisfy
type Repository interface {
	FindByEmail(ctx any, email string) (*User, error)
	UpdateUser(ctx any, user *User) error
	Close() error
}
