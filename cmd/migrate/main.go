package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Migration represents a database migration
type Migration struct {
	Version     string
	Description string
	SQL         []string
}

func main() {
	fmt.Println("🗃️  CarGuessr Database Migration Tool")
	fmt.Println("=====================================")

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/migrate/main.go <command> [args]")
		fmt.Println("Commands:")
		fmt.Println("  init          - Initialize database with current schema")
		fmt.Println("  migrate       - Run pending migrations")
		fmt.Println("  import-json   - Import leaderboard from JSON file")
		fmt.Println("  status        - Show migration status")
		os.Exit(1)
	}

	command := os.Args[1]

	// Ensure data directory exists
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}

	// Connect to database
	db, err := sql.Open("sqlite3", "./data/carguessr.db")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Enable foreign keys and WAL mode
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		log.Fatal("Failed to enable foreign keys:", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		log.Fatal("Failed to enable WAL mode:", err)
	}

	switch command {
	case "init":
		initializeDatabase(db)
	case "migrate":
		runMigrations(db)
	case "import-json":
		jsonPath := "./data/leaderboard.json"
		if len(os.Args) >= 3 {
			jsonPath = os.Args[2]
		}
		importLeaderboardFromJSON(db, jsonPath)
	case "status":
		showMigrationStatus(db)
	default:
		log.Fatal("Unknown command:", command)
	}
}

func initializeDatabase(db *sql.DB) {
	fmt.Println("Initializing database with current schema...")

	// Create all tables from schema
	schema := getCurrentSchema()

	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			log.Fatalf("Failed to execute schema statement: %v\nStatement: %s", err, stmt)
		}
	}

	fmt.Println("✅ Database initialized successfully!")
}

func runMigrations(db *sql.DB) {
	fmt.Println("Running database migrations...")

	// Ensure metadata table exists
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS database_metadata (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatal("Failed to create metadata table:", err)
	}

	// Check current schema version
	var currentVersion string
	err = db.QueryRow("SELECT value FROM database_metadata WHERE key = 'schema_version'").Scan(&currentVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			currentVersion = "0.0" // No version found, treat as new database
			_, err = db.Exec("INSERT INTO database_metadata (key, value) VALUES ('schema_version', '0.0')")
			if err != nil {
				log.Fatal("Failed to set initial version:", err)
			}
		} else {
			log.Fatal("Failed to check current schema version:", err)
		}
	}

	fmt.Printf("Current schema version: %s\n", currentVersion)

	migrations := getMigrations()
	applied := 0

	for _, migration := range migrations {
		if shouldApplyMigration(currentVersion, migration.Version) {
			fmt.Printf("Applying migration %s: %s\n", migration.Version, migration.Description)

			if err := applyMigration(db, migration); err != nil {
				log.Fatalf("Failed to apply migration %s: %v", migration.Version, err)
			}

			// Update schema version
			_, err := db.Exec("UPDATE database_metadata SET value = ?, updated_at = datetime('now') WHERE key = 'schema_version'", migration.Version)
			if err != nil {
				log.Fatal("Failed to update schema version:", err)
			}

			applied++
			currentVersion = migration.Version
		}
	}

	if applied == 0 {
		fmt.Println("✅ No migrations needed - database is up to date!")
	} else {
		fmt.Printf("✅ Applied %d migrations successfully!\n", applied)
	}
}

