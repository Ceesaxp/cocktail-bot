package domain

import (
	"fmt"
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

// ReportType defines the type of report to generate
type ReportType string

const (
	// ReportTypeRedeemed represents a report of redeemed cocktails
	ReportTypeRedeemed ReportType = "redeemed"
	// ReportTypeAdded represents a report of added users
	ReportTypeAdded ReportType = "added"
	// ReportTypeAll represents a report of all users
	ReportTypeAll ReportType = "all"
)

// ValidateReportType checks if the provided string is a valid report type
func ValidateReportType(reportType string) (ReportType, error) {
	switch reportType {
	case string(ReportTypeRedeemed):
		return ReportTypeRedeemed, nil
	case string(ReportTypeAdded):
		return ReportTypeAdded, nil
	case string(ReportTypeAll):
		return ReportTypeAll, nil
	default:
		return "", fmt.Errorf("invalid report type: %s", reportType)
	}
}

// ReportParams holds parameters for generating reports
type ReportParams struct {
	Type      ReportType
	From      time.Time
	To        time.Time
}

// Repository is the interface that all database implementations must satisfy
type Repository interface {
	FindByEmail(ctx any, email string) (*User, error)
	UpdateUser(ctx any, user *User) error
	AddUser(ctx any, user *User) error
	GetReport(ctx any, params ReportParams) ([]*User, error)
	Close() error
}
