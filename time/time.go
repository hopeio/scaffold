package time

import (
	"fmt"
	"time"
)

// FormatRelativeTime formats the time difference from fromTime to now.
func FormatRelativeTime(fromTime time.Time) string {
	now := time.Now()
	duration := now.Sub(fromTime)

	days := int(duration.Hours() / 24)
	weeks := days / 7
	months := int(duration.Hours() / (24 * 30)) // simplified; real month lengths vary
	years := months / 12

	switch {
	case duration.Minutes() < 1:
		return "just now"
	case duration.Hours() < 1:
		return fmt.Sprintf("%d minutes ago", int(duration.Minutes()))
	case days < 1:
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	case days < 7:
		return fmt.Sprintf("%d days ago", days)
	case weeks < 1:
		return fmt.Sprintf("%d weeks ago", weeks)
	case months < 1:
		return fmt.Sprintf("%d months ago", months)
	default:
		return fmt.Sprintf("%d years ago", years)
	}
}