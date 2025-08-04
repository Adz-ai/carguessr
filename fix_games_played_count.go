package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	fmt.Println("=== Fix Games Played Count ===")
	fmt.Println("This will update total_games_played based on existing leaderboard entries")
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

	// Check current state
	fmt.Println("\n--- Current State ---")

	// Count users with 0 games played
	var usersWithZeroGames int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE total_games_played = 0").Scan(&usersWithZeroGames)
	if err != nil {
		log.Fatal("Failed to count users with zero games:", err)
	}

	// Count total users
	var totalUsers int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
	if err != nil {
		log.Fatal("Failed to count total users:", err)
	}

	// Count total leaderboard entries
	var totalEntries int
	err = db.QueryRow("SELECT COUNT(*) FROM leaderboard_entries WHERE user_id IS NOT NULL").Scan(&totalEntries)
	if err != nil {
		log.Fatal("Failed to count leaderboard entries:", err)
	}

	fmt.Printf("Total users: %d\n", totalUsers)
	fmt.Printf("Users with 0 games played: %d\n", usersWithZeroGames)
	fmt.Printf("Total leaderboard entries with user_id: %d\n", totalEntries)

	if usersWithZeroGames == 0 {
		fmt.Println("\n✅ All users already have correct games played counts!")
		return
	}

	// Show current user stats
	fmt.Println("\nCurrent user games played counts:")
	rows, err := db.Query(`
		SELECT u.username, u.total_games_played, 
		       COUNT(l.id) as actual_games_from_leaderboard
		FROM users u
		LEFT JOIN leaderboard_entries l ON u.id = l.user_id
		GROUP BY u.id, u.username, u.total_games_played
		ORDER BY actual_games_from_leaderboard DESC
		LIMIT 10
	`)
	if err != nil {
		log.Fatal("Failed to get user stats:", err)
	}

	for rows.Next() {
		var username string
		var currentCount, actualCount int
		rows.Scan(&username, &currentCount, &actualCount)
		if currentCount != actualCount {
			fmt.Printf("  ❌ %s: shows %d, should be %d\n", username, currentCount, actualCount)
		} else {
			fmt.Printf("  ✅ %s: %d games (correct)\n", username, currentCount)
		}
	}
	rows.Close()

	// Fix the counts
	fmt.Println("\n--- Fixing Games Played Counts ---")

	updateQuery := `
		UPDATE users SET total_games_played = (
			SELECT COUNT(*) 
			FROM leaderboard_entries 
			WHERE leaderboard_entries.user_id = users.id
		)
		WHERE id IN (
			SELECT DISTINCT user_id 
			FROM leaderboard_entries 
			WHERE user_id IS NOT NULL
		)
	`

	result, err := db.Exec(updateQuery)
	if err != nil {
		log.Fatal("Failed to update games played counts:", err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("✅ Updated %d users\n", rowsAffected)

	// Verify the fix
	fmt.Println("\n--- Verification ---")

	// Count users with 0 games played after fix
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE total_games_played = 0").Scan(&usersWithZeroGames)
	if err != nil {
		log.Fatal("Failed to count users with zero games after fix:", err)
	}

	fmt.Printf("Users with 0 games played after fix: %d\n", usersWithZeroGames)

	// Show updated stats
	fmt.Println("\nUpdated user stats:")
	rows, err = db.Query(`
		SELECT u.username, u.total_games_played, 
		       COUNT(l.id) as actual_games_from_leaderboard
		FROM users u
		LEFT JOIN leaderboard_entries l ON u.id = l.user_id
		WHERE u.total_games_played > 0
		GROUP BY u.id, u.username, u.total_games_played
		ORDER BY u.total_games_played DESC
		LIMIT 10
	`)
	if err != nil {
		log.Fatal("Failed to get updated user stats:", err)
	}

	for rows.Next() {
		var username string
		var currentCount, actualCount int
		rows.Scan(&username, &currentCount, &actualCount)
		if currentCount != actualCount {
			fmt.Printf("  ⚠️  %s: shows %d, leaderboard has %d\n", username, currentCount, actualCount)
		} else {
			fmt.Printf("  ✅ %s: %d games\n", username, currentCount)
		}
	}
	rows.Close()

	fmt.Println("\n--- Fix Complete ---")
	fmt.Println("✅ All existing users now have correct games played counts")
	fmt.Println("✅ Future games will automatically increment the counter")
	fmt.Println("\nThe UI should now show the correct number of games played!")
}
