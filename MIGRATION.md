
# Database Migration Guide

This guide explains how to set up the CarGuessr database on a new machine or migrate from JSON to SQLite.

## Quick Start

### For a brand new deployment:

```bash
# Initialize fresh database with current schema
go run cmd/migrate/main.go init
```

### For migrating from existing JSON data:

```bash
# Step 1: Initialize the database
go run cmd/migrate/main.go init

# Step 2: Import your existing leaderboard.json
go run cmd/migrate/main.go import-json ./data/leaderboard.json

# Step 3: Check everything worked
go run cmd/migrate/main.go status
```

### For updating an existing database:

```bash
# Run any pending migrations
go run cmd/migrate/main.go migrate

# Check current status
go run cmd/migrate/main.go status
```

## Commands

- **`init`** - Creates all tables with the latest schema (v2.0)
- **`migrate`** - Applies any pending database migrations
- **`import-json [path]`** - Imports leaderboard data from JSON file (defaults to ./data/leaderboard.json)
- **`status`** - Shows database status, table counts, and schema version

## Schema Changes

The database schema has evolved through these versions:

- **v1.0**: Initial SQLite schema
- **v1.1**: Added security_question and security_answer_hash fields
- **v1.2**: Removed email field, made display_name unique
- **v2.0**: Updated challenge expiration to 2 days

## Key Features

- **Safe Migrations**: All migrations run in transactions with rollback on failure
- **Duplicate Prevention**: Won't import JSON data twice
- **Version Tracking**: Tracks schema version to apply only needed migrations
- **Foreign Key Support**: Ensures data integrity with proper relationships
- **WAL Mode**: Optimized for concurrent read/write operations

## Current Schema (v2.0)

### Tables:
- `users` - User accounts (no email field, unique display names)
- `challenge_sessions` - Individual game sessions
- `challenge_guesses` - Individual guesses within sessions
- `friend_challenges` - Multiplayer challenges (2-day expiration)
- `challenge_participants` - Users participating in challenges
- `leaderboard_entries` - High scores with legacy import support
- `game_sessions` - Streak/zero mode sessions
- `database_metadata` - Schema versioning and migration tracking

### Security Features:
- bcrypt password hashing
- Security questions/answers for password reset
- Foreign key constraints
- Input validation at database level

## Troubleshooting

### Migration fails with "table already exists"
The migration tool uses `CREATE TABLE IF NOT EXISTS` so this shouldn't happen. If it does, check for table name conflicts.

### JSON import shows "0 imported"
Check that:
1. The JSON file exists and is readable
2. The JSON structure matches the expected format
3. You haven't already imported (check with `status` command)

### "No schema version found"
This means you need to run `init` first to create the initial database structure.

## File Structure

After migration, you'll have:
```
data/
├── carguessr.db         # Main SQLite database
├── carguessr.db-wal     # Write-ahead log (SQLite)
├── carguessr.db-shm     # Shared memory file (SQLite)
└── leaderboard.json     # Original JSON (kept as backup)
```

The application will automatically use the SQLite database if it exists, falling back to JSON only if the database is unavailable.