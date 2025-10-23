package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"autotraderguesser/internal/models"
)

type Database struct {
	db *sql.DB
}

var randomReader io.Reader = rand.Reader

// NewDatabase creates a new database connection
func NewDatabase(dbPath string) (*Database, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection with SQLite optimizations
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on&_cache_size=10000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1) // SQLite works best with single connection
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	database := &Database{db: db}

	// Initialize schema
	if err := database.initializeSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return database, nil
}

// Ping checks database connectivity
func (d *Database) Ping() error {
	return d.db.Ping()
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// initializeSchema creates tables and runs migrations
func (d *Database) initializeSchema() error {
	// Read schema file
	schemaPath := filepath.Join("internal", "database", "schema.sql")
	schemaFile, err := os.Open(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to open schema file: %w", err)
	}
	defer schemaFile.Close()

	schema, err := io.ReadAll(schemaFile)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Execute schema
	if _, err := d.db.Exec(string(schema)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// User management methods

// CreateUser creates a new user in the database
func (d *Database) CreateUser(user *models.User) error {
	query := `
		INSERT INTO users (username, password_hash, display_name, is_guest, session_token, avatar_url, security_question, security_answer_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := d.db.Exec(query, user.Username, user.PasswordHash,
		user.DisplayName, user.IsGuest, user.SessionToken, user.AvatarURL,
		user.SecurityQuestion, user.SecurityAnswerHash)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Get the inserted user ID
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get user ID: %w", err)
	}

	user.ID = int(id)
	user.CreatedAt = time.Now()
	user.LastActive = time.Now()

	return nil
}

// GetUserBySessionToken retrieves a user by their session token
func (d *Database) GetUserBySessionToken(token string) (*models.User, error) {
	query := `
		SELECT id, username, display_name, password_hash, avatar_url, session_token, session_expires_at, is_guest,
		       security_question, security_answer_hash, created_at, last_active, total_games_played, favorite_difficulty
		FROM users
		WHERE session_token = ?
	`

	var user models.User
	var passwordHash, avatarURL, securityQuestion, securityAnswerHash sql.NullString
	var sessionExpiresAt sql.NullTime

	err := d.db.QueryRow(query, token).Scan(
		&user.ID, &user.Username, &user.DisplayName, &passwordHash,
		&avatarURL, &user.SessionToken, &sessionExpiresAt, &user.IsGuest, &securityQuestion, &securityAnswerHash,
		&user.CreatedAt, &user.LastActive, &user.TotalGamesPlayed, &user.FavoriteDifficulty,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Handle nullable fields
	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if securityQuestion.Valid {
		user.SecurityQuestion = securityQuestion.String
	}
	if securityAnswerHash.Valid {
		user.SecurityAnswerHash = securityAnswerHash.String
	}
	if sessionExpiresAt.Valid {
		user.SessionExpiresAt = &sessionExpiresAt.Time
	}

	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (d *Database) GetUserByUsername(username string) (*models.User, error) {
	query := `
		SELECT id, username, display_name, password_hash, avatar_url, session_token, is_guest, 
		       security_question, security_answer_hash, created_at, last_active, total_games_played, favorite_difficulty
		FROM users 
		WHERE username = ? COLLATE NOCASE
	`

	var user models.User
	var passwordHash, avatarURL, sessionToken, securityQuestion, securityAnswerHash sql.NullString

	err := d.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.DisplayName, &passwordHash,
		&avatarURL, &sessionToken, &user.IsGuest, &securityQuestion, &securityAnswerHash,
		&user.CreatedAt, &user.LastActive, &user.TotalGamesPlayed, &user.FavoriteDifficulty,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Handle nullable fields
	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if sessionToken.Valid {
		user.SessionToken = sessionToken.String
	}
	if securityQuestion.Valid {
		user.SecurityQuestion = securityQuestion.String
	}
	if securityAnswerHash.Valid {
		user.SecurityAnswerHash = securityAnswerHash.String
	}

	return &user, nil
}

// UpdateUserActivity updates the last active timestamp
func (d *Database) UpdateUserActivity(userID int) error {
	query := `UPDATE users SET last_active = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := d.db.Exec(query, userID)
	return err
}

// GetUserByDisplayName retrieves a user by display name
func (d *Database) GetUserByDisplayName(displayName string) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, display_name, avatar_url, 
			   is_guest, session_token, security_question, security_answer_hash, created_at, last_active, total_games_played, favorite_difficulty
		FROM users 
		WHERE display_name = ? COLLATE NOCASE
	`

	var user models.User
	var passwordHash, avatarURL, sessionToken, securityQuestion, securityAnswerHash sql.NullString

	err := d.db.QueryRow(query, displayName).Scan(
		&user.ID, &user.Username, &passwordHash, &user.DisplayName, &avatarURL,
		&user.IsGuest, &sessionToken, &securityQuestion, &securityAnswerHash,
		&user.CreatedAt, &user.LastActive, &user.TotalGamesPlayed, &user.FavoriteDifficulty,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Handle nullable fields
	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if sessionToken.Valid {
		user.SessionToken = sessionToken.String
	}
	if securityQuestion.Valid {
		user.SecurityQuestion = securityQuestion.String
	}
	if securityAnswerHash.Valid {
		user.SecurityAnswerHash = securityAnswerHash.String
	}

	return &user, nil
}

// UpdateUserSession updates a user's session token
func (d *Database) UpdateUserSession(userID int, sessionToken string) error {
	// Sessions expire after 7 days
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	query := `UPDATE users SET session_token = ?, session_expires_at = ?, last_active = ? WHERE id = ?`

	_, err := d.db.Exec(query, sessionToken, expiresAt, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update user session: %w", err)
	}

	return nil
}

// UpdateUserLastActive updates a user's last active time
func (d *Database) UpdateUserLastActive(userID int) error {
	query := `UPDATE users SET last_active = ? WHERE id = ?`

	_, err := d.db.Exec(query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update last active: %w", err)
	}

	return nil
}

// UpdateUserPassword updates a user's password
func (d *Database) UpdateUserPassword(userID int, passwordHash string) error {
	query := `UPDATE users SET password_hash = ?, last_active = ? WHERE id = ?`

	_, err := d.db.Exec(query, passwordHash, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}

	return nil
}

// UpdateUser updates user profile information
func (d *Database) UpdateUser(user *models.User) error {
	query := `
		UPDATE users 
		SET display_name = ?, avatar_url = ?, favorite_difficulty = ?, last_active = ?
		WHERE id = ?
	`

	_, err := d.db.Exec(query, user.DisplayName, user.AvatarURL,
		user.FavoriteDifficulty, time.Now(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdateUserFavoriteDifficulty calculates and updates user's favorite difficulty based on gameplay
func (d *Database) UpdateUserFavoriteDifficulty(userID int) error {
	// Count games played by difficulty
	query := `
		SELECT difficulty, COUNT(*) as count
		FROM leaderboard_entries 
		WHERE user_id = ?
		GROUP BY difficulty
		ORDER BY count DESC
		LIMIT 1
	`

	var favoriteDifficulty string
	var count int
	err := d.db.QueryRow(query, userID).Scan(&favoriteDifficulty, &count)

	if err != nil {
		if err == sql.ErrNoRows {
			// User has no games played, keep default
			return nil
		}
		return fmt.Errorf("failed to calculate favorite difficulty: %w", err)
	}

	// Update user's favorite difficulty
	updateQuery := `UPDATE users SET favorite_difficulty = ? WHERE id = ?`
	_, err = d.db.Exec(updateQuery, favoriteDifficulty, userID)
	if err != nil {
		return fmt.Errorf("failed to update favorite difficulty: %w", err)
	}

	return nil
}

// UpdateAllUsersFavoriteDifficulty updates favorite difficulty for all users based on their gameplay
func (d *Database) UpdateAllUsersFavoriteDifficulty() error {
	query := `
		UPDATE users 
		SET favorite_difficulty = (
			SELECT difficulty 
			FROM leaderboard_entries 
			WHERE user_id = users.id 
			GROUP BY difficulty 
			ORDER BY COUNT(*) DESC 
			LIMIT 1
		) 
		WHERE id IN (
			SELECT DISTINCT user_id 
			FROM leaderboard_entries 
			WHERE user_id IS NOT NULL
		)
	`

	_, err := d.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to update all users' favorite difficulty: %w", err)
	}

	return nil
}

// IncrementUserGamesPlayed increments the total_games_played counter for a user
func (d *Database) IncrementUserGamesPlayed(userID int) error {
	query := `
		UPDATE users 
		SET total_games_played = total_games_played + 1, 
		    last_active = datetime('now')
		WHERE id = ?
	`

	_, err := d.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to increment games played: %w", err)
	}

	return nil
}

// Challenge session methods

// CreateChallengeSession creates a new challenge session
func (d *Database) CreateChallengeSession(session *models.ChallengeSession) error {
	carsJSON, err := json.Marshal(session.Cars)
	if err != nil {
		return fmt.Errorf("failed to marshal cars: %w", err)
	}

	query := `
		INSERT INTO challenge_sessions (session_id, user_id, difficulty, cars_json, current_car, total_score)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	var userID *int
	if session.UserID != 0 {
		userID = &session.UserID
	}

	_, err = d.db.Exec(query, session.SessionID, userID, session.Difficulty,
		string(carsJSON), session.CurrentCar, session.TotalScore)

	if err != nil {
		return fmt.Errorf("failed to create challenge session: %w", err)
	}

	return nil
}

// GetChallengeSession retrieves a challenge session by ID
func (d *Database) GetChallengeSession(sessionID string) (*models.ChallengeSession, error) {
	query := `
		SELECT session_id, user_id, difficulty, cars_json, current_car, total_score, 
		       is_complete, created_at, completed_at
		FROM challenge_sessions 
		WHERE session_id = ? AND expires_at > CURRENT_TIMESTAMP
	`

	var session models.ChallengeSession
	var userID sql.NullInt64
	var carsJSON string
	var completedAt sql.NullTime

	err := d.db.QueryRow(query, sessionID).Scan(
		&session.SessionID, &userID, &session.Difficulty, &carsJSON,
		&session.CurrentCar, &session.TotalScore, &session.IsComplete,
		&session.StartTime, &completedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Session not found or expired
		}
		return nil, fmt.Errorf("failed to get challenge session: %w", err)
	}

	// Parse user ID
	if userID.Valid {
		session.UserID = int(userID.Int64)
	}

	// Parse completed time
	if completedAt.Valid {
		session.CompletedTime = completedAt.Time.Format(time.RFC3339)
	}

	// Parse cars JSON
	if err := json.Unmarshal([]byte(carsJSON), &session.Cars); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cars: %w", err)
	}

	// Load guesses
	guesses, err := d.getChallengeGuesses(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load guesses: %w", err)
	}
	session.Guesses = guesses

	return &session, nil
}

