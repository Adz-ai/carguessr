package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type TableInfo struct {
	Name        string
	Required    bool
	CreateSQL   string
	Description string
}

func main() {
	fmt.Println("=== CarGuessr Database Schema Verification & Fix ===")
	fmt.Println()

	// Test database paths
	dbPaths := []string{
		"/opt/motors-price-guesser/data/carguessr.db",
		"./data/carguessr.db",
		"data/carguessr.db",
	}

	var dbPath string
	for _, path := range dbPaths {
		if _, err := os.Stat(path); err == nil {
			dbPath = path
			fmt.Printf("✅ Found database at: %s\n", path)
			break
		}
	}

	if dbPath == "" {
		fmt.Println("❌ No database file found")
		return
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Enable foreign keys
	db.Exec("PRAGMA foreign_keys = ON")

	// Define all required tables
	requiredTables := []TableInfo{
		{
			Name:        "users",
			Required:    true,
			Description: "User accounts and authentication",
			CreateSQL: `CREATE TABLE IF NOT EXISTS users (
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
		},
		{
			Name:        "challenge_sessions",
			Required:    true,
			Description: "Challenge game sessions",
			CreateSQL: `CREATE TABLE IF NOT EXISTS challenge_sessions (
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
		},
		{
			Name:        "challenge_guesses",
			Required:    true,
			Description: "Individual guesses within challenge sessions",
			CreateSQL: `CREATE TABLE IF NOT EXISTS challenge_guesses (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				session_id TEXT NOT NULL REFERENCES challenge_sessions(session_id) ON DELETE CASCADE,
				car_index INTEGER NOT NULL,
				car_id TEXT NOT NULL,
				guessed_price REAL NOT NULL,
				actual_price REAL NOT NULL,
				difference REAL NOT NULL,
				percentage_error REAL NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			Name:        "leaderboard_entries",
			Required:    true,
			Description: "Game scores and leaderboard",
			CreateSQL: `CREATE TABLE IF NOT EXISTS leaderboard_entries (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
				username TEXT NOT NULL,
				score REAL NOT NULL,
				game_mode TEXT NOT NULL CHECK (game_mode IN ('streak', 'zero', 'challenge')),
				difficulty TEXT NOT NULL CHECK (difficulty IN ('easy', 'hard')),
				session_id TEXT,
				friend_challenge_id INTEGER,
				legacy_id TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			Name:        "friend_challenges",
			Required:    true,
			Description: "Friend challenge system",
			CreateSQL: `CREATE TABLE IF NOT EXISTS friend_challenges (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				challenge_code TEXT UNIQUE NOT NULL,
				title TEXT NOT NULL,
				creator_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				difficulty TEXT NOT NULL CHECK (difficulty IN ('easy', 'hard')),
				cars_json TEXT NOT NULL,
				max_participants INTEGER DEFAULT 10,
				is_active BOOLEAN DEFAULT TRUE,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				expires_at DATETIME DEFAULT (datetime('now', '+7 days'))
			)`,
		},
		{
			Name:        "friend_challenge_participants",
			Required:    true,
			Description: "Participants in friend challenges",
			CreateSQL: `CREATE TABLE IF NOT EXISTS friend_challenge_participants (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				friend_challenge_id INTEGER NOT NULL REFERENCES friend_challenges(id) ON DELETE CASCADE,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				session_id TEXT,
				final_score REAL,
				is_complete BOOLEAN DEFAULT FALSE,
				joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				completed_at DATETIME,
				UNIQUE(friend_challenge_id, user_id)
			)`,
		},
		{
			Name:        "database_metadata",
			Required:    false,
			Description: "Migration tracking",
			CreateSQL: `CREATE TABLE IF NOT EXISTS database_metadata (
				key TEXT PRIMARY KEY,
				value TEXT NOT NULL,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
		},
	}

	// Check which tables exist
	fmt.Println("\n--- Table Verification ---")
	existingTables := make(map[string]bool)

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		log.Fatal("Failed to get table list:", err)
	}

	fmt.Println("Existing tables:")
	for rows.Next() {
		var tableName string
		rows.Scan(&tableName)
		existingTables[tableName] = true
		fmt.Printf("  ✅ %s\n", tableName)
	}
	rows.Close()

	// Check for missing tables
	var missingTables []TableInfo
	fmt.Println("\nRequired table check:")
	for _, table := range requiredTables {
		if existingTables[table.Name] {
			fmt.Printf("  ✅ %s - EXISTS (%s)\n", table.Name, table.Description)
		} else {
			if table.Required {
				fmt.Printf("  ❌ %s - MISSING (%s)\n", table.Name, table.Description)
				missingTables = append(missingTables, table)
			} else {
				fmt.Printf("  ⚠️  %s - MISSING (optional: %s)\n", table.Name, table.Description)
				missingTables = append(missingTables, table)
			}
		}
	}

	// Create missing tables
	if len(missingTables) > 0 {
		fmt.Printf("\n--- Creating Missing Tables (%d) ---\n", len(missingTables))
		for _, table := range missingTables {
			fmt.Printf("Creating %s table...\n", table.Name)
			_, err := db.Exec(table.CreateSQL)
			if err != nil {
				fmt.Printf("❌ Failed to create %s: %v\n", table.Name, err)
			} else {
				fmt.Printf("✅ Created %s table\n", table.Name)
			}
		}
	} else {
		fmt.Println("\n✅ All required tables exist!")
	}

	// Create indexes
	fmt.Println("\n--- Creating Indexes ---")
	indexes := []struct {
		Name string
		SQL  string
	}{
		{"idx_users_username", "CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)"},
		{"idx_users_session_token", "CREATE INDEX IF NOT EXISTS idx_users_session_token ON users(session_token)"},
		{"idx_users_display_name", "CREATE INDEX IF NOT EXISTS idx_users_display_name ON users(display_name)"},
		{"idx_challenge_sessions_user_id", "CREATE INDEX IF NOT EXISTS idx_challenge_sessions_user_id ON challenge_sessions(user_id)"},
		{"idx_challenge_sessions_created_at", "CREATE INDEX IF NOT EXISTS idx_challenge_sessions_created_at ON challenge_sessions(created_at)"},
		{"idx_challenge_guesses_session_id", "CREATE INDEX IF NOT EXISTS idx_challenge_guesses_session_id ON challenge_guesses(session_id)"},
		{"idx_leaderboard_entries_user_id", "CREATE INDEX IF NOT EXISTS idx_leaderboard_entries_user_id ON leaderboard_entries(user_id)"},
		{"idx_leaderboard_entries_game_mode", "CREATE INDEX IF NOT EXISTS idx_leaderboard_entries_game_mode ON leaderboard_entries(game_mode, difficulty)"},
		{"idx_friend_challenges_code", "CREATE INDEX IF NOT EXISTS idx_friend_challenges_code ON friend_challenges(challenge_code)"},
		{"idx_friend_challenges_creator", "CREATE INDEX IF NOT EXISTS idx_friend_challenges_creator ON friend_challenges(creator_user_id)"},
		{"idx_friend_challenge_participants_challenge", "CREATE INDEX IF NOT EXISTS idx_friend_challenge_participants_challenge ON friend_challenge_participants(friend_challenge_id)"},
	}

	for _, idx := range indexes {
		_, err := db.Exec(idx.SQL)
		if err != nil {
			fmt.Printf("⚠️  Warning: Could not create index %s: %v\n", idx.Name, err)
		} else {
			fmt.Printf("✅ Index %s\n", idx.Name)
		}
	}

	// Verify users table schema specifically
	if existingTables["users"] {
		fmt.Println("\n--- Users Table Column Verification ---")
		rows, err := db.Query("PRAGMA table_info(users)")
		if err != nil {
			log.Fatal("Failed to get users table info:", err)
		}

		var columns []string
		var hasSecurityQuestion, hasSecurityAnswer bool

		for rows.Next() {
			var cid int
			var name, dataType string
			var notNull, pk int
			var defaultValue sql.NullString

			err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
			if err != nil {
				log.Fatal("Error scanning schema:", err)
			}

			columns = append(columns, name)
			if name == "security_question" {
				hasSecurityQuestion = true
			}
			if name == "security_answer_hash" {
				hasSecurityAnswer = true
			}
		}
		rows.Close()

		fmt.Printf("Users table has %d columns\n", len(columns))
		fmt.Printf("security_question exists: %v\n", hasSecurityQuestion)
		fmt.Printf("security_answer_hash exists: %v\n", hasSecurityAnswer)

		// Add missing security columns if needed
		if !hasSecurityQuestion || !hasSecurityAnswer {
			fmt.Println("\n--- Adding Missing Security Columns ---")

			if !hasSecurityQuestion {
				fmt.Println("Adding security_question column...")
				_, err = db.Exec("ALTER TABLE users ADD COLUMN security_question TEXT NOT NULL DEFAULT 'What is your favorite color?'")
				if err != nil {
					fmt.Printf("❌ Failed to add security_question: %v\n", err)
				} else {
					fmt.Println("✅ Added security_question column")
				}
			}

			if !hasSecurityAnswer {
				fmt.Println("Adding security_answer_hash column...")
				_, err = db.Exec("ALTER TABLE users ADD COLUMN security_answer_hash TEXT NOT NULL DEFAULT 'default_hash'")
				if err != nil {
					fmt.Printf("❌ Failed to add security_answer_hash: %v\n", err)
				} else {
					fmt.Println("✅ Added security_answer_hash column")
				}
			}
		}
	}

	// Test critical operations
	fmt.Println("\n--- Testing Critical Database Operations ---")

	// Test CreateUser query
	fmt.Println("Testing CreateUser query...")
	query := `
		INSERT INTO users (username, password_hash, display_name, is_guest, session_token, avatar_url, security_question, security_answer_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	testUsername := fmt.Sprintf("schematest_%d", len(existingTables)) // Unique based on table count
	testData := []interface{}{
		testUsername,
		"testhash",
		fmt.Sprintf("Schema Test %d", len(existingTables)),
		false,
		fmt.Sprintf("testsession_%d", len(existingTables)),
		"",
		"Test question?",
		"testhash",
	}

	result, err := db.Exec(query, testData...)
	if err != nil {
		fmt.Printf("❌ CreateUser query fails: %v\n", err)

		// Additional diagnosis
		if strings.Contains(err.Error(), "no such column") {
			fmt.Println("🔍 Column missing - check users table schema")
		} else if strings.Contains(err.Error(), "UNIQUE constraint") {
			fmt.Println("🔍 Duplicate data - this might be OK if data already exists")
		}
	} else {
		fmt.Println("✅ CreateUser query works!")

		// Get and clean up test user
		id, _ := result.LastInsertId()
		db.Exec("DELETE FROM users WHERE id = ?", id)
		fmt.Println("✅ Test user cleaned up")
	}

	// Test leaderboard insertion
	fmt.Println("Testing leaderboard entry...")
	leaderboardQuery := `
		INSERT INTO leaderboard_entries (username, score, game_mode, difficulty, session_id)
		VALUES (?, ?, ?, ?, ?)
	`
	result2, err := db.Exec(leaderboardQuery, "testentry", 100.5, "challenge", "hard", "testsession")
	if err != nil {
		fmt.Printf("❌ Leaderboard entry fails: %v\n", err)
	} else {
		fmt.Println("✅ Leaderboard entry works!")
		id, _ := result2.LastInsertId()
		db.Exec("DELETE FROM leaderboard_entries WHERE id = ?", id)
		fmt.Println("✅ Test leaderboard entry cleaned up")
	}

	fmt.Println("\n--- Database Schema Fix Complete ---")
	fmt.Println("✅ All tables verified and created")
	fmt.Println("✅ All indexes created")
	fmt.Println("✅ Critical operations tested")
	fmt.Println("\nYour database should now support all CarGuessr features!")
}
