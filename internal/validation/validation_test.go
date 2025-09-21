package validation

import (
	"strings"
	"testing"
)

func TestValidateChallengeCode(t *testing.T) {
	cases := []struct {
		name    string
		code    string
		wantErr string
	}{
		{"short", "ABC", "challenge code must be exactly 6 characters"},
		{"invalidChars", "ABC$12", "challenge code must contain only uppercase letters and numbers"},
		{"valid", "ABC123", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateChallengeCode(tc.code)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("expected error %q, got %v", tc.wantErr, err)
				}
			}
		})
	}
}

func TestValidateSessionID(t *testing.T) {
	cases := []struct {
		name    string
		id      string
		wantErr string
	}{
		{"short", "ABC", "session ID must be exactly 16 characters"},
		{"invalidChars", "ABC!123456789012", "session ID must contain only letters and numbers"},
		{"valid", "abcd1234ABCD5678", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSessionID(tc.id)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("expected error %q, got %v", tc.wantErr, err)
				}
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantErr string
	}{
		{"tooShort", "ab", "username must be between 3 and 20 characters"},
		{"tooLong", "abcdefghijklmnopqrstuvwxyz", "username must be between 3 and 20 characters"},
		{"invalidChars", "user!*", "username can only contain letters, numbers, underscores and hyphens"},
		{"valid", "user_name-123", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateUsername(tc.value)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("expected error %q, got %v", tc.wantErr, err)
				}
			}
		})
	}
}

func TestValidateDisplayName(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantErr string
	}{
		{"tooShort", "", "display name must be between 1 and 30 characters"},
		{"tooLong", strings.Repeat("a", 31), "display name must be between 1 and 30 characters"},
		{"invalidChars", "Name@", "display name contains invalid characters"},
		{"valid", "Friendly User", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDisplayName(tc.value)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("expected error %q, got %v", tc.wantErr, err)
				}
			}
		})
	}
}