// UpdateChallengeSession updates an existing challenge session
func (d *Database) UpdateChallengeSession(session *models.ChallengeSession) error {
	query := `
		UPDATE challenge_sessions 
		SET current_car = ?, total_score = ?, is_complete = ?, completed_at = ?
		WHERE session_id = ?
	`

	var completedAt *time.Time
	if session.IsComplete {
		now := time.Now()
		completedAt = &now
	}

	_, err := d.db.Exec(query, session.CurrentCar, session.TotalScore,
		session.IsComplete, completedAt, session.SessionID)

	if err != nil {
		return fmt.Errorf("failed to update challenge session: %w", err)
	}

	return nil
}

// AddChallengeGuess adds a guess to a challenge session
func (d *Database) AddChallengeGuess(sessionID string, guess *models.ChallengeGuess) error {
	query := `
		INSERT INTO challenge_guesses 
		(session_id, car_index, car_id, guessed_price, actual_price, points, accuracy_percentage)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query, sessionID, guess.CarIndex, guess.CarID,
		guess.GuessedPrice, guess.ActualPrice, guess.Points, guess.Percentage)

	if err != nil {
		return fmt.Errorf("failed to add challenge guess: %w", err)
	}

	return nil
}

// getChallengeGuesses retrieves all guesses for a challenge session
func (d *Database) getChallengeGuesses(sessionID string) ([]models.ChallengeGuess, error) {
	query := `
		SELECT car_index, car_id, guessed_price, actual_price, points, accuracy_percentage, created_at
		FROM challenge_guesses 
		WHERE session_id = ?
		ORDER BY car_index
	`

	rows, err := d.db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guesses []models.ChallengeGuess
	for rows.Next() {
		var guess models.ChallengeGuess
		var createdAt time.Time

		err := rows.Scan(&guess.CarIndex, &guess.CarID, &guess.GuessedPrice,
			&guess.ActualPrice, &guess.Points, &guess.Percentage, &createdAt)
		if err != nil {
			return nil, err
		}

		guesses = append(guesses, guess)
	}

	return guesses, nil
}

// Friend challenge methods

// CreateFriendChallenge creates a new friend challenge
func (d *Database) CreateFriendChallenge(challenge *models.FriendChallenge) error {
	query := `
		INSERT INTO friend_challenges 
		(challenge_code, title, creator_user_id, template_session_id, difficulty, max_participants, is_active, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := d.db.Exec(query, challenge.ChallengeCode, challenge.Title, challenge.CreatorUserID,
		challenge.TemplateSessionID, challenge.Difficulty, challenge.MaxParticipants, challenge.IsActive,
		challenge.CreatedAt, challenge.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to create friend challenge: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get challenge ID: %w", err)
	}

	challenge.ID = int(id)
	return nil
}

// ChallengeCodeExists checks if a challenge code already exists
func (d *Database) ChallengeCodeExists(code string) (bool, error) {
	query := `SELECT COUNT(*) FROM friend_challenges WHERE challenge_code = ? AND is_active = TRUE AND expires_at > ?`

	var count int
	err := d.db.QueryRow(query, code, time.Now()).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check challenge code: %w", err)
	}

	return count > 0, nil
}

