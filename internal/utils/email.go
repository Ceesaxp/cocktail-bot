package utils

import (
	"net/mail"
	"strings"
)

// IsValidEmail checks if a string is a valid email address
func IsValidEmail(email string) bool {
	// Check using standard library first
	_, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}

	// Additional checks
	email = strings.TrimSpace(email)

	// Check for @ symbol and domain part
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}

	// Check domain has at least one dot
	domainParts := strings.Split(parts[1], ".")
	if len(domainParts) < 2 {
		return false
	}

	// Check TLD is not empty and reasonable length
	tld := domainParts[len(domainParts)-1]
	if tld == "" || len(tld) < 2 {
		return false
	}

	return true
}

// NormalizeEmail normalizes an email address for storage and comparison
func NormalizeEmail(email string) string {
	// Trim spaces and convert to lowercase
	return strings.ToLower(strings.TrimSpace(email))
}
