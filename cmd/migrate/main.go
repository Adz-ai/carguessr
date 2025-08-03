package main

import (
	"fmt"
	"log"
	"os"

	"autotraderguesser/internal/database"
)

func main() {
	fmt.Println("🗃️  CarGuessr Database Migration Tool")
	fmt.Println("=====================================")

	// Initialize database
	dbPath := "./data/carguessr.db"
	db, err := database.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ Database initialized successfully")

	// Check if data directory exists
	dataDir := "./data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		fmt.Println("❌ Data directory not found, creating...")
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Fatalf("Failed to create data directory: %v", err)
		}
	}

	// Create backup of existing JSON data
	if _, err := os.Stat("./data/leaderboard.json"); err == nil {
		fmt.Println("📦 Creating backup of existing data...")
		if err := db.BackupCurrentData("./data"); err != nil {
			log.Printf("Warning: Failed to create backup: %v", err)
		} else {
			fmt.Println("✅ Backup created successfully")
		}
	}

	// Migrate leaderboard data
	leaderboardPath := "./data/leaderboard.json"
	if _, err := os.Stat(leaderboardPath); err == nil {
		fmt.Println("🔄 Migrating leaderboard data...")
		if err := db.MigrateLeaderboardFromJSON(leaderboardPath); err != nil {
			log.Fatalf("Failed to migrate leaderboard: %v", err)
		}
		fmt.Println("✅ Leaderboard migration completed")
	} else {
		fmt.Println("ℹ️  No existing leaderboard.json found, starting fresh")
	}

	// Test database by querying leaderboard
	fmt.Println("🔍 Testing database with sample query...")
	entries, err := db.GetLeaderboard("", "", 5)
	if err != nil {
		log.Fatalf("Failed to query leaderboard: %v", err)
	}

	fmt.Printf("📊 Found %d leaderboard entries (showing top 5):\n", len(entries))
	for i, entry := range entries {
		fmt.Printf("   %d. %s - %d points (%s mode, %s difficulty)\n",
			i+1, entry.Name, entry.Score, entry.GameMode, entry.Difficulty)
	}

	fmt.Println("\n🎉 Database migration completed successfully!")
	fmt.Println("   Database file: " + dbPath)
	fmt.Printf("   Database size: ")

	// Show database file size
	if stat, err := os.Stat(dbPath); err == nil {
		fmt.Printf("%.2f KB\n", float64(stat.Size())/1024)
	} else {
		fmt.Println("Unknown")
	}
}