// GetFriendChallengeByCode retrieves a friend challenge by its code
func (d *Database) GetFriendChallengeByCode(code string) (*models.FriendChallenge, error) {
	query := `
		SELECT fc.id, fc.challenge_code, fc.title, fc.creator_user_id, fc.template_session_id,
		       fc.difficulty, fc.max_participants, fc.is_active, fc.created_at, fc.expires_at,
		       u.display_name as creator_display_name
		FROM friend_challenges fc
		JOIN users u ON fc.creator_user_id = u.id
		WHERE fc.challenge_code = ? AND fc.is_active = TRUE AND fc.expires_at > ?
	`

	var challenge models.FriendChallenge
	err := d.db.QueryRow(query, code, time.Now()).Scan(
		&challenge.ID, &challenge.ChallengeCode, &challenge.Title, &challenge.CreatorUserID,
		&challenge.TemplateSessionID, &challenge.Difficulty, &challenge.MaxParticipants,
		&challenge.IsActive, &challenge.CreatedAt, &challenge.ExpiresAt, &challenge.CreatorDisplayName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("challenge not found")
		}
		return nil, fmt.Errorf("failed to get challenge: %w", err)
	}

	return &challenge, nil
}

// AddChallengeParticipant adds a participant to a friend challenge
func (d *Database) AddChallengeParticipant(participant *models.ChallengeParticipant) error {
	query := `
		INSERT INTO challenge_participants 
		(friend_challenge_id, user_id, session_id, joined_at)
		VALUES (?, ?, ?, ?)
	`

	result, err := d.db.Exec(query, participant.FriendChallengeID, participant.UserID,
		participant.SessionID, participant.JoinedAt)
	if err != nil {
		return fmt.Errorf("failed to add participant: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get participant ID: %w", err)
	}

	participant.ID = int(id)
	return nil
}

