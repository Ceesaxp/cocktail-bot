package utils

import (
	"fmt"
	"time"
)

// FormatTime formats a time in a human-readable format
func FormatTime(t time.Time) string {
	return t.Format("January 2, 2006 at 15:04")
}

// ParseTime parses a time string using multiple common formats
func ParseTime(timeStr string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"01/02/2006 15:04:05",
		"01/02/2006",
		"2006-01-02",
		"January 2, 2006 at 15:04",
		"January 2, 2006",
	}

	var lastErr error
	for _, format := range formats {
		t, err := time.Parse(format, timeStr)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}

	return time.Time{}, fmt.Errorf("cannot parse time '%s': %v", timeStr, lastErr)
}

// TimeDiffHumanReadable returns a human-readable time difference description
func TimeDiffHumanReadable(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 30*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if diff < 365*24*time.Hour {
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	} else {
		years := int(diff.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}
