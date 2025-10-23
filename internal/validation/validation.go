package validation

import (
	"fmt"
	"regexp"
)

// ValidateChallengeCode validates that a challenge code is in the correct format (6 uppercase alphanumeric characters)
func ValidateChallengeCode(code string) error {
	if len(code) != 6 {
		return fmt.Errorf("challenge code must be exactly 6 characters")
	}

	if !regexp.MustCompile(`^[A-Z0-9]{6}$`).MatchString(code) {
		return fmt.Errorf("challenge code must contain only uppercase letters and numbers")
	}

	return nil
}

// ValidateSessionID validates that a session ID is in the correct format
// Supports multiple formats:
// - 16 characters alphanumeric (legacy format)
// - 22 characters base64 RawURL (new cryptographic format, no padding)
// - 19-25 characters for timestamp-based fallback format (session_<timestamp>)
func ValidateSessionID(sessionID string) error {
	if len(sessionID) < 16 || len(sessionID) > 32 {
		return fmt.Errorf("session ID must be between 16 and 32 characters")
	}

	// Allow alphanumeric plus URL-safe base64 characters (-, _)
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(sessionID) {
		return fmt.Errorf("session ID contains invalid characters")
	}

	return nil
}

// ValidateUsername validates username format
func ValidateUsername(username string) error {
	if len(username) < 3 || len(username) > 20 {
		return fmt.Errorf("username must be between 3 and 20 characters")
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(username) {
		return fmt.Errorf("username can only contain letters, numbers, underscores and hyphens")
	}

	return nil
}

// ValidateDisplayName validates display name format
func ValidateDisplayName(displayName string) error {
	if len(displayName) < 1 || len(displayName) > 30 {
		return fmt.Errorf("display name must be between 1 and 30 characters")
	}

	// Allow more characters for display names but prevent harmful content
	if !regexp.MustCompile(`^[a-zA-Z0-9\s._-]+$`).MatchString(displayName) {
		return fmt.Errorf("display name contains invalid characters")
	}

	return nil
}

// ValidateChallengeTitle validates and sanitizes challenge titles
func ValidateChallengeTitle(title string) (string, error) {
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")     // Normalize whitespace
	title = regexp.MustCompile(`[<>\"'&]`).ReplaceAllString(title, "") // Remove HTML/XSS chars

	if len(title) < 1 || len(title) > 100 {
		return "", fmt.Errorf("challenge title must be between 1 and 100 characters")
	}

	return title, nil
}