// GetChallengeParticipants gets all participants for a challenge
func (d *Database) GetChallengeParticipants(challengeID int) ([]models.ChallengeParticipant, error) {
	query := `
		SELECT cp.id, cp.friend_challenge_id, cp.user_id, cp.session_id,
		       cp.final_score, cp.rank_position, cp.completed_at, cp.joined_at,
		       u.display_name as user_display_name
		FROM challenge_participants cp
		JOIN users u ON cp.user_id = u.id
		WHERE cp.friend_challenge_id = ?
		ORDER BY cp.rank_position ASC, cp.joined_at ASC
	`

	rows, err := d.db.Query(query, challengeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}
	defer rows.Close()

	var participants []models.ChallengeParticipant
	for rows.Next() {
		var p models.ChallengeParticipant
		var finalScore sql.NullInt64
		var rankPosition sql.NullInt64
		var completedAt sql.NullTime

		err := rows.Scan(&p.ID, &p.FriendChallengeID, &p.UserID, &p.SessionID,
			&finalScore, &rankPosition, &completedAt, &p.JoinedAt, &p.UserDisplayName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}

		if finalScore.Valid {
			score := int(finalScore.Int64)
			p.FinalScore = &score
		}
		if rankPosition.Valid {
			rank := int(rankPosition.Int64)
			p.RankPosition = &rank
		}
		if completedAt.Valid {
			p.CompletedAt = &completedAt.Time
		}

		participants = append(participants, p)
	}

	return participants, nil
}

