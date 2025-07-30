// AutoTrader Price Guesser Game API
// @title AutoTrader Price Guesser API
// @version 1.0
// @description A fun game API where players guess car prices from real AutoTrader UK listings
// @host localhost:8080
// @BasePath /

package main

import (
	"log"
	"net/http"
	"os"

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
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept"}
	r.Use(cors.New(config))

	// Serve static files
	r.Static("/static", "./static")
	r.StaticFile("/", "./static/index.html")

	// Initialize game handler
	gameHandler := game.NewHandler()

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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
