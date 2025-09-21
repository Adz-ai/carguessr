package database

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"autotraderguesser/internal/models"
)

func TestMigrateLeaderboardFromJSON(t *testing.T) {
	db := newTestDatabase(t)
	defer db.Close()

	jsonData := `{"entries":[{"name":"Alice","score":123,"gameMode":"challenge","difficulty":"easy","date":"2024-01-02 03:04:05","id":"legacy"}]}`
	jsonPath := filepath.Join(t.TempDir(), "leaderboard.json")
	if err := os.WriteFile(jsonPath, []byte(jsonData), 0o644); err != nil {
		t.Fatalf("failed to write json: %v", err)
	}

	if err := db.MigrateLeaderboardFromJSON(jsonPath); err != nil {
		t.Fatalf("first migration failed: %v", err)
	}

	var count int
	if err := db.db.QueryRow("SELECT COUNT(*) FROM leaderboard_entries WHERE legacy_id = ?", "legacy").Scan(&count); err != nil {
		t.Fatalf("failed to query migrated entries: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 migrated entry, got %d", count)
	}

	// Second run should detect completed migration and skip
	if err := db.MigrateLeaderboardFromJSON(jsonPath); err != nil {
		t.Fatalf("second migration failed: %v", err)
	}

	var status string
	if err := db.db.QueryRow("SELECT value FROM database_metadata WHERE key = 'migration_status'").Scan(&status); err != nil {
		t.Fatalf("failed to read migration status: %v", err)
	}
	if status != "completed" {
		t.Fatalf("expected migration status completed, got %s", status)
	}
}

func TestLeaderboardQueriesAndBackup(t *testing.T) {
	db := newTestDatabase(t)
	defer db.Close()

	samples := []models.LeaderboardEntry{
		{Name: "Alice", Score: 200, GameMode: "challenge", Difficulty: "easy"},
		{Name: "Bob", Score: 150, GameMode: "challenge", Difficulty: "easy"},
		{Name: "Cara", Score: 3, GameMode: "streak", Difficulty: "hard"},
	}

	for i := range samples {
		sample := samples[i]
		if err := db.AddLeaderboardEntry(&sample); err != nil {
			t.Fatalf("AddLeaderboardEntry failed: %v", err)
		}
	}

	entries, err := db.GetLeaderboard("challenge", "easy", 0)
	if err != nil {
		t.Fatalf("GetLeaderboard challenge failed: %v", err)
	}
	if len(entries) != 2 || entries[0].Score != 200 {
		t.Fatalf("unexpected challenge leaderboard ordering: %v", entries)
	}

	streakEntries, err := db.GetLeaderboard("streak", "hard", 10)
	if err != nil {
		t.Fatalf("GetLeaderboard streak failed: %v", err)
	}
	if len(streakEntries) != 1 || streakEntries[0].Score != 3 {
		t.Fatalf("unexpected streak leaderboard: %v", streakEntries)
	}

	// Backup current data
	dataDir := t.TempDir()
	files := map[string]string{
		"leaderboard.json":   "{\"entries\":[]}",
		"bonhams_cache.json": "{}",
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(dataDir, name), []byte(contents), 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	if err := db.BackupCurrentData(dataDir); err != nil {
		t.Fatalf("BackupCurrentData failed: %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(dataDir, "backup_*"))
	if err != nil {
		t.Fatalf("failed to glob backups: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one backup directory, got %v", matches)
	}

	for name := range files {
		backupPath := filepath.Join(matches[0], name)
		if _, err := os.Stat(backupPath); err != nil {
			t.Fatalf("expected %s to be backed up: %v", name, err)
		}
	}

	// Ensure timestamped directory name
	if info, err := os.Stat(matches[0]); err == nil {
		if !info.IsDir() {
			t.Fatalf("backup path should be directory")
		}
	} else {
		t.Fatalf("stat backup dir failed: %v", err)
	}

	// Run backup again and ensure files still present
	time.Sleep(10 * time.Millisecond)
	if err := db.BackupCurrentData(dataDir); err != nil {
		t.Fatalf("second backup failed: %v", err)
	}
	matches, err = filepath.Glob(filepath.Join(dataDir, "backup_*"))
	if err != nil {
		t.Fatalf("glob after second backup failed: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected at least one backup directory")
	}
	for _, dir := range matches {
		for name := range files {
			if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
				t.Fatalf("expected %s to exist in %s: %v", name, dir, err)
			}
		}
	}
}
