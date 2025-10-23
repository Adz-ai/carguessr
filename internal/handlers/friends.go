package handlers

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"autotraderguesser/internal/database"
	"autotraderguesser/internal/models"
	"autotraderguesser/internal/util"
	"autotraderguesser/internal/validation"
)

type FriendsHandler struct {
	db          *database.Database
	gameHandler GameHandlerInterface // Interface for game operations
}

var challengeRandReader io.Reader = rand.Reader
var sessionRandReader io.Reader = rand.Reader

// GameHandlerInterface defines the methods we need from game handler
type GameHandlerInterface interface {
	CreateTemplateChallenge(difficulty string, userID int) (*models.ChallengeSession, error)
}

// NewFriendsHandler wires the database and game bridge used for friend challenges.
func NewFriendsHandler(db *database.Database, gameHandler GameHandlerInterface) *FriendsHandler {
	return &FriendsHandler{
		db:          db,
		gameHandler: gameHandler,
	}
}

// CreateFriendChallenge creates a new friend challenge
func (h *FriendsHandler) CreateFriendChallenge(c *gin.Context) {
	// Require authentication
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Authentication required to create challenges",
		})
		return
	}

	u := user.(*models.User)

	var req models.CreateFriendChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	// Validate and sanitize challenge title
	sanitizedTitle, err := validation.ValidateChallengeTitle(req.Title)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	req.Title = sanitizedTitle

	// Generate unique 6-character challenge code
	challengeCode := generateChallengeCode()

	// Ensure uniqueness (retry if collision)
	for attempts := 0; attempts < 5; attempts++ {
		if exists, err := h.db.ChallengeCodeExists(challengeCode); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Failed to validate challenge code",
			})
			return
		} else if !exists {
			break
		}
		challengeCode = generateChallengeCode()
	}

	// Create template challenge session (10 cars for consistent gameplay)
	templateSession, err := h.gameHandler.CreateTemplateChallenge(req.Difficulty, u.ID)
	if err != nil {
		util.SafeErrorResponse(c, http.StatusInternalServerError, "Failed to create challenge template", err)
		return
	}

	// Create friend challenge
	challenge := &models.FriendChallenge{
		ChallengeCode:     challengeCode,
		Title:             req.Title,
		CreatorUserID:     u.ID,
		TemplateSessionID: templateSession.SessionID,
		Difficulty:        req.Difficulty,
		MaxParticipants:   req.MaxParticipants,
		IsActive:          true,
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(48 * time.Hour), // 48 hours to complete
	}

	if err := h.db.CreateFriendChallenge(challenge); err != nil {
		util.SafeErrorResponse(c, http.StatusInternalServerError, "Failed to create friend challenge", err)
		return
	}

	// Add creator as first participant
	participant := &models.ChallengeParticipant{
		FriendChallengeID: challenge.ID,
		UserID:            u.ID,
		SessionID:         templateSession.SessionID,
		JoinedAt:          time.Now(),
	}

	if err := h.db.AddChallengeParticipant(participant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to add creator as participant",
		})
		return
	}

	// Return challenge details including creator's session ID
	c.JSON(http.StatusCreated, gin.H{
		"success":          true,
		"message":          "Friend challenge created successfully!",
		"challenge":        challenge,
		"challengeCode":    challengeCode,
		"sessionId":        templateSession.SessionID, // Add creator's session ID
		"participantCount": 1,
		"shareMessage":     fmt.Sprintf("Join my CarGuessr challenge '%s'! Use code: %s", req.Title, challengeCode),
	})
}

// GetFriendChallenge gets challenge details by code
func (h *FriendsHandler) GetFriendChallenge(c *gin.Context) {
	challengeCode := strings.ToUpper(c.Param("code"))

	// Validate challenge code format
	if err := validation.ValidateChallengeCode(challengeCode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid challenge code format",
		})
		return
	}

	challenge, err := h.db.GetFriendChallengeByCode(challengeCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Challenge not found or expired",
		})
		return
	}

	// Get participants with their completion status
	participants, err := h.db.GetChallengeParticipants(challenge.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get participants",
		})
		return
	}

	challenge.Participants = participants
	challenge.ParticipantCount = len(participants)

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"challenge": challenge,
	})
}

