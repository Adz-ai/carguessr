// Motors Price Guesser Game API
// @title Motors Price Guesser API
// @version 2.1
// @description A fun car price guessing game with multiple game modes using real Bonhams Car Auction data. Now with enhanced security, rate limiting, and 250 cars with 7-day refresh cycles.
// @termsOfService https://github.com/your-repo/motors-price-guesser
//
// @contact.name Motors Price Guesser Support
// @contact.url https://github.com/your-repo/motors-price-guesser/issues
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
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/time/rate"

	_ "autotraderguesser/docs"
	"autotraderguesser/internal/game"
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

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
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

	// Strict rate limiter for expensive operations: 2 requests per minute
	strictLimiter := middleware.NewRateLimiter(rate.Limit(0.033), 2) // 2 requests/minute

	// Get admin key from environment or use a default (should be changed in production)
	adminKey := os.Getenv("ADMIN_KEY")
	if adminKey == "" {
		adminKey = "change-this-in-production-" + strings.ToUpper(strings.ReplaceAll(generateSessionID(), "-", ""))
		log.Printf("⚠️  No ADMIN_KEY set in environment. Generated temporary key: %s", adminKey)
		log.Println("⚠️  Please set ADMIN_KEY environment variable for production use!")
	}

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

	// Initialize game handler
	gameHandler := game.NewHandler()

	// Swagger documentation (only in development mode)
	if gin.Mode() != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		log.Println("📚 Swagger documentation available at /swagger/index.html")
	}

	// Public API routes with general rate limiting
	api := r.Group("/api")
	api.Use(middleware.RateLimitMiddleware(generalLimiter))
	{
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

		// Health check (no additional rate limiting)
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
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

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// generateSessionID creates a random session ID
func generateSessionID() string {
	rand.Seed(time.Now().UnixNano())
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
