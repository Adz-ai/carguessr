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

// ValidateSessionID validates that a session ID is in the correct format (16 alphanumeric characters)
func ValidateSessionID(sessionID string) error {
	if len(sessionID) != 16 {
		return fmt.Errorf("session ID must be exactly 16 characters")
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9]{16}$`).MatchString(sessionID) {
		return fmt.Errorf("session ID must contain only letters and numbers")
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