// GetUserChallengeParticipation gets a user's participation in a specific challenge
func (d *Database) GetUserChallengeParticipation(challengeID, userID int) (*models.ChallengeParticipant, error) {
	query := `
		SELECT cp.id, cp.friend_challenge_id, cp.user_id, cp.session_id,
		       cp.final_score, cp.rank_position, cp.completed_at, cp.joined_at,
		       u.display_name as user_display_name
		FROM challenge_participants cp
		JOIN users u ON cp.user_id = u.id
		WHERE cp.friend_challenge_id = ? AND cp.user_id = ?
	`

	var p models.ChallengeParticipant
	var finalScore sql.NullInt64
	var rankPosition sql.NullInt64
	var completedAt sql.NullTime

	err := d.db.QueryRow(query, challengeID, userID).Scan(
		&p.ID, &p.FriendChallengeID, &p.UserID, &p.SessionID,
		&finalScore, &rankPosition, &completedAt, &p.JoinedAt, &p.UserDisplayName)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("participation not found")
		}
		return nil, fmt.Errorf("failed to get participation: %w", err)
	}

	if finalScore.Valid {
		score := int(finalScore.Int64)
		p.FinalScore = &score
	}
	if rankPosition.Valid {
		rank := int(rankPosition.Int64)
		p.RankPosition = &rank
	}
	if completedAt.Valid {
		p.CompletedAt = &completedAt.Time
	}

	return &p, nil
}

// GetUserCreatedChallenges gets challenges created by a user
func (d *Database) GetUserCreatedChallenges(userID int) ([]models.FriendChallenge, error) {
	query := `
		SELECT fc.id, fc.challenge_code, fc.title, fc.creator_user_id, fc.template_session_id,
		       fc.difficulty, fc.max_participants, fc.is_active, fc.created_at, fc.expires_at,
		       u.display_name as creator_display_name,
		       COUNT(cp.id) as participant_count,
		       creator_cp.joined_at,
		       creator_cs.is_complete
		FROM friend_challenges fc
		JOIN users u ON fc.creator_user_id = u.id
		LEFT JOIN challenge_participants cp ON fc.id = cp.friend_challenge_id
		LEFT JOIN challenge_participants creator_cp ON fc.id = creator_cp.friend_challenge_id AND creator_cp.user_id = fc.creator_user_id
		LEFT JOIN challenge_sessions creator_cs ON creator_cp.session_id = creator_cs.session_id
		WHERE fc.creator_user_id = ?
		GROUP BY fc.id, creator_cp.joined_at, creator_cs.is_complete
		ORDER BY fc.created_at DESC
	`

	rows, err := d.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created challenges: %w", err)
	}
	defer rows.Close()

	var challenges []models.FriendChallenge
	for rows.Next() {
		var c models.FriendChallenge
		var joinedAt *time.Time
		var isComplete *bool

		err := rows.Scan(&c.ID, &c.ChallengeCode, &c.Title, &c.CreatorUserID,
			&c.TemplateSessionID, &c.Difficulty, &c.MaxParticipants, &c.IsActive,
			&c.CreatedAt, &c.ExpiresAt, &c.CreatorDisplayName, &c.ParticipantCount,
			&joinedAt, &isComplete)
		if err != nil {
			return nil, fmt.Errorf("failed to scan challenge: %w", err)
		}

		// Set creator participation fields (if they've joined their own challenge)
		c.JoinedAt = joinedAt
		if isComplete != nil {
			c.IsComplete = *isComplete
		} else {
			c.IsComplete = false
		}

		challenges = append(challenges, c)
	}

	return challenges, nil
}

// GetUserParticipatingChallenges gets challenges a user is participating in
func (d *Database) GetUserParticipatingChallenges(userID int) ([]models.FriendChallenge, error) {
	query := `
		SELECT fc.id, fc.challenge_code, fc.title, fc.creator_user_id, fc.template_session_id,
		       fc.difficulty, fc.max_participants, fc.is_active, fc.created_at, fc.expires_at,
		       u.display_name as creator_display_name,
		       COUNT(cp2.id) as participant_count,
		       cp.joined_at,
		       cs.is_complete
		FROM friend_challenges fc
		JOIN users u ON fc.creator_user_id = u.id
		JOIN challenge_participants cp ON fc.id = cp.friend_challenge_id
		LEFT JOIN challenge_participants cp2 ON fc.id = cp2.friend_challenge_id
		LEFT JOIN challenge_sessions cs ON cp.session_id = cs.session_id
		WHERE cp.user_id = ? AND fc.creator_user_id != ?
		GROUP BY fc.id, cp.joined_at, cs.is_complete
		ORDER BY fc.created_at DESC
	`

	rows, err := d.db.Query(query, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participating challenges: %w", err)
	}
	defer rows.Close()

	var challenges []models.FriendChallenge
	for rows.Next() {
		var c models.FriendChallenge
		var joinedAt time.Time
		var isComplete *bool

		err := rows.Scan(&c.ID, &c.ChallengeCode, &c.Title, &c.CreatorUserID,
			&c.TemplateSessionID, &c.Difficulty, &c.MaxParticipants, &c.IsActive,
			&c.CreatedAt, &c.ExpiresAt, &c.CreatorDisplayName, &c.ParticipantCount,
			&joinedAt, &isComplete)
		if err != nil {
			return nil, fmt.Errorf("failed to scan challenge: %w", err)
		}

		// Set participant-specific fields
		c.JoinedAt = &joinedAt
		if isComplete != nil {
			c.IsComplete = *isComplete
		} else {
			c.IsComplete = false
		}

		challenges = append(challenges, c)
	}

	return challenges, nil
}

