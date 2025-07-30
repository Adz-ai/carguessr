// Motors Price Guesser Game API
// @title Motors Price Guesser API
// @version 2.0
// @description A fun car price guessing game with multiple game modes using real Bonhams Car Auction data
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
// @tag.name game
// @tag.description Core game endpoints for different game modes
// @tag.name challenge
// @tag.description Challenge Mode - GeoGuessr style scoring with 10 cars
// @tag.name listings
// @tag.description Car listing management and data access
// @tag.name admin
// @tag.description Administrative functions for cache and refresh
// @tag.name debug
// @tag.description Debug and monitoring endpoints

package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "autotraderguesser/docs"
	"autotraderguesser/internal/game"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize Gin router
	r := gin.Default()

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

	// Add request logging middleware for debugging
	r.Use(func(c *gin.Context) {
		log.Printf("Request: %s %s from %s", c.Request.Method, c.Request.URL.Path, c.ClientIP())
		c.Next()
	})

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

	// Initialize game handler
	gameHandler := game.NewHandler()

	// Swagger documentation (only in development mode)
	if gin.Mode() != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		log.Println("📚 Swagger documentation available at /swagger/index.html")
	}

	// API routes
	api := r.Group("/api")
	{
		api.GET("/random-listing", gameHandler.GetRandomListing)
		api.GET("/random-enhanced-listing", gameHandler.GetRandomEnhancedListing)
		api.POST("/check-guess", gameHandler.CheckGuess)
		api.GET("/leaderboard", gameHandler.GetLeaderboard)
		api.GET("/listings", gameHandler.GetAllListings)
		api.GET("/test-scraper", gameHandler.TestScraper)
		api.GET("/data-source", gameHandler.GetDataSource)
		api.POST("/refresh-listings", gameHandler.ManualRefresh)
		api.GET("/cache-status", gameHandler.GetCacheStatus)

		// Challenge Mode routes
		api.POST("/challenge/start", gameHandler.StartChallenge)
		api.GET("/challenge/:sessionId", gameHandler.GetChallengeSession)
		api.POST("/challenge/:sessionId/guess", gameHandler.SubmitChallengeGuess)

		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
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
