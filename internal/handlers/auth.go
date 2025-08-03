package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"autotraderguesser/internal/database"
	"autotraderguesser/internal/models"
	"autotraderguesser/internal/validation"
)

type AuthHandler struct {
	db *database.Database
}

func NewAuthHandler(db *database.Database) *AuthHandler {
	return &AuthHandler{db: db}
}

// Registration and Login requests
type RegisterRequest struct {
	Username         string `json:"username" binding:"required,min=3,max=20"`
	Password         string `json:"password" binding:"required,min=6"`
	DisplayName      string `json:"displayName" binding:"required,min=1,max=30"`
	SecurityQuestion string `json:"securityQuestion" binding:"required,min=5,max=200"`
	SecurityAnswer   string `json:"securityAnswer" binding:"required,min=2,max=100"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Success      bool         `json:"success"`
	Message      string       `json:"message"`
	User         *models.User `json:"user,omitempty"`
	SessionToken string       `json:"sessionToken,omitempty"`
}

// Register creates a new user account
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Message: "Invalid request data",
		})
		return
	}

	// Validate input formats
	if err := validation.ValidateUsername(req.Username); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	if err := validation.ValidateDisplayName(req.DisplayName); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	// Check if username already exists
	existingUser, _ := h.db.GetUserByUsername(req.Username)
	if existingUser != nil {
		c.JSON(http.StatusConflict, AuthResponse{
			Success: false,
			Message: "Username already exists",
		})
		return
	}

	// Check if display name already exists
	existingDisplayName, _ := h.db.GetUserByDisplayName(req.DisplayName)
	if existingDisplayName != nil {
		c.JSON(http.StatusConflict, AuthResponse{
			Success: false,
			Message: "Display name already exists",
		})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Message: "Failed to process password",
		})
		return
	}

	// Hash security answer (normalize to lowercase for case-insensitive comparison)
	hashedSecurityAnswer, err := bcrypt.GenerateFromPassword([]byte(strings.ToLower(strings.TrimSpace(req.SecurityAnswer))), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Message: "Failed to process security answer",
		})
		return
	}

	// Generate session token
	sessionToken := generateSessionToken()

	user := &models.User{
		Username:           req.Username,
		PasswordHash:       string(hashedPassword),
		DisplayName:        req.DisplayName,
		SecurityQuestion:   req.SecurityQuestion,
		SecurityAnswerHash: string(hashedSecurityAnswer),
		IsGuest:            false,
		SessionToken:       sessionToken,
	}

	// Create user in database
	if err := h.db.CreateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Message: "Failed to create user account",
		})
		return
	}

	// Set session cookie
	c.SetCookie("session_token", sessionToken, 86400*30, "/", "", false, true) // 30 days for registered users

	// Don't return password hash
	user.PasswordHash = ""

	c.JSON(http.StatusCreated, AuthResponse{
		Success:      true,
		Message:      "Account created successfully",
		User:         user,
		SessionToken: sessionToken,
	})
}

// Login authenticates an existing user
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Message: "Invalid request data",
		})
		return
	}

	// Get user by username
	user, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Message: "Invalid username or password",
		})
		return
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Message: "Invalid username or password",
		})
		return
	}

	// Generate new session token
	sessionToken := generateSessionToken()
	user.SessionToken = sessionToken
	user.LastActive = time.Now()

	// Update user session in database
	if err := h.db.UpdateUserSession(user.ID, sessionToken); err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Message: "Failed to create session",
		})
		return
	}

	// Set session cookie
	c.SetCookie("session_token", sessionToken, 86400*30, "/", "", false, true) // 30 days

	// Don't return password hash
	user.PasswordHash = ""

	c.JSON(http.StatusOK, AuthResponse{
		Success:      true,
		Message:      "Login successful",
		User:         user,
		SessionToken: sessionToken,
	})
}

// Logout removes the user session
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get current user from context (set by middleware)
	user, exists := c.Get("user")
	if exists {
		if u, ok := user.(*models.User); ok {
			// Clear session token in database
			h.db.UpdateUserSession(u.ID, "")
		}
	}

	// Clear session cookie
	c.SetCookie("session_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Logout successful",
	})
}

// GetProfile returns the current user's profile with leaderboard statistics
func (h *AuthHandler) GetProfile(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Message: "Not authenticated",
		})
		return
	}

	u := user.(*models.User)
	// Don't return password hash
	u.PasswordHash = ""

	// Get leaderboard statistics
	leaderboardStats, err := h.db.GetUserLeaderboardStats(u.ID)
	if err != nil {
		// Log error but don't fail the request - just return user without stats
		fmt.Printf("Warning: Failed to get leaderboard stats for user %d: %v\n", u.ID, err)
		leaderboardStats = make(map[string]interface{})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"user":             u,
		"leaderboardStats": leaderboardStats,
	})
}

// UpdateProfile allows users to update their profile information
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Message: "Not authenticated",
		})
		return
	}

	var updateReq struct {
		DisplayName string `json:"displayName" binding:"omitempty,min=1,max=30"`
		AvatarURL   string `json:"avatarUrl" binding:"omitempty,url"`
	}

	if err := c.ShouldBindJSON(&updateReq); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Message: "Invalid request data",
		})
		return
	}

	u := user.(*models.User)

	// Update fields if provided
	if updateReq.DisplayName != "" {
		// Check if display name is already taken by another user
		existingDisplayName, _ := h.db.GetUserByDisplayName(updateReq.DisplayName)
		if existingDisplayName != nil && existingDisplayName.ID != u.ID {
			c.JSON(http.StatusConflict, AuthResponse{
				Success: false,
				Message: "Display name already exists",
			})
			return
		}
		u.DisplayName = updateReq.DisplayName
	}
	if updateReq.AvatarURL != "" {
		u.AvatarURL = updateReq.AvatarURL
	}

	// Update in database
	if err := h.db.UpdateUser(u); err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Message: "Failed to update profile",
		})
		return
	}

	// Don't return password hash
	u.PasswordHash = ""

	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Profile updated successfully",
		User:    u,
	})
}

// ResetPassword allows users to reset their password using security question
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	start := time.Now()
	defer func() {
		// Ensure minimum response time to prevent timing attacks
		elapsed := time.Since(start)
		if elapsed < 200*time.Millisecond {
			time.Sleep(200*time.Millisecond - elapsed)
		}
	}()

	var req models.PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Message: "Invalid request data",
		})
		return
	}

	// Get user by username
	user, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Message: "Invalid credentials",
		})
		return
	}

	// Verify display name matches
	if user.DisplayName != req.DisplayName {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Message: "Invalid credentials",
		})
		return
	}

	// Verify security answer (normalize to lowercase for comparison)
	normalizedAnswer := strings.ToLower(strings.TrimSpace(req.SecurityAnswer))
	if err := bcrypt.CompareHashAndPassword([]byte(user.SecurityAnswerHash), []byte(normalizedAnswer)); err != nil {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Message: "Invalid security answer",
		})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Message: "Failed to process new password",
		})
		return
	}

	// Update password in database
	if err := h.db.UpdateUserPassword(user.ID, string(hashedPassword)); err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Message: "Failed to update password",
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Password reset successfully",
	})
}

// GetSecurityQuestion returns a user's security question for password reset
func (h *AuthHandler) GetSecurityQuestion(c *gin.Context) {
	var req struct {
		Username    string `json:"username" binding:"required"`
		DisplayName string `json:"displayName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request data",
		})
		return
	}

	// Get user by username
	user, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "User not found",
		})
		return
	}

	// Verify display name matches (case-insensitive)
	if strings.ToLower(user.DisplayName) != strings.ToLower(req.DisplayName) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "User not found",
		})
		return
	}

	// Return only the security question (never the answer hash)
	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"securityQuestion": user.SecurityQuestion,
	})
}

// AuthMiddleware validates session tokens and sets user context
func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get session token from cookie first, then header
		sessionToken, err := c.Cookie("session_token")
		if err != nil || sessionToken == "" {
			// Try Authorization header as fallback
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				sessionToken = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if sessionToken == "" {
			c.Next() // Continue without user context
			return
		}

		// Get user by session token
		user, err := h.db.GetUserBySessionToken(sessionToken)
		if err != nil {
			c.Next() // Continue without user context
			return
		}

		// Update last active time
		user.LastActive = time.Now()
		h.db.UpdateUserLastActive(user.ID)

		// Set user in context
		c.Set("user", user)
		c.Next()
	}
}

// RequireAuth middleware that requires authentication
func (h *AuthHandler) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Authentication required",
			})
			c.Abort()
			return
		}

		// Ensure user is not nil
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Invalid session",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// generateSessionToken creates a cryptographically secure session token
func generateSessionToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