// CalculateChallengeRankings calculates and updates rankings for a challenge
func (d *Database) CalculateChallengeRankings(challengeID int, participants []models.ChallengeParticipant) error {
	// Sort participants by score (descending), then by completion time (ascending)
	// Only completed participants get rankings
	var completedParticipants []models.ChallengeParticipant
	for _, p := range participants {
		if p.IsComplete && p.FinalScore != nil {
			completedParticipants = append(completedParticipants, p)
		}
	}

	// Simple bubble sort for ranking
	for i := 0; i < len(completedParticipants); i++ {
		for j := i + 1; j < len(completedParticipants); j++ {
			// Sort by score (higher is better)
			if *completedParticipants[j].FinalScore > *completedParticipants[i].FinalScore {
				completedParticipants[i], completedParticipants[j] = completedParticipants[j], completedParticipants[i]
			} else if *completedParticipants[j].FinalScore == *completedParticipants[i].FinalScore {
				// If scores are equal, sort by completion time (earlier is better)
				if completedParticipants[j].CompletedAt != nil && completedParticipants[i].CompletedAt != nil {
					if completedParticipants[j].CompletedAt.Before(*completedParticipants[i].CompletedAt) {
						completedParticipants[i], completedParticipants[j] = completedParticipants[j], completedParticipants[i]
					}
				}
			}
		}
	}

	// Update rankings in database using a single transaction
	if len(completedParticipants) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare the statement once
	stmt, err := tx.Prepare(`UPDATE challenge_participants SET rank_position = ?, final_score = ? WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Execute updates in a single transaction
	for i, p := range completedParticipants {
		rank := i + 1
		if _, err := stmt.Exec(rank, *p.FinalScore, p.ID); err != nil {
			return fmt.Errorf("failed to update ranking for participant %d: %w", p.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rankings: %w", err)
	}

	return nil
}

// GetUserLeaderboardRank gets a user's rank on the leaderboard for a specific game mode and difficulty
func (d *Database) GetUserLeaderboardRank(userID int, gameMode string, difficulty string) (*int, error) {
	// First get the user's best score for this game mode and difficulty
	bestScoreQuery := `
		SELECT MAX(score) as best_score
		FROM leaderboard_entries 
		WHERE user_id = ? AND game_mode = ? AND difficulty = ?
	`

	var bestScore sql.NullInt64
	err := d.db.QueryRow(bestScoreQuery, userID, gameMode, difficulty).Scan(&bestScore)
	if err != nil {
		return nil, fmt.Errorf("failed to get user's best score: %w", err)
	}

	// If user has no scores, return nil (no rank)
	if !bestScore.Valid {
		return nil, nil
	}

	// Now count how many registered users have a better score (registered users only ranking)
	rankQuery := `
		SELECT COUNT(DISTINCT user_id) + 1 as rank
		FROM leaderboard_entries 
		WHERE game_mode = ? AND difficulty = ? 
		AND user_id IS NOT NULL AND user_id > 0
		AND (
			SELECT MAX(score) 
			FROM leaderboard_entries le2 
			WHERE le2.user_id = leaderboard_entries.user_id 
			AND le2.game_mode = ? AND le2.difficulty = ?
		) > ?
	`

	var rank int
	err = d.db.QueryRow(rankQuery, gameMode, difficulty, gameMode, difficulty, bestScore.Int64).Scan(&rank)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate user rank: %w", err)
	}

	return &rank, nil
}

// GetUserOverallLeaderboardRank gets a user's rank against ALL entries (including legacy guest entries)
func (d *Database) GetUserOverallLeaderboardRank(userID int, gameMode string, difficulty string) (*int, error) {
	// First get the user's best score for this game mode and difficulty
	bestScoreQuery := `
		SELECT MAX(score) as best_score
		FROM leaderboard_entries 
		WHERE user_id = ? AND game_mode = ? AND difficulty = ?
	`

	var bestScore sql.NullInt64
	err := d.db.QueryRow(bestScoreQuery, userID, gameMode, difficulty).Scan(&bestScore)
	if err != nil {
		return nil, fmt.Errorf("failed to get user's best score: %w", err)
	}

	// If user has no scores, return nil (no rank)
	if !bestScore.Valid {
		return nil, nil
	}

	// Count how many distinct users (including legacy entries) have a better score
	// For legacy entries without user_id, treat each entry as a unique user
	rankQuery := `
		WITH user_best_scores AS (
			SELECT 
				CASE 
					WHEN user_id IS NULL OR user_id = 0 THEN 'guest_' || id 
					ELSE CAST(user_id AS TEXT) 
				END as unique_user,
				MAX(score) as best_score
			FROM leaderboard_entries 
			WHERE game_mode = ? AND difficulty = ?
			GROUP BY unique_user
		)
		SELECT COUNT(*) + 1 as rank
		FROM user_best_scores 
		WHERE best_score > ?
	`

	var rank int
	err = d.db.QueryRow(rankQuery, gameMode, difficulty, bestScore.Int64).Scan(&rank)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate overall user rank: %w", err)
	}

	return &rank, nil
}

// GetUserLeaderboardStats gets comprehensive leaderboard statistics for a user
func (d *Database) GetUserLeaderboardStats(userID int) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	difficulties := []string{"easy", "hard"}
	gameModes := []string{"challenge", "streak"}

	for _, difficulty := range difficulties {
		for _, gameMode := range gameModes {
			// Get registered users rank
			registeredRank, err := d.GetUserLeaderboardRank(userID, gameMode, difficulty)
			if err != nil {
				return nil, fmt.Errorf("failed to get registered rank for %s %s: %w", gameMode, difficulty, err)
			}

			// Get overall rank (including legacy entries)
			overallRank, err := d.GetUserOverallLeaderboardRank(userID, gameMode, difficulty)
			if err != nil {
				return nil, fmt.Errorf("failed to get overall rank for %s %s: %w", gameMode, difficulty, err)
			}

			// Store registered users rank
			registeredKey := fmt.Sprintf("%s_%s_registered_rank", gameMode, difficulty)
			if registeredRank != nil {
				stats[registeredKey] = *registeredRank
			} else {
				stats[registeredKey] = nil
			}

			// Store overall rank
			overallKey := fmt.Sprintf("%s_%s_overall_rank", gameMode, difficulty)
			if overallRank != nil {
				stats[overallKey] = *overallRank
			} else {
				stats[overallKey] = nil
			}

			// Keep the old key for backward compatibility (registered rank)
			legacyKey := fmt.Sprintf("%s_%s_rank", gameMode, difficulty)
			if registeredRank != nil {
				stats[legacyKey] = *registeredRank
			} else {
				stats[legacyKey] = nil
			}
		}
	}

	return stats, nil
}

// generateSessionToken generates a cryptographically secure random session token
func generateSessionToken() string {
	bytes := make([]byte, 32)
	if _, err := randomReader.Read(bytes); err != nil {
		// Fallback to time-based if crypto/rand fails
		return fmt.Sprintf("%d%d", time.Now().UnixNano(), time.Now().Unix()%10000)
	}
	return hex.EncodeToString(bytes)
}
