package leaderboard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"autotraderguesser/internal/models"
)

const LeaderboardFile = "data/leaderboard.json"

// LeaderboardData represents the structure saved to file
type LeaderboardData struct {
	Entries   []models.LeaderboardEntry `json:"entries"`
	LastSaved time.Time                 `json:"lastSaved"`
	Version   string                    `json:"version"`
}

// SaveToFile saves leaderboard entries to a JSON file
func SaveToFile(entries []models.LeaderboardEntry) error {
	// Ensure data directory exists
	if err := os.MkdirAll("data", 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	data := LeaderboardData{
		Entries:   entries,
		LastSaved: time.Now(),
		Version:   "1.0",
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal leaderboard data: %v", err)
	}

	// Write to file
	if err := os.WriteFile(LeaderboardFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write leaderboard file: %v", err)
	}

	return nil
}

// LoadFromFile loads leaderboard entries from a JSON file
func LoadFromFile() ([]models.LeaderboardEntry, error) {
	// Check if file exists
	if _, err := os.Stat(LeaderboardFile); os.IsNotExist(err) {
		// File doesn't exist, return empty leaderboard
		return []models.LeaderboardEntry{}, nil
	}

	// Read file
	jsonData, err := os.ReadFile(LeaderboardFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read leaderboard file: %v", err)
	}

	// Parse JSON
	var data LeaderboardData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal leaderboard data: %v", err)
	}

	return data.Entries, nil
}

// GetFileAge returns the age of the leaderboard file
func GetFileAge() (time.Duration, error) {
	info, err := os.Stat(LeaderboardFile)
	if err != nil {
		return 0, err
	}
	return time.Since(info.ModTime()), nil
}

// FileExists checks if leaderboard file exists
func FileExists() bool {
	_, err := os.Stat(LeaderboardFile)
	return !os.IsNotExist(err)
}

// GetAbsolutePath returns the absolute path to the leaderboard file
func GetAbsolutePath() (string, error) {
	return filepath.Abs(LeaderboardFile)
}