func importLeaderboardFromJSON(db *sql.DB, jsonPath string) {
	fmt.Printf("Importing leaderboard data from %s...\n", jsonPath)

	// Check if file exists
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		fmt.Printf("❌ File not found: %s\n", jsonPath)
		return
	}

	// Check if already imported
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM leaderboard_entries WHERE legacy_id IS NOT NULL").Scan(&count)
	if err != nil {
		log.Fatal("Failed to check existing data:", err)
	}

	if count > 0 {
		fmt.Printf("Found %d existing legacy entries. Skipping import to avoid duplicates.\n", count)
		fmt.Println("Use 'migrate status' to see current state.")
		return
	}

	// Read JSON file
	file, err := os.Open(jsonPath)
	if err != nil {
		log.Fatal("Failed to open JSON file:", err)
	}
	defer file.Close()

	var leaderboardData struct {
		Entries []struct {
			UserID     *int   `json:"userId,omitempty"`
			Name       string `json:"name"`
			Score      int    `json:"score"`
			GameMode   string `json:"gameMode"`
			Difficulty string `json:"difficulty,omitempty"`
			Date       string `json:"date"`
			SessionID  string `json:"sessionId,omitempty"`
		} `json:"entries"`
	}

	if err := json.NewDecoder(file).Decode(&leaderboardData); err != nil {
		log.Fatal("Failed to decode JSON:", err)
	}

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		log.Fatal("Failed to begin transaction:", err)
	}
	defer tx.Rollback()

	// Prepare statement
	stmt, err := tx.Prepare(`
		INSERT INTO leaderboard_entries 
		(user_id, username, score, game_mode, difficulty, session_id, legacy_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatal("Failed to prepare statement:", err)
	}
	defer stmt.Close()

	imported := 0
	for i, entry := range leaderboardData.Entries {
		// Set default difficulty if missing
		difficulty := entry.Difficulty
		if difficulty == "" {
			if entry.GameMode == "challenge" {
				difficulty = "easy" // Default for challenge mode
			} else {
				difficulty = "easy" // Default fallback
			}
		}

		// Parse date
		createdAt, err := time.Parse("2006-01-02 15:04:05", entry.Date)
		if err != nil {
			log.Printf("Warning: Failed to parse date for entry %d, using current time", i)
			createdAt = time.Now()
		}

		// Generate legacy ID
		legacyID := fmt.Sprintf("json_import_%d_%s", i, entry.SessionID)

		// Insert entry
		_, err = stmt.Exec(
			entry.UserID,
			entry.Name,
			entry.Score,
			entry.GameMode,
			difficulty,
			entry.SessionID,
			legacyID,
			createdAt,
		)
		if err != nil {
			log.Printf("Warning: Failed to insert entry %d (%s): %v", i, entry.Name, err)
			continue
		}

		imported++
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Fatal("Failed to commit transaction:", err)
	}

	fmt.Printf("✅ Successfully imported %d leaderboard entries!\n", imported)
}

func showMigrationStatus(db *sql.DB) {
	fmt.Println("Migration Status Report")
	fmt.Println("=====================")

	// Schema version
	var version string
	err := db.QueryRow("SELECT value FROM database_metadata WHERE key = 'schema_version'").Scan(&version)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("❌ No schema version found - database needs initialization")
		} else {
			fmt.Printf("❌ Error checking schema version: %v\n", err)
		}
		return
	}

	fmt.Printf("📊 Current Schema Version: %s\n", version)

	// Table counts
	tables := []string{"users", "leaderboard_entries", "challenge_sessions", "friend_challenges", "challenge_participants"}

	for _, table := range tables {
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			fmt.Printf("❌ Error counting %s: %v\n", table, err)
		} else {
			fmt.Printf("📋 %s: %d records\n", table, count)
		}
	}

	// Legacy data check
	var legacyCount int
	err = db.QueryRow("SELECT COUNT(*) FROM leaderboard_entries WHERE legacy_id IS NOT NULL").Scan(&legacyCount)
	if err == nil && legacyCount > 0 {
		fmt.Printf("📥 Legacy JSON entries: %d imported\n", legacyCount)
	}

	// Database file size
	if stat, err := os.Stat("./data/carguessr.db"); err == nil {
		fmt.Printf("💾 Database size: %.2f KB\n", float64(stat.Size())/1024)
	}

	fmt.Println("✅ Migration status check complete!")
}

func getCurrentSchema() []string {
	return []string{
		// Enable foreign keys
		"PRAGMA foreign_keys = ON",

		// Users table (updated schema without email field)
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL COLLATE NOCASE,
			password_hash TEXT NOT NULL,
			display_name TEXT UNIQUE NOT NULL COLLATE NOCASE,
			avatar_url TEXT,
			is_guest BOOLEAN DEFAULT FALSE,
			session_token TEXT UNIQUE,
			security_question TEXT NOT NULL,
			security_answer_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_active DATETIME DEFAULT CURRENT_TIMESTAMP,
			total_games_played INTEGER DEFAULT 0,
			favorite_difficulty TEXT DEFAULT 'easy' CHECK (favorite_difficulty IN ('easy', 'hard'))
		)`,

		// User indexes
		"CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)",
		"CREATE INDEX IF NOT EXISTS idx_users_session_token ON users(session_token)",
		"CREATE INDEX IF NOT EXISTS idx_users_display_name ON users(display_name)",

		// Challenge sessions table
		`CREATE TABLE IF NOT EXISTS challenge_sessions (
			session_id TEXT PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
			difficulty TEXT NOT NULL CHECK (difficulty IN ('easy', 'hard')),
			cars_json TEXT NOT NULL,
			current_car INTEGER DEFAULT 0,
			total_score INTEGER DEFAULT 0,
			is_complete BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			expires_at DATETIME DEFAULT (datetime('now', '+24 hours'))
		)`,

		// Challenge session indexes
		"CREATE INDEX IF NOT EXISTS idx_challenge_sessions_user_id ON challenge_sessions(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_challenge_sessions_created_at ON challenge_sessions(created_at)",

		// Challenge guesses table
		`CREATE TABLE IF NOT EXISTS challenge_guesses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL REFERENCES challenge_sessions(session_id) ON DELETE CASCADE,
			car_index INTEGER NOT NULL,
			car_id TEXT NOT NULL,
			guessed_price INTEGER NOT NULL,
			actual_price INTEGER NOT NULL,
			points INTEGER NOT NULL,
			accuracy_percentage REAL NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		"CREATE INDEX IF NOT EXISTS idx_challenge_guesses_session_id ON challenge_guesses(session_id)",

		// Friend challenges table (updated to 2-day expiration)
		`CREATE TABLE IF NOT EXISTS friend_challenges (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			challenge_code TEXT UNIQUE NOT NULL,
			title TEXT NOT NULL,
			creator_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			template_session_id TEXT NOT NULL REFERENCES challenge_sessions(session_id) ON DELETE CASCADE,
			difficulty TEXT NOT NULL CHECK (difficulty IN ('easy', 'hard')),
			max_participants INTEGER DEFAULT 10,
			is_active BOOLEAN DEFAULT TRUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME DEFAULT (datetime('now', '+2 days'))
		)`,

		// Friend challenge indexes
		"CREATE INDEX IF NOT EXISTS idx_friend_challenges_code ON friend_challenges(challenge_code)",
		"CREATE INDEX IF NOT EXISTS idx_friend_challenges_creator ON friend_challenges(creator_user_id)",

		// Challenge participants table
		`CREATE TABLE IF NOT EXISTS challenge_participants (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			friend_challenge_id INTEGER NOT NULL REFERENCES friend_challenges(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			session_id TEXT NOT NULL REFERENCES challenge_sessions(session_id) ON DELETE CASCADE,
			final_score INTEGER,
			rank_position INTEGER,
			completed_at DATETIME,
			joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(friend_challenge_id, user_id)
		)`,

		// Challenge participant indexes
		"CREATE INDEX IF NOT EXISTS idx_challenge_participants_challenge ON challenge_participants(friend_challenge_id)",
		"CREATE INDEX IF NOT EXISTS idx_challenge_participants_user ON challenge_participants(user_id)",

		// Leaderboard entries table
		`CREATE TABLE IF NOT EXISTS leaderboard_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
			username TEXT NOT NULL,
			score INTEGER NOT NULL,
			game_mode TEXT NOT NULL CHECK (game_mode IN ('streak', 'challenge', 'zero')),
			difficulty TEXT NOT NULL CHECK (difficulty IN ('easy', 'hard')),
			session_id TEXT,
			friend_challenge_id INTEGER REFERENCES friend_challenges(id) ON DELETE SET NULL,
			legacy_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Leaderboard indexes
		"CREATE INDEX IF NOT EXISTS idx_leaderboard_game_mode ON leaderboard_entries(game_mode)",
		"CREATE INDEX IF NOT EXISTS idx_leaderboard_difficulty ON leaderboard_entries(difficulty)",
		"CREATE INDEX IF NOT EXISTS idx_leaderboard_score ON leaderboard_entries(score)",
		"CREATE INDEX IF NOT EXISTS idx_leaderboard_user_id ON leaderboard_entries(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_leaderboard_created_at ON leaderboard_entries(created_at)",

		// Game sessions table
		`CREATE TABLE IF NOT EXISTS game_sessions (
			session_id TEXT PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
			game_mode TEXT NOT NULL CHECK (game_mode IN ('streak', 'zero')),
			difficulty TEXT NOT NULL CHECK (difficulty IN ('easy', 'hard')),
			current_score INTEGER DEFAULT 0,
			current_difference REAL DEFAULT 0,
			cars_shown TEXT DEFAULT '[]',
			is_active BOOLEAN DEFAULT TRUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_guess_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Game session indexes
		"CREATE INDEX IF NOT EXISTS idx_game_sessions_user_id ON game_sessions(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_game_sessions_active ON game_sessions(is_active)",

		// Database metadata table
		`CREATE TABLE IF NOT EXISTS database_metadata (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Initial metadata
		`INSERT OR REPLACE INTO database_metadata (key, value) VALUES 
			('schema_version', '2.0'),
			('created_at', datetime('now')),
			('migration_status', 'completed')`,
	}
}

func getMigrations() []Migration {
	return []Migration{
		{
			Version:     "1.0",
			Description: "Initial schema setup",
			SQL:         getCurrentSchema(),
		},
		{
			Version:     "1.1",
			Description: "Add security fields to users table",
			SQL: []string{
				"ALTER TABLE users ADD COLUMN security_question TEXT",
				"ALTER TABLE users ADD COLUMN security_answer_hash TEXT",
			},
		},
		{
			Version:     "1.2",
			Description: "Make display_name unique and remove email field",
			SQL: []string{
				// Create new users table without email
				`CREATE TABLE users_new (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					username TEXT UNIQUE NOT NULL COLLATE NOCASE,
					password_hash TEXT NOT NULL,
					display_name TEXT UNIQUE NOT NULL COLLATE NOCASE,
					avatar_url TEXT,
					is_guest BOOLEAN DEFAULT FALSE,
					session_token TEXT UNIQUE,
					security_question TEXT NOT NULL,
					security_answer_hash TEXT NOT NULL,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					last_active DATETIME DEFAULT CURRENT_TIMESTAMP,
					total_games_played INTEGER DEFAULT 0,
					favorite_difficulty TEXT DEFAULT 'easy' CHECK (favorite_difficulty IN ('easy', 'hard'))
				)`,
				// Copy data only for users with security fields
				`INSERT INTO users_new 
					SELECT id, username, password_hash, display_name, avatar_url, is_guest, 
						   session_token, security_question, security_answer_hash, created_at, 
						   last_active, total_games_played, favorite_difficulty 
					FROM users 
					WHERE security_question IS NOT NULL AND security_answer_hash IS NOT NULL`,
				"DROP TABLE users",
				"ALTER TABLE users_new RENAME TO users",
				"CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)",
				"CREATE INDEX IF NOT EXISTS idx_users_session_token ON users(session_token)",
				"CREATE INDEX IF NOT EXISTS idx_users_display_name ON users(display_name)",
			},
		},
		{
			Version:     "2.0",
			Description: "Update challenge expiration to 2 days",
			SQL: []string{
				"UPDATE friend_challenges SET expires_at = datetime(created_at, '+2 days') WHERE expires_at > datetime(created_at, '+2 days')",
			},
		},
		{
			Version:     "2.1",
			Description: "Add session expiration to users table",
			SQL: []string{
				"ALTER TABLE users ADD COLUMN session_expires_at DATETIME",
				// Set expiration for existing sessions to 7 days from last_active
				"UPDATE users SET session_expires_at = datetime(last_active, '+7 days') WHERE session_token IS NOT NULL",
			},
		},
	}
}

func shouldApplyMigration(currentVersion, migrationVersion string) bool {
	// Simple version comparison for semantic versioning
	return migrationVersion > currentVersion
}

func applyMigration(db *sql.DB, migration Migration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range migration.SQL {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute: %s\nError: %w", stmt, err)
		}
	}

	return tx.Commit()
}
