package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	fmt.Println("=== CarGuessr Registration Debug Tool ===")
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
		fmt.Println("❌ No database file found at any expected location")
		fmt.Println("Expected locations:")
		for _, path := range dbPaths {
			fmt.Printf("  - %s\n", path)
		}
		return
	}

	// Check file permissions
	fmt.Println("\n--- File Permissions Check ---")
	info, err := os.Stat(dbPath)
	if err != nil {
		fmt.Printf("❌ Cannot stat database file: %v\n", err)
		return
	}
	fmt.Printf("Database file mode: %v\n", info.Mode())

	// Check directory permissions
	dbDir := filepath.Dir(dbPath)
	dirInfo, err := os.Stat(dbDir)
	if err != nil {
		fmt.Printf("❌ Cannot stat database directory: %v\n", err)
		return
	}
	fmt.Printf("Database directory mode: %v\n", dirInfo.Mode())

	// Test database connection
	fmt.Println("\n--- Database Connection Test ---")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Printf("❌ Cannot open database: %v\n", err)
		return
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Printf("❌ Cannot ping database: %v\n", err)
		return
	}
	fmt.Println("✅ Database connection successful")

	// Check if users table exists
	fmt.Println("\n--- Table Schema Check ---")
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableName)
	if err != nil {
		fmt.Printf("❌ Users table not found: %v\n", err)
		return
	}
	fmt.Println("✅ Users table exists")

	// Get table schema
	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		fmt.Printf("❌ Cannot get table info: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Println("\nUsers table schema:")
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString

		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			fmt.Printf("Error scanning schema: %v\n", err)
			continue
		}

		defaultStr := "NULL"
		if defaultValue.Valid {
			defaultStr = defaultValue.String
		}
		fmt.Printf("  %s: %s (NOT NULL: %v, DEFAULT: %s, PK: %v)\n",
			name, dataType, notNull == 1, defaultStr, pk == 1)
	}

	// Test write permissions
	fmt.Println("\n--- Write Permission Test ---")
	testQuery := "CREATE TABLE IF NOT EXISTS test_permissions (id INTEGER PRIMARY KEY, test_data TEXT)"
	_, err = db.Exec(testQuery)
	if err != nil {
		fmt.Printf("❌ Cannot create test table: %v\n", err)
		return
	}

	insertTest := "INSERT INTO test_permissions (test_data) VALUES (?)"
	_, err = db.Exec(insertTest, "test")
	if err != nil {
		fmt.Printf("❌ Cannot insert test data: %v\n", err)
		return
	}

	// Clean up test table
	db.Exec("DROP TABLE test_permissions")
	fmt.Println("✅ Write permissions OK")

	// Test the exact CreateUser query
	fmt.Println("\n--- CreateUser Query Test ---")

	// Generate test data
	testUsername := fmt.Sprintf("debugtest_%d", time.Now().Unix())
	testDisplayName := fmt.Sprintf("Debug Test %d", time.Now().Unix())
	testPassword := "debugpass123"
	testQuestion := "What is your favorite debug tool?"
	testAnswer := "gdb"

	// Hash password and security answer
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("❌ Cannot hash password: %v\n", err)
		return
	}

	hashedAnswer, err := bcrypt.GenerateFromPassword([]byte(strings.ToLower(strings.TrimSpace(testAnswer))), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("❌ Cannot hash security answer: %v\n", err)
		return
	}

	// Generate session token
	sessionToken := fmt.Sprintf("debug_%d", time.Now().UnixNano())

	// The exact query from CreateUser
	query := `
		INSERT INTO users (username, password_hash, display_name, is_guest, session_token, avatar_url, security_question, security_answer_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	fmt.Printf("Testing with:\n")
	fmt.Printf("  Username: %s\n", testUsername)
	fmt.Printf("  Display Name: %s\n", testDisplayName)
	fmt.Printf("  Session Token: %s\n", sessionToken)
	fmt.Printf("  Is Guest: %v\n", false)

	result, err := db.Exec(query, testUsername, string(hashedPassword),
		testDisplayName, false, sessionToken, "",
		testQuestion, string(hashedAnswer))

	if err != nil {
		fmt.Printf("❌ CreateUser query FAILED: %v\n", err)
		fmt.Println("\nThis is the exact error your production app is getting!")

		// Check for common issues
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			if strings.Contains(err.Error(), "username") {
				fmt.Println("🔍 Issue: Username already exists")
			} else if strings.Contains(err.Error(), "display_name") {
				fmt.Println("🔍 Issue: Display name already exists")
			} else if strings.Contains(err.Error(), "session_token") {
				fmt.Println("🔍 Issue: Session token collision")
			}
		} else if strings.Contains(err.Error(), "database is locked") {
			fmt.Println("🔍 Issue: Database is locked (WAL mode conflict?)")
		} else if strings.Contains(err.Error(), "no such table") {
			fmt.Println("🔍 Issue: Users table missing")
		} else if strings.Contains(err.Error(), "no such column") {
			fmt.Println("🔍 Issue: Column mismatch in table schema")
		}
		return
	}

	// Success - get the inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		fmt.Printf("❌ Cannot get last insert ID: %v\n", err)
		return
	}

	fmt.Printf("✅ CreateUser query SUCCESS! Inserted user ID: %d\n", id)

	// Verify the user was created
	var dbUsername, dbDisplayName string
	err = db.QueryRow("SELECT username, display_name FROM users WHERE id = ?", id).Scan(&dbUsername, &dbDisplayName)
	if err != nil {
		fmt.Printf("❌ Cannot verify created user: %v\n", err)
		return
	}

	fmt.Printf("✅ User verified: %s (%s)\n", dbUsername, dbDisplayName)

	// Clean up test user
	_, err = db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not clean up test user: %v\n", err)
	} else {
		fmt.Println("✅ Test user cleaned up")
	}

	fmt.Println("\n--- Summary ---")
	fmt.Println("✅ Database connection: OK")
	fmt.Println("✅ Users table: EXISTS")
	fmt.Println("✅ Write permissions: OK")
	fmt.Println("✅ CreateUser query: WORKS")
	fmt.Println()
	fmt.Println("If this test passes but registration still fails in production,")
	fmt.Println("the issue is likely in the web service layer, not the database.")
}
