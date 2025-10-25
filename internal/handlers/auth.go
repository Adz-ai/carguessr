package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"autotraderguesser/internal/database"
	"autotraderguesser/internal/models"
	"autotraderguesser/internal/validation"
)

const (
	// bcryptCost is the work factor for password hashing
	// 12 provides good security/performance balance in 2025 (4096 rounds)
	bcryptCost = 12
)

// isSecureCookieEnabled checks if cookies should be marked as Secure (HTTPS only)
// Returns true in production or when HTTPS_ENABLED=true environment variable is set
func isSecureCookieEnabled() bool {
	// Check explicit HTTPS_ENABLED environment variable
	if os.Getenv("HTTPS_ENABLED") == "true" {
		return true
	}
	// In release mode, assume HTTPS is enabled
	if os.Getenv("GIN_MODE") == "release" {
		return true
	}
	// Default to false for development
	return false
}

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

// Register godoc
// @Summary Register a new user account
// @Description Creates a new user account with username, password, display name, and security question for password recovery. Password is hashed with bcrypt.
// @Tags auth
// @Accept json
// @Produce json
// @Param registration body RegisterRequest true "Registration data"
// @Success 201 {object} AuthResponse "Account created successfully"
// @Failure 400 {object} AuthResponse "Invalid request data or validation failed"
// @Failure 409 {object} AuthResponse "Username or display name already exists"
// @Failure 500 {object} AuthResponse "Failed to create user account"
// @Router /api/auth/register [post]
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

	// Hash password with secure cost factor
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Message: "Failed to process password",
		})
		return
	}

	// Hash security answer (normalize to lowercase for case-insensitive comparison)
	hashedSecurityAnswer, err := bcrypt.GenerateFromPassword([]byte(strings.ToLower(strings.TrimSpace(req.SecurityAnswer))), bcryptCost)
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

	// Set session cookie with SameSite protection (7 days to match server expiration)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		MaxAge:   86400 * 7, // 7 days
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureCookieEnabled(), // Automatically enabled in production/HTTPS
		SameSite: http.SameSiteLaxMode,
	})

	// Don't return password hash
	user.PasswordHash = ""

	c.JSON(http.StatusCreated, AuthResponse{
		Success:      true,
		Message:      "Account created successfully",
		User:         user,
		SessionToken: sessionToken,
	})
}

// Login godoc
// @Summary Login to an existing account
// @Description Authenticates a user with username and password. Returns a session token valid for 7 days. Automatically upgrades password hashes to current security standards.
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body LoginRequest true "Login credentials"
// @Success 200 {object} AuthResponse "Login successful"
// @Failure 400 {object} AuthResponse "Invalid request data"
// @Failure 401 {object} AuthResponse "Invalid username or password"
// @Failure 500 {object} AuthResponse "Failed to create session"
// @Router /api/auth/login [post]
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

	// Automatic hash upgrading: if the stored hash uses an older cost factor, rehash with current cost
	currentHashCost, err := bcrypt.Cost([]byte(user.PasswordHash))
	if err == nil && currentHashCost < bcryptCost {
		// Password verified successfully, but using old cost - upgrade it
		newHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
		if err == nil {
			// Update password with new hash (ignore errors - session creation is more important)
			if err := h.db.UpdateUserPassword(user.ID, string(newHash)); err == nil {
				fmt.Printf("Upgraded password hash for user %s from cost %d to %d\n", user.Username, currentHashCost, bcryptCost)
			}
		}
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

	// Set session cookie with SameSite protection (7 days to match server expiration)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		MaxAge:   86400 * 7, // 7 days
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureCookieEnabled(), // Automatically enabled in production/HTTPS
		SameSite: http.SameSiteLaxMode,
	})

	// Don't return password hash
	user.PasswordHash = ""

	c.JSON(http.StatusOK, AuthResponse{
		Success:      true,
		Message:      "Login successful",
		User:         user,
		SessionToken: sessionToken,
	})
}

// Logout godoc
// @Summary Logout from current session
// @Description Invalidates the current session token and clears the session cookie. Can be called without authentication.
// @Tags auth
// @Produce json
// @Success 200 {object} AuthResponse "Logout successful"
// @Router /api/auth/logout [post]
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
	c.SetCookie("session_token", "", -1, "/", "", isSecureCookieEnabled(), true)

	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Logout successful",
	})
}

// GetProfile godoc
// @Summary Get current user profile
// @Description Returns the authenticated user's profile information including leaderboard statistics and rankings. Requires authentication via session token.
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "user: User object, leaderboardStats: rankings and stats"
// @Failure 401 {object} AuthResponse "Not authenticated"
// @Router /api/auth/profile [get]
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

// UpdateProfile godoc
// @Summary Update user profile
// @Description Allows authenticated users to update their display name or avatar URL. Display names must be unique.
// @Tags auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param profile body object{displayName=string,avatarUrl=string} true "Profile update data"
// @Success 200 {object} AuthResponse "Profile updated successfully"
// @Failure 400 {object} AuthResponse "Invalid request data"
// @Failure 401 {object} AuthResponse "Not authenticated"
// @Failure 409 {object} AuthResponse "Display name already exists"
// @Failure 500 {object} AuthResponse "Failed to update profile"
// @Router /api/auth/profile [put]
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

// ResetPassword godoc
// @Summary Reset user password
// @Description Resets a user's password by verifying their username, display name, and security question answer. Includes timing attack protection.
// @Tags auth
// @Accept json
// @Produce json
// @Param reset body models.PasswordResetRequest true "Password reset data"
// @Success 200 {object} AuthResponse "Password reset successfully"
// @Failure 400 {object} AuthResponse "Invalid request data"
// @Failure 401 {object} AuthResponse "Invalid credentials or security answer"
// @Failure 500 {object} AuthResponse "Failed to update password"
// @Router /api/auth/reset-password [post]
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
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcryptCost)
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

// GetSecurityQuestion godoc
// @Summary Get security question for password reset
// @Description Returns a user's security question after verifying their username and display name. Used for password reset flow.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body object{username=string,displayName=string} true "Username and display name"
// @Success 200 {object} map[string]interface{} "success: true, securityQuestion: string"
// @Failure 400 {object} map[string]string "Invalid request data"
// @Failure 404 {object} map[string]string "User not found"
// @Router /api/auth/security-question [post]
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

		// Check if session has expired
		if user.SessionExpiresAt != nil && time.Now().After(*user.SessionExpiresAt) {
			// Session expired - clear the cookie and reject
			c.SetCookie("session_token", "", -1, "/", "", false, true)
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
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based token if crypto/rand fails
		// This is not ideal but better than returning an error during auth
		return fmt.Sprintf("fallback_%d_%d", time.Now().UnixNano(), time.Now().Unix()%10000)
	}
	return hex.EncodeToString(bytes)
}
