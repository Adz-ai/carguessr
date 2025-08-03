package models

import "time"

// User represents a user account
type User struct {
	ID                 int       `json:"id" db:"id"`
	Username           string    `json:"username" db:"username"`
	Email              string    `json:"email,omitempty" db:"email"`
	PasswordHash       string    `json:"-" db:"password_hash"` // Never include in JSON responses
	DisplayName        string    `json:"displayName" db:"display_name"`
	AvatarURL          string    `json:"avatarUrl,omitempty" db:"avatar_url"`
	IsGuest            bool      `json:"isGuest" db:"is_guest"`
	SessionToken       string    `json:"sessionToken,omitempty" db:"session_token"`
	CreatedAt          time.Time `json:"createdAt" db:"created_at"`
	LastActive         time.Time `json:"lastActive" db:"last_active"`
	TotalGamesPlayed   int       `json:"totalGamesPlayed" db:"total_games_played"`
	FavoriteDifficulty string    `json:"favoriteDifficulty" db:"favorite_difficulty"`
}

// UserRegistrationRequest for creating new users
type UserRegistrationRequest struct {
	Username    string `json:"username" binding:"required,min=2,max=20"`
	DisplayName string `json:"displayName" binding:"required,min=1,max=50"`
	Email       string `json:"email,omitempty" binding:"omitempty,email"`
}

// UserStats represents aggregated user statistics
type UserStats struct {
	UserID              int     `json:"userId"`
	TotalGamesPlayed    int     `json:"totalGamesPlayed"`
	BestChallengeScore  int     `json:"bestChallengeScore"`
	BestStreakScore     int     `json:"bestStreakScore"`
	AverageAccuracy     float64 `json:"averageAccuracy"`
	FavoriteGameMode    string  `json:"favoriteGameMode"`
	FavoriteDifficulty  string  `json:"favoriteDifficulty"`
	TotalPlayTimeHours  float64 `json:"totalPlayTimeHours"`
	LeaderboardEntries  int     `json:"leaderboardEntries"`
}

// FriendChallenge represents a multiplayer challenge
type FriendChallenge struct {
	ID                  int                        `json:"id" db:"id"`
	ChallengeCode       string                     `json:"challengeCode" db:"challenge_code"`
	Title               string                     `json:"title" db:"title"`
	CreatorUserID       int                        `json:"creatorUserId" db:"creator_user_id"`
	TemplateSessionID   string                     `json:"templateSessionId" db:"template_session_id"`
	Difficulty          string                     `json:"difficulty" db:"difficulty"`
	MaxParticipants     int                        `json:"maxParticipants" db:"max_participants"`
	IsActive            bool                       `json:"isActive" db:"is_active"`
	CreatedAt           time.Time                  `json:"createdAt" db:"created_at"`
	ExpiresAt           time.Time                  `json:"expiresAt" db:"expires_at"`
	Participants        []ChallengeParticipant     `json:"participants,omitempty"`
	CreatorDisplayName  string                     `json:"creatorDisplayName,omitempty"` // Populated in queries
	ParticipantCount    int                        `json:"participantCount,omitempty"`   // Populated in queries
	JoinedAt            *time.Time                 `json:"joinedAt,omitempty"`           // For participating challenges - when user joined
	IsComplete          bool                       `json:"isComplete,omitempty"`         // For participating challenges - completion status
}

// ChallengeParticipant represents a user participating in a friend challenge
type ChallengeParticipant struct {
	ID                 int       `json:"id" db:"id"`
	FriendChallengeID  int       `json:"friendChallengeId" db:"friend_challenge_id"`
	UserID             int       `json:"userId" db:"user_id"`
	SessionID          string    `json:"sessionId" db:"session_id"`
	FinalScore         *int      `json:"finalScore,omitempty" db:"final_score"`
	RankPosition       *int      `json:"rankPosition,omitempty" db:"rank_position"`
	CompletedAt        *time.Time `json:"completedAt,omitempty" db:"completed_at"`
	JoinedAt           time.Time `json:"joinedAt" db:"joined_at"`
	UserDisplayName    string    `json:"userDisplayName,omitempty"` // Populated in queries
	IsComplete         bool      `json:"isComplete"`                // Calculated field
}

// CreateFriendChallengeRequest for creating new friend challenges
type CreateFriendChallengeRequest struct {
	Title      string `json:"title" binding:"required,min=1,max=100"`
	Difficulty string `json:"difficulty" binding:"required,oneof=easy hard"`
	MaxParticipants int `json:"maxParticipants" binding:"min=2,max=50"`
}

// JoinFriendChallengeRequest for joining friend challenges
type JoinFriendChallengeRequest struct {
	ChallengeCode string `json:"challengeCode" binding:"required,len=6"`
}

// GameSession represents a streak or zero mode session
type GameSession struct {
	SessionID        string    `json:"sessionId" db:"session_id"`
	UserID           *int      `json:"userId,omitempty" db:"user_id"`
	GameMode         string    `json:"gameMode" db:"game_mode"`
	Difficulty       string    `json:"difficulty" db:"difficulty"`
	CurrentScore     int       `json:"currentScore" db:"current_score"`
	CurrentDifference float64  `json:"currentDifference" db:"current_difference"`
	CarsShown        []string  `json:"carsShown" db:"-"` // Parsed from JSON
	CarsShownJSON    string    `json:"-" db:"cars_shown"` // Raw JSON for database
	IsActive         bool      `json:"isActive" db:"is_active"`
	CreatedAt        time.Time `json:"createdAt" db:"created_at"`
	LastGuessAt      time.Time `json:"lastGuessAt" db:"last_guess_at"`
}