// JoinFriendChallenge allows a user to join an existing challenge
func (h *FriendsHandler) JoinFriendChallenge(c *gin.Context) {
	// Require authentication
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Authentication required to join challenges",
		})
		return
	}

	u := user.(*models.User)
	challengeCode := strings.ToUpper(c.Param("code"))

	// Validate challenge code format
	if err := validation.ValidateChallengeCode(challengeCode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid challenge code format",
		})
		return
	}

	challenge, err := h.db.GetFriendChallengeByCode(challengeCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Challenge not found or expired",
		})
		return
	}

	// Check if challenge is still active
	if !challenge.IsActive || time.Now().After(challenge.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Challenge has expired or is no longer active",
		})
		return
	}

	// Check if user is already participating
	participants, err := h.db.GetChallengeParticipants(challenge.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to check participants",
		})
		return
	}

	for _, participant := range participants {
		if participant.UserID == u.ID {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "You are already participating in this challenge",
			})
			return
		}
	}

	// Check if challenge is full
	if len(participants) >= challenge.MaxParticipants {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Challenge is full",
		})
		return
	}

	// Create a new challenge session for this participant (same cars as template)
	templateSession, err := h.db.GetChallengeSession(challenge.TemplateSessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get challenge template",
		})
		return
	}

	// Create participant's session (copy of template with new session ID)
	participantSession := &models.ChallengeSession{
		SessionID:  generateSessionID(),
		UserID:     u.ID,
		Difficulty: challenge.Difficulty,
		Cars:       templateSession.Cars, // Same cars as template
		CurrentCar: 0,
		Guesses:    []models.ChallengeGuess{},
		TotalScore: 0,
		IsComplete: false,
		StartTime:  time.Now().Format(time.RFC3339),
	}

	if err := h.db.CreateChallengeSession(participantSession); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create participant session",
		})
		return
	}

	// Add participant
	participant := &models.ChallengeParticipant{
		FriendChallengeID: challenge.ID,
		UserID:            u.ID,
		SessionID:         participantSession.SessionID,
		JoinedAt:          time.Now(),
	}

	if err := h.db.AddChallengeParticipant(participant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to join challenge",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   fmt.Sprintf("Successfully joined challenge '%s'!", challenge.Title),
		"sessionId": participantSession.SessionID,
		"challenge": challenge,
	})
}

// GetChallengeLeaderboard gets the leaderboard for a friend challenge
func (h *FriendsHandler) GetChallengeLeaderboard(c *gin.Context) {
	challengeCode := strings.ToUpper(c.Param("code"))

	// Validate challenge code format
	if err := validation.ValidateChallengeCode(challengeCode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid challenge code format",
		})
		return
	}

	challenge, err := h.db.GetFriendChallengeByCode(challengeCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Challenge not found",
		})
		return
	}

	// Get participants with their scores
	participants, err := h.db.GetChallengeParticipants(challenge.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get leaderboard",
		})
		return
	}

	// Update completion status and scores
	for i := range participants {
		session, err := h.db.GetChallengeSession(participants[i].SessionID)
		if err != nil {
			continue
		}

		participants[i].IsComplete = session.IsComplete
		if session.IsComplete {
			participants[i].FinalScore = &session.TotalScore
			if session.CompletedTime != "" {
				if completedTime, err := time.Parse(time.RFC3339, session.CompletedTime); err == nil {
					participants[i].CompletedAt = &completedTime
				}
			}
		}
	}

	// Calculate rankings (sort by score, then by completion time)
	h.db.CalculateChallengeRankings(challenge.ID, participants)

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"challenge":    challenge,
		"participants": participants,
		"totalCount":   len(participants),
	})
}

// GetUserParticipation gets a user's participation in a specific challenge
func (h *FriendsHandler) GetUserParticipation(c *gin.Context) {
	// Require authentication
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Authentication required",
		})
		return
	}

	u := user.(*models.User)
	challengeCode := strings.ToUpper(c.Param("code"))

	// Validate challenge code format
	if err := validation.ValidateChallengeCode(challengeCode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid challenge code format",
		})
		return
	}

	challenge, err := h.db.GetFriendChallengeByCode(challengeCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Challenge not found",
		})
		return
	}

	// Get user's participation
	participant, err := h.db.GetUserChallengeParticipation(challenge.ID, u.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "You are not participating in this challenge",
		})
		return
	}

	// Get session details
	session, err := h.db.GetChallengeSession(participant.SessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get session details",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"challenge":     challenge,
		"participation": participant,
		"session":       session,
	})
}

// GetMyChallenges gets all challenges a user has created or participated in
func (h *FriendsHandler) GetMyChallenges(c *gin.Context) {
	// Require authentication
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Authentication required",
		})
		return
	}

	u := user.(*models.User)

	// Get challenges created by user
	createdChallenges, err := h.db.GetUserCreatedChallenges(u.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get created challenges",
		})
		return
	}

	// Get challenges user is participating in
	participatingChallenges, err := h.db.GetUserParticipatingChallenges(u.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get participating challenges",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"created":       createdChallenges,
		"participating": participatingChallenges,
	})
}

// generateChallengeCode generates a 6-character alphanumeric code
func generateChallengeCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, 6)
	challengeRandReader.Read(bytes)

	for i := range bytes {
		bytes[i] = charset[bytes[i]%byte(len(charset))]
	}

	return string(bytes)
}

// generateSessionID generates a unique session ID (matching game handler format)
func generateSessionID() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, 16)
	sessionRandReader.Read(bytes)
	result := make([]byte, 16)
	for i := range result {
		result[i] = letters[bytes[i]%byte(len(letters))]
	}
	return string(result)
}
