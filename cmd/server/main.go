// CarGuessr Game API
// @title CarGuessr API
// @version 2.1
// @description A fun car price guessing game with multiple game modes using real Bonhams Car Auction data. Now with enhanced security, rate limiting, and 250 cars with 7-day refresh cycles.
// @termsOfService https://carguessr.uk
//
// @contact.name CarGuessr Support
// @contact.url https://github.com/your-repo/carguessr/issues
//
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
//
// @host localhost:8080
// @BasePath /
// @schemes http https
//
// @securityDefinitions.apikey AdminKey
// @in header
// @name X-Admin-Key
// @description Admin API key required for administrative endpoints. Can also be passed as 'admin_key' query parameter.
//
// @tag.name game
// @tag.description Core game endpoints for different game modes (Rate limited: 60 req/min)
// @tag.name challenge
// @tag.description Challenge Mode - GeoGuessr style scoring with 10 cars (Rate limited: 60 req/min)
// @tag.name admin
// @tag.description Administrative functions requiring authentication (Rate limited: 2 req/min)
// @tag.name public
// @tag.description Public endpoints with general rate limiting

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/time/rate"

	_ "autotraderguesser/docs"
	"autotraderguesser/internal/database"
	"autotraderguesser/internal/game"
	"autotraderguesser/internal/handlers"
	"autotraderguesser/internal/middleware"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize Gin router
	r := gin.Default()

	// Limit request body size (1MB max)
	r.MaxMultipartMemory = 1 << 20 // 1MB

	// Configure trusted proxies for Cloudflare Tunnels
	r.SetTrustedProxies([]string{
		"127.0.0.1",
		"::1",
		"172.16.0.0/12",  // Docker networks
		"10.0.0.0/8",     // Private networks
		"192.168.0.0/16", // Private networks
	})

	// Configure CORS with specific allowed origins for security
	config := cors.DefaultConfig()
	// Allow specific origins instead of wildcard for security
	allowedOrigins := []string{
		"http://localhost:8080",    // Development
		"http://127.0.0.1:8080",    // Development
		"https://carguessr.uk",     // Production domain
		"https://www.carguessr.uk", // Production www subdomain
	}
	config.AllowOrigins = allowedOrigins
	config.AllowMethods = []string{"GET", "POST", "OPTIONS"} // Aligned with HTTPMethodFilter for consistency
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "X-Session-ID"}
	config.ExposeHeaders = []string{"Content-Length", "X-Session-ID"}
	config.AllowCredentials = true
	config.MaxAge = 12 * 3600
	r.Use(cors.New(config))

	// Add security headers
	r.Use(middleware.SecurityHeaders())

	// Add security scan detection (for fail2ban)
	r.Use(middleware.SecurityScanDetection())

	// Add HTTP method filtering (only allow GET and POST)
	r.Use(middleware.HTTPMethodFilter([]string{"GET", "POST", "OPTIONS"}))

	// Add user agent filtering
	r.Use(middleware.UserAgentFilter())

	// Add honeypot endpoints
	r.Use(middleware.HoneypotEndpoints())

	// Add request logging middleware for debugging
	r.Use(func(c *gin.Context) {
		log.Printf("Request: %s %s from %s", c.Request.Method, c.Request.URL.Path, c.ClientIP())
		c.Next()
	})

	// Create rate limiters
	// General API rate limiter: 60 requests per minute
	generalLimiter := middleware.NewRateLimiter(rate.Limit(1), 60) // 1 request/second, burst of 60

	// Auth rate limiter: 10 requests per minute (prevent brute force)
	authLimiter := middleware.NewRateLimiter(rate.Limit(0.17), 10) // ~10 requests/minute

	// Strict rate limiter for expensive operations: 2 requests per minute
	strictLimiter := middleware.NewRateLimiter(rate.Limit(0.033), 2) // 2 requests/minute

	// Get admin key from environment - REQUIRED for security
	adminKey := os.Getenv("ADMIN_KEY")
	if adminKey == "" {
		log.Fatal("❌ ADMIN_KEY environment variable not set. Cannot start without proper admin key. Set ADMIN_KEY in your .env file or environment variables.")
	}

	// Validate admin key strength
	if err := validateAdminKey(adminKey); err != nil {
		log.Fatalf("❌ ADMIN_KEY validation failed: %v", err)
	}

	log.Println("✅ Admin key validated successfully")

	// Serve static files with no-cache headers to prevent Cloudflare caching issues
	r.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}
		c.Next()
	})

	r.Static("/static", "./static")
	r.StaticFile("/", "./static/index.html")

	// SEO files at root level
	r.StaticFile("/sitemap.xml", "./static/sitemap.xml")
	r.StaticFile("/robots.txt", "./static/robots.txt")
	r.StaticFile("/favicon.ico", "./static/favicon_io/favicon.ico")

	// Initialize database
	dbPath := "./data/carguessr.db"
	db, err := database.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Println("✅ Database initialized successfully")

	// Initialize handlers
	gameHandler := game.NewHandler(db)
	authHandler := handlers.NewAuthHandler(db)
	friendsHandler := handlers.NewFriendsHandler(db, gameHandler)

	// Swagger documentation (only in development mode)
	if gin.Mode() != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		log.Println("📚 Swagger documentation available at /swagger/index.html")
	}

	// Public API routes with general rate limiting
	api := r.Group("/api")
	api.Use(middleware.RateLimitMiddleware(generalLimiter))
	api.Use(authHandler.AuthMiddleware()) // Optional authentication - adds user context if authenticated
	{
		// Authentication endpoints with stricter rate limiting (prevent brute force)
		auth := api.Group("/auth")
		auth.Use(middleware.RateLimitMiddleware(authLimiter))
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.POST("/reset-password", authHandler.ResetPassword)
			auth.POST("/security-question", authHandler.GetSecurityQuestion)
			auth.GET("/profile", authHandler.RequireAuth(), authHandler.GetProfile)
			auth.PUT("/profile", authHandler.RequireAuth(), authHandler.UpdateProfile)
		}

		// Game endpoints
		api.GET("/random-listing", gameHandler.GetRandomListing)
		api.GET("/random-enhanced-listing", gameHandler.GetRandomEnhancedListing)
		api.POST("/check-guess", gameHandler.CheckGuess)
		api.GET("/leaderboard", gameHandler.GetLeaderboard)
		api.POST("/leaderboard/submit", gameHandler.SubmitScore)
		api.GET("/data-source", gameHandler.GetDataSource)

		// Challenge Mode routes
		api.POST("/challenge/start", gameHandler.StartChallenge)
		api.GET("/challenge/:sessionId", gameHandler.GetChallengeSession)
		api.POST("/challenge/:sessionId/guess", gameHandler.SubmitChallengeGuess)

		// Friend Challenge routes (require authentication)
		api.POST("/friends/challenges", friendsHandler.CreateFriendChallenge)
		api.GET("/friends/challenges/:code", friendsHandler.GetFriendChallenge)
		api.POST("/friends/challenges/:code/join", friendsHandler.JoinFriendChallenge)
		api.GET("/friends/challenges/:code/leaderboard", friendsHandler.GetChallengeLeaderboard)
		api.GET("/friends/challenges/:code/participation", friendsHandler.GetUserParticipation)
		api.GET("/friends/challenges/my-challenges", friendsHandler.GetMyChallenges)

		// Health check (no additional rate limiting)
		api.GET("/health", func(c *gin.Context) {
			health := gin.H{
				"status": "ok",
				"checks": gin.H{},
			}

			// Check database connectivity
			if err := db.Ping(); err != nil {
				health["status"] = "degraded"
				health["checks"].(gin.H)["database"] = "unhealthy"
			} else {
				health["checks"].(gin.H)["database"] = "healthy"
			}

			// Check if data sources are available
			dataSource := gameHandler.GetDataSourceInfo()
			if dataSource["total_listings"].(int) == 0 {
				health["status"] = "degraded"
				health["checks"].(gin.H)["listings"] = "no data available"
			} else {
				health["checks"].(gin.H)["listings"] = "healthy"
			}

			// Return appropriate status code
			if health["status"] == "degraded" {
				c.JSON(http.StatusServiceUnavailable, health)
			} else {
				c.JSON(http.StatusOK, health)
			}
		})
	}

	// Admin routes - protected with admin key and strict rate limiting
	admin := r.Group("/api/admin")
	admin.Use(middleware.AdminKeyMiddleware(adminKey))
	admin.Use(middleware.RateLimitMiddleware(strictLimiter))
	{
		admin.POST("/refresh-listings", middleware.RefreshProtectionMiddleware(), gameHandler.ManualRefresh)
		admin.GET("/cache-status", gameHandler.GetCacheStatus)
		admin.GET("/leaderboard-status", gameHandler.GetLeaderboardStatus)
		admin.GET("/listings", gameHandler.GetAllListings)
		admin.GET("/test-scraper", gameHandler.TestScraper)
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP server with explicit configuration
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Println("🛑 Shutting down server gracefully...")

	// Stop auto-refresh tickers
	gameHandler.StopAutoRefresh()

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Close database connection
	db.Close()

	log.Println("✅ Server shutdown complete")
}

