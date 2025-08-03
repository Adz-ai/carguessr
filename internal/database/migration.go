package database

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"autotraderguesser/internal/models"
)

// MigrateLeaderboardFromJSON migrates existing leaderboard data from JSON to database
func (d *Database) MigrateLeaderboardFromJSON(jsonPath string) error {
	// Check if migration already completed
	var migrationStatus string
	err := d.db.QueryRow("SELECT value FROM database_metadata WHERE key = 'migration_status'").Scan(&migrationStatus)
	if err == nil && migrationStatus == "completed" {
		fmt.Println("Migration already completed, skipping...")
		return nil
	}

	// Read existing leaderboard JSON
	file, err := os.Open(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to open leaderboard file: %w", err)
	}
	defer file.Close()

	// Legacy JSON structure for migration
	var leaderboardData struct {
		Entries []struct {
			Name       string `json:"name"`
			Score      int    `json:"score"`
			GameMode   string `json:"gameMode"`
			Difficulty string `json:"difficulty,omitempty"`
			Date       string `json:"date"`
			ID         string `json:"id,omitempty"` // Legacy string ID
		} `json:"entries"`
	}

	if err := json.NewDecoder(file).Decode(&leaderboardData); err != nil {
		return fmt.Errorf("failed to decode leaderboard JSON: %w", err)
	}

	// Begin transaction
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare insert statement
	stmt, err := tx.Prepare(`
		INSERT INTO leaderboard_entries 
		(username, score, game_mode, difficulty, legacy_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Insert each entry
	for _, entry := range leaderboardData.Entries {
		// Parse date
		var createdAt time.Time
		if entry.Date != "" {
			if parsed, err := time.Parse("2006-01-02 15:04:05", entry.Date); err == nil {
				createdAt = parsed
			} else {
				createdAt = time.Now() // Fallback to current time
			}
		} else {
			createdAt = time.Now()
		}

		// Set default difficulty if missing
		difficulty := entry.Difficulty
		if difficulty == "" {
			difficulty = "hard" // Default for legacy entries
		}

		_, err := stmt.Exec(entry.Name, entry.Score, entry.GameMode, difficulty, entry.ID, createdAt)
		if err != nil {
			return fmt.Errorf("failed to insert entry %s: %w", entry.Name, err)
		}
	}

	// Update migration status
	_, err = tx.Exec("UPDATE database_metadata SET value = 'completed' WHERE key = 'migration_status'")
	if err != nil {
		return fmt.Errorf("failed to update migration status: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("Successfully migrated %d leaderboard entries to database\n", len(leaderboardData.Entries))
	return nil
}

// GetLeaderboard retrieves leaderboard entries with filtering
func (d *Database) GetLeaderboard(gameMode, difficulty string, limit int) ([]models.LeaderboardEntry, error) {
	query := `
		SELECT username, score, game_mode, difficulty, created_at
		FROM leaderboard_entries
		WHERE 1=1
	`
	args := []interface{}{}

	if gameMode != "" {
		query += " AND game_mode = ?"
		args = append(args, gameMode)
	}

	if difficulty != "" {
		query += " AND difficulty = ?"
		args = append(args, difficulty)
	}

	// Sort by score (descending for challenge, ascending for streak/zero)
	if gameMode == "challenge" {
		query += " ORDER BY score DESC"
	} else {
		query += " ORDER BY score ASC"
	}

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []models.LeaderboardEntry
	for rows.Next() {
		var entry models.LeaderboardEntry
		var createdAt time.Time

		err := rows.Scan(&entry.Name, &entry.Score, &entry.GameMode, &entry.Difficulty, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		entry.Date = createdAt.Format("2006-01-02 15:04:05")
		entries = append(entries, entry)
	}

	return entries, nil
}

// AddLeaderboardEntry adds a new leaderboard entry
func (d *Database) AddLeaderboardEntry(entry *models.LeaderboardEntry) error {
	query := `
		INSERT INTO leaderboard_entries 
		(user_id, username, score, game_mode, difficulty, session_id, friend_challenge_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query, entry.UserID, entry.Name, entry.Score, 
		entry.GameMode, entry.Difficulty, entry.SessionID, entry.FriendChallengeID)
	
	if err != nil {
		return fmt.Errorf("failed to add leaderboard entry: %w", err)
	}

	return nil
}

// BackupCurrentData creates a backup of current JSON files before migration
func (d *Database) BackupCurrentData(dataDir string) error {
	backupDir := fmt.Sprintf("%s/backup_%d", dataDir, time.Now().Unix())
	
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Files to backup
	files := []string{"leaderboard.json", "bonhams_cache.json", "lookers_cache.json"}
	
	for _, filename := range files {
		srcPath := fmt.Sprintf("%s/%s", dataDir, filename)
		dstPath := fmt.Sprintf("%s/%s", backupDir, filename)
		
		// Check if source file exists
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue // Skip if file doesn't exist
		}
		
		// Copy file
		if err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to backup %s: %w", filename, err)
		}
	}

	fmt.Printf("Data backed up to: %s\n", backupDir)
	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}