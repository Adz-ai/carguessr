-- CarGuessr SQLite Database Schema
-- Version: 1.0

-- Enable foreign key constraints
PRAGMA foreign_keys = ON;

-- Users table for account system
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL COLLATE NOCASE,
    password_hash TEXT NOT NULL, -- bcrypt hashed password
    display_name TEXT UNIQUE NOT NULL COLLATE NOCASE, -- Made unique and required
    avatar_url TEXT,
    is_guest BOOLEAN DEFAULT FALSE, -- Changed default to FALSE since we removed guest accounts
    session_token TEXT UNIQUE, -- Simple session management
    security_question TEXT NOT NULL, -- Security question for password reset
    security_answer_hash TEXT NOT NULL, -- bcrypt hashed security answer
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_active DATETIME DEFAULT CURRENT_TIMESTAMP,
    total_games_played INTEGER DEFAULT 0,
    favorite_difficulty TEXT DEFAULT 'easy' CHECK (favorite_difficulty IN ('easy', 'hard'))
);

-- Create index for fast username lookups
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_session_token ON users(session_token);

-- Challenge sessions table (persistent storage)
CREATE TABLE IF NOT EXISTS challenge_sessions (
    session_id TEXT PRIMARY KEY, -- 16 character session ID
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    difficulty TEXT NOT NULL CHECK (difficulty IN ('easy', 'hard')),
    cars_json TEXT NOT NULL, -- JSON array of the 10 cars with details
    current_car INTEGER DEFAULT 0,
    total_score INTEGER DEFAULT 0,
    is_complete BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    expires_at DATETIME DEFAULT (datetime('now', '+24 hours')) -- Sessions expire in 24 hours
);

-- Create index for session lookups
CREATE INDEX IF NOT EXISTS idx_challenge_sessions_user_id ON challenge_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_challenge_sessions_created_at ON challenge_sessions(created_at);

-- Individual guesses within challenge sessions
CREATE TABLE IF NOT EXISTS challenge_guesses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES challenge_sessions(session_id) ON DELETE CASCADE,
    car_index INTEGER NOT NULL, -- Which car in the challenge (0-9)
    car_id TEXT NOT NULL, -- Reference to the car in cache
    guessed_price INTEGER NOT NULL,
    actual_price INTEGER NOT NULL,
    points INTEGER NOT NULL,
    accuracy_percentage REAL NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create index for guess lookups
CREATE INDEX IF NOT EXISTS idx_challenge_guesses_session_id ON challenge_guesses(session_id);

-- Friend challenges table for multiplayer
CREATE TABLE IF NOT EXISTS friend_challenges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    challenge_code TEXT UNIQUE NOT NULL, -- 6-8 character code like "GEOG123"
    title TEXT NOT NULL,
    creator_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    template_session_id TEXT NOT NULL REFERENCES challenge_sessions(session_id) ON DELETE CASCADE,
    difficulty TEXT NOT NULL CHECK (difficulty IN ('easy', 'hard')),
    max_participants INTEGER DEFAULT 10,
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME DEFAULT (datetime('now', '+7 days')) -- Challenges expire in 7 days
);

-- Create index for challenge code lookups
CREATE INDEX IF NOT EXISTS idx_friend_challenges_code ON friend_challenges(challenge_code);
CREATE INDEX IF NOT EXISTS idx_friend_challenges_creator ON friend_challenges(creator_user_id);
-- Composite index for common query pattern: lookup by code AND check if active/expired
CREATE INDEX IF NOT EXISTS idx_friend_challenges_code_active ON friend_challenges(challenge_code, is_active, expires_at);

-- Challenge participants table
CREATE TABLE IF NOT EXISTS challenge_participants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    friend_challenge_id INTEGER NOT NULL REFERENCES friend_challenges(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id TEXT NOT NULL REFERENCES challenge_sessions(session_id) ON DELETE CASCADE,
    final_score INTEGER,
    rank_position INTEGER, -- 1st, 2nd, 3rd place etc
    completed_at DATETIME,
    joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(friend_challenge_id, user_id) -- One entry per user per challenge
);

-- Create index for participant lookups
CREATE INDEX IF NOT EXISTS idx_challenge_participants_challenge ON challenge_participants(friend_challenge_id);
CREATE INDEX IF NOT EXISTS idx_challenge_participants_user ON challenge_participants(user_id);

-- Enhanced leaderboard entries with user references
CREATE TABLE IF NOT EXISTS leaderboard_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    username TEXT NOT NULL, -- Denormalized for performance and guest support
    score INTEGER NOT NULL,
    game_mode TEXT NOT NULL CHECK (game_mode IN ('streak', 'challenge', 'zero')),
    difficulty TEXT NOT NULL CHECK (difficulty IN ('easy', 'hard')),
    session_id TEXT, -- Link to challenge session if applicable
    friend_challenge_id INTEGER REFERENCES friend_challenges(id) ON DELETE SET NULL, -- If part of friend challenge
    legacy_id TEXT, -- For migrating existing JSON data
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for leaderboard performance
CREATE INDEX IF NOT EXISTS idx_leaderboard_game_mode ON leaderboard_entries(game_mode);
CREATE INDEX IF NOT EXISTS idx_leaderboard_difficulty ON leaderboard_entries(difficulty);
CREATE INDEX IF NOT EXISTS idx_leaderboard_score ON leaderboard_entries(score);
CREATE INDEX IF NOT EXISTS idx_leaderboard_user_id ON leaderboard_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_leaderboard_created_at ON leaderboard_entries(created_at);
-- Composite index for common query pattern: filter by game_mode AND difficulty
CREATE INDEX IF NOT EXISTS idx_leaderboard_mode_difficulty ON leaderboard_entries(game_mode, difficulty, score DESC);

-- Game sessions for tracking streaks and zero mode
CREATE TABLE IF NOT EXISTS game_sessions (
    session_id TEXT PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    game_mode TEXT NOT NULL CHECK (game_mode IN ('streak', 'zero')),
    difficulty TEXT NOT NULL CHECK (difficulty IN ('easy', 'hard')),
    current_score INTEGER DEFAULT 0,
    current_difference REAL DEFAULT 0, -- For zero mode
    cars_shown TEXT DEFAULT '[]', -- JSON array of car IDs shown to prevent repetition
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_guess_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create index for active session lookups
CREATE INDEX IF NOT EXISTS idx_game_sessions_user_id ON game_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_game_sessions_active ON game_sessions(is_active);

-- Database metadata table
CREATE TABLE IF NOT EXISTS database_metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert initial metadata
INSERT OR REPLACE INTO database_metadata (key, value) VALUES 
    ('schema_version', '1.0'),
    ('created_at', datetime('now')),
    ('migration_status', 'pending');