// validateAdminKey performs comprehensive admin key validation
// Checks for length, obvious patterns, and cryptographic entropy
func validateAdminKey(key string) error {
	// Minimum length check
	if len(key) < 32 {
		return fmt.Errorf("admin key must be at least 32 characters (got %d)", len(key))
	}

	// Check for obvious weak patterns
	lowercaseKey := strings.ToLower(key)
	weakPatterns := []string{"change-this", "default", "temp", "test", "admin", "password", "secret"}
	for _, pattern := range weakPatterns {
		if strings.Contains(lowercaseKey, pattern) {
			return fmt.Errorf("admin key contains weak pattern: %s", pattern)
		}
	}

	// Check for repetitive characters (e.g., "aaaaaaa...")
	if hasExcessiveRepetition(key) {
		return fmt.Errorf("admin key has excessive character repetition - use a more random key")
	}

	// Check character variety (must contain at least 3 different character types)
	charTypes := 0
	hasLower := strings.ContainsAny(key, "abcdefghijklmnopqrstuvwxyz")
	hasUpper := strings.ContainsAny(key, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	hasDigit := strings.ContainsAny(key, "0123456789")
	hasSpecial := strings.ContainsAny(key, "!@#$%^&*()_+-=[]{}|;:,.<>?/~`")

	if hasLower {
		charTypes++
	}
	if hasUpper {
		charTypes++
	}
	if hasDigit {
		charTypes++
	}
	if hasSpecial {
		charTypes++
	}

	if charTypes < 3 {
		return fmt.Errorf("admin key must contain at least 3 different character types (lowercase, uppercase, digits, special)")
	}

	// Check for unique character count (must have at least 16 unique characters)
	uniqueChars := make(map[rune]bool)
	for _, char := range key {
		uniqueChars[char] = true
	}
	if len(uniqueChars) < 16 {
		return fmt.Errorf("admin key must have at least 16 unique characters (got %d)", len(uniqueChars))
	}

	return nil
}

// hasExcessiveRepetition checks if a string has excessive character repetition
// Returns true if any character appears more than 30% of the time
func hasExcessiveRepetition(s string) bool {
	if len(s) == 0 {
		return false
	}

	charCount := make(map[rune]int)
	for _, char := range s {
		charCount[char]++
	}

	threshold := len(s) * 3 / 10 // 30%
	for _, count := range charCount {
		if count > threshold {
			return true
		}
	}

	return false
}
