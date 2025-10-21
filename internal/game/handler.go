package game

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"autotraderguesser/internal/cache"
	"autotraderguesser/internal/database"
	"autotraderguesser/internal/models"
	"autotraderguesser/internal/scraper"
	"autotraderguesser/internal/validation"
)

const ListingAmount int = 250

type Handler struct {
	db                   *database.Database // Database connection
	scraper              *scraper.Scraper
	bonhamsListings      map[string]*models.BonhamsCar // Hard mode data
	lookersListings      map[string]*models.LookersCar // Easy mode data
	mu                   sync.RWMutex
	zeroScores           map[string]float64
	streakScores         map[string]int
	challengeSessions    map[string]*models.ChallengeSession // Store challenge sessions
	recentlyShown        map[string][]string                 // Track recently shown car IDs per session
	bonhamsRefreshTicker *time.Ticker                        // Auto-refresh for Bonhams (7 days)
	lookersRefreshTicker *time.Ticker                        // Auto-refresh for Lookers (7 days)
}

// NewHandler creates a game handler, primes both data sources, and starts refresh schedulers.
func NewHandler(db *database.Database) *Handler {
	h := &Handler{
		db:                db,
		scraper:           scraper.New(),
		bonhamsListings:   make(map[string]*models.BonhamsCar),
		lookersListings:   make(map[string]*models.LookersCar),
		zeroScores:        make(map[string]float64),
		streakScores:      make(map[string]int),
		challengeSessions: make(map[string]*models.ChallengeSession),
		recentlyShown:     make(map[string][]string),
	}

	// Initialize both scrapers before starting (both modes must be ready)
	fmt.Println("🚀 Initializing CarGuessr data sources...")
	fmt.Println("   📊 Both Hard Mode (Bonhams) and Easy Mode (Lookers) must be ready before startup")

	h.initializeBonhamsListings()
	h.initializeLookersListings()

	// Verify both data sources are ready
	h.verifyDataSourcesReady()
	fmt.Println("✅ All game modes ready for play!")

	// Start automatic refresh timers
	h.startAutoRefresh()
	fmt.Printf("🔄 Auto-refresh scheduled:\n")
	fmt.Printf("  Bonhams (Hard): every 7 days (next: %s)\n",
		time.Now().Add(7*24*time.Hour).Format("Mon, 02 Jan 2006 15:04"))
	fmt.Printf("  Lookers (Easy): every 7 days (next: %s)\n",
		time.Now().Add(7*24*time.Hour).Format("Mon, 02 Jan 2006 15:04"))

	return h
}

// GetRandomListing godoc
// @Summary Get a random car listing for the game
// @Description Returns a random car listing with the price hidden (set to 0) for the guessing game. Rate limited to 60 requests per minute per IP.
// @Tags game
// @Produce json
// @Success 200 {object} models.EnhancedCar
// @Failure 404 {object} map[string]string "error: No listings available"
// @Failure 429 {object} map[string]string "error: Too Many Requests - Rate limited"
// @Router /api/random-listing [get]
func (h *Handler) GetRandomListing(c *gin.Context) {
	// Get session ID from header for history tracking
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		sessionID = generateSessionID()
	} else if err := validation.ValidateSessionID(sessionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID format"})
		return
	}

	h.mu.Lock() // Using Lock instead of RLock because we modify recentlyShown
	defer h.mu.Unlock()

	// Get all Bonhams listing IDs
	ids := make([]string, 0, len(h.bonhamsListings))
	for id := range h.bonhamsListings {
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No listings available"})
		return
	}

	// Select random listing avoiding recently shown cars
	randomID := h.selectRandomCarWithHistory(sessionID, ids)
	h.addToRecentlyShown(sessionID, randomID)

	bonhamsListing := h.bonhamsListings[randomID]

	// Convert to enhanced format and hide price
	enhancedListing := bonhamsListing.ToEnhancedCar()
	enhancedListing.Price = 0

	c.JSON(http.StatusOK, enhancedListing)
}

// GetRandomEnhancedListing godoc
// @Summary Get a random car listing with all characteristics based on difficulty
// @Description Returns a random car listing with full details, supports difficulty query param (easy/hard)
// @Tags game
// @Produce json
// @Param difficulty query string false "Difficulty mode (easy for Lookers, hard for Bonhams)" Enums(easy, hard)
// @Success 200 {object} models.EnhancedCar
// @Failure 404 {object} map[string]string "error: No listings available"
// @Router /api/random-enhanced-listing [get]
func (h *Handler) GetRandomEnhancedListing(c *gin.Context) {
	difficulty := c.DefaultQuery("difficulty", "hard") // Default to hard mode for backward compatibility

	// Get session ID from header (frontend should send this)
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		sessionID = generateSessionID() // Generate one if not provided
	} else if err := validation.ValidateSessionID(sessionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID format"})
		return
	}

	h.mu.Lock() // Using Lock instead of RLock because we modify recentlyShown
	defer h.mu.Unlock()

	if difficulty == "easy" {
		// Easy mode - use Lookers listings
		if len(h.lookersListings) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "No easy mode listings available"})
			return
		}

		// Get all Lookers listing IDs
		ids := make([]string, 0, len(h.lookersListings))
		for id := range h.lookersListings {
			ids = append(ids, id)
		}

		// Select random listing avoiding recently shown cars
		randomID := h.selectRandomCarWithHistory(sessionID, ids)
		h.addToRecentlyShown(sessionID, randomID)

		lookersListing := h.lookersListings[randomID]

		// Convert to enhanced format and hide price
		enhancedListing := lookersListing.ToEnhancedCar()
		enhancedListing.Price = 0

		c.JSON(http.StatusOK, enhancedListing)
	} else {
		// Hard mode - use Bonhams listings (default)
		if len(h.bonhamsListings) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "No hard mode listings available"})
			return
		}

		// Get all Bonhams listing IDs
		ids := make([]string, 0, len(h.bonhamsListings))
		for id := range h.bonhamsListings {
			ids = append(ids, id)
		}

		// Select random listing avoiding recently shown cars
		randomID := h.selectRandomCarWithHistory(sessionID, ids)
		h.addToRecentlyShown(sessionID, randomID)

		bonhamsListing := h.bonhamsListings[randomID]

		// Convert to enhanced format and hide price
		enhancedListing := bonhamsListing.ToEnhancedCar()
		enhancedListing.Price = 0

		c.JSON(http.StatusOK, enhancedListing)
	}
}

// CheckGuess godoc
// @Summary Submit a price guess for a car
// @Description Submit a price guess and get feedback on accuracy, score, and game status. Rate limited to 60 requests per minute per IP.
// @Tags game
// @Accept json
// @Produce json
// @Param guess body models.GuessRequest true "Price guess data (max price: £10,000,000)"
// @Success 200 {object} models.GuessResponse
// @Failure 400 {object} map[string]string "error: Invalid request or price exceeds maximum"
// @Failure 404 {object} map[string]string "error: Listing not found"
// @Failure 429 {object} map[string]string "error: Too Many Requests - Rate limited"
// @Router /api/check-guess [post]
func (h *Handler) CheckGuess(c *gin.Context) {
	// Read the raw body for debugging
	bodyBytes, _ := c.GetRawData()
	log.Printf("CheckGuess: Raw request body: %s", string(bodyBytes))

	// Restore the body so it can be read again
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var req models.GuessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("CheckGuess: Failed to parse JSON - %v", err)
		log.Printf("CheckGuess: Headers: %v", c.Request.Header)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	log.Printf("CheckGuess: Received request - ListingID: %s, GuessedPrice: %.2f, GameMode: %s",
		req.ListingID, req.GuessedPrice, req.GameMode)

	// Additional security validation - allow higher values via text input
	if req.GuessedPrice > 10000000 { // £10 million max
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid price", "message": "Price cannot exceed £10,000,000"})
		return
	}

	// Validate listing ID format (alphanumeric with hyphens)
	if !isValidListingID(req.ListingID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid listing ID format"})
		return
	}

	h.mu.RLock()

	// Try to find the listing in the appropriate difficulty mode
	var actualPrice float64
	var originalURL string
	var exists bool

	difficulty := req.Difficulty
	if difficulty == "" {
		difficulty = "hard" // Default to hard mode for backward compatibility
	}

	if difficulty == "easy" {
		// Check Lookers listings
		if lookersListing, found := h.lookersListings[req.ListingID]; found {
			actualPrice = lookersListing.Price
			originalURL = lookersListing.OriginalURL
			exists = true
		}
	} else {
		// Check Bonhams listings (hard mode)
		if bonhamsListing, found := h.bonhamsListings[req.ListingID]; found {
			actualPrice = bonhamsListing.Price
			originalURL = bonhamsListing.OriginalURL
			exists = true
		}
	}

	h.mu.RUnlock()

	if !exists {
		log.Printf("CheckGuess: Listing not found - ID: %s, Difficulty: %s", req.ListingID, difficulty)
		log.Printf("CheckGuess: Available listings (Hard): %d, (Easy): %d", len(h.bonhamsListings), len(h.lookersListings))
		c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found", "requestedId": req.ListingID, "difficulty": difficulty})
		return
	}

	// Calculate difference and percentage
	difference := math.Abs(actualPrice - req.GuessedPrice)
	percentage := (difference / actualPrice) * 100

	response := models.GuessResponse{
		ActualPrice:  actualPrice,
		GuessedPrice: req.GuessedPrice,
		Difference:   difference,
		Percentage:   percentage,
		OriginalURL:  originalURL,
	}

	// Handle game mode logic
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		sessionID = generateSessionID()
	} else if err := validation.ValidateSessionID(sessionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID format"})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	switch req.GameMode {
	case "zero":
		// Stay at Zero mode
		h.zeroScores[sessionID] += difference
		response.Score = int(h.zeroScores[sessionID])
		response.Correct = true // Always continue in this mode
		response.Message = "Keep your cumulative difference as low as possible!"

	case "streak":
		// Streak mode - must guess within 10%
		if percentage <= 10 {
			h.streakScores[sessionID]++
			response.Correct = true
			response.Score = h.streakScores[sessionID]
			response.Message = "Great guess! Keep the streak going!"
		} else {
			response.Correct = false
			response.GameOver = true
			response.Score = h.streakScores[sessionID]
			response.Message = "Game Over! Your guess was off by more than 10%"

			// Reset streak
			delete(h.streakScores, sessionID)
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetLeaderboard godoc
// @Summary Get the game leaderboard
// @Description Returns the leaderboard optionally filtered by game mode and difficulty, sorted by score (descending for challenge, ascending for streak)
// @Tags game
// @Produce json
// @Param mode query string false "Game mode filter (streak or challenge)"
// @Param difficulty query string false "Difficulty filter (easy or hard)"
// @Param limit query int false "Maximum number of entries to return (default: 10)"
// @Success 200 {array} models.LeaderboardEntry
// @Router /api/leaderboard [get]
func (h *Handler) GetLeaderboard(c *gin.Context) {
	gameMode := c.Query("mode")
	difficulty := c.Query("difficulty")
	limit := 10 // Default limit

	// Parse limit if provided
	if limitStr := c.Query("limit"); limitStr != "" {
		if l := parseInt(limitStr); l > 0 && l <= 100 {
			limit = l
		}
	}

	// Get leaderboard from database
	entries, err := h.db.GetLeaderboard(gameMode, difficulty, limit)
	if err != nil {
		log.Printf("Failed to get leaderboard from database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboard"})
		return
	}

	c.JSON(http.StatusOK, entries)
}

// SubmitScore godoc
// @Summary Submit a score to the leaderboard
// @Description Submit a score to the leaderboard for streak or challenge mode. Validates the score against the session data.
// @Tags game
// @Accept json
// @Produce json
// @Param submission body models.LeaderboardSubmissionRequest true "Score submission data"
// @Success 200 {object} map[string]interface{} "success message and leaderboard position"
// @Failure 400 {object} map[string]string "error: Invalid request or score validation failed"
// @Failure 429 {object} map[string]string "error: Too Many Requests - Rate limited"
// @Router /api/leaderboard/submit [post]
func (h *Handler) SubmitScore(c *gin.Context) {
	var req models.LeaderboardSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Validate name (additional server-side validation)
	if len(req.Name) == 0 || len(req.Name) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name must be between 1 and 20 characters"})
		return
	}

	// Sanitize name (remove any potentially harmful content)
	req.Name = sanitizeName(req.Name)

	h.mu.Lock()
	defer h.mu.Unlock()

	// Validate score for challenge mode by checking the session
	if req.GameMode == "challenge" && req.SessionID != "" {
		if session, exists := h.challengeSessions[req.SessionID]; exists {
			if !session.IsComplete {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Challenge session is not complete"})
				return
			}
			if session.TotalScore != req.Score {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Score does not match session data"})
				return
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
			return
		}
	}

	// For streak mode, we'll trust the submitted score for now
	// In a production environment, you'd want to validate this against session data too

	// Create leaderboard entry
	difficulty := req.Difficulty
	if difficulty == "" {
		difficulty = "hard" // Default to hard for backward compatibility
	}

	// Get user context if available
	var userID *int
	if user, exists := c.Get("user"); exists {
		if u, ok := user.(*models.User); ok {
			userID = &u.ID
		}
	}

	entry := models.LeaderboardEntry{
		UserID:     userID,
		Name:       req.Name,
		Score:      req.Score,
		GameMode:   req.GameMode,
		Difficulty: difficulty,
		SessionID:  req.SessionID,
	}

	// Save to database
	if err := h.db.AddLeaderboardEntry(&entry); err != nil {
		log.Printf("Failed to save leaderboard entry to database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save score"})
		return
	}

	// Update user's favorite difficulty and increment games played (for logged-in users)
	if userID != nil {
		// Increment total games played
		if err := h.db.IncrementUserGamesPlayed(*userID); err != nil {
			log.Printf("Failed to increment user's games played count: %v", err)
			// Don't fail the request for this non-critical operation
		}

		// Update favorite difficulty based on gameplay patterns
		if err := h.db.UpdateUserFavoriteDifficulty(*userID); err != nil {
			log.Printf("Failed to update user's favorite difficulty: %v", err)
			// Don't fail the request for this non-critical operation
		}
	}

	// Find position in leaderboard using database query
	position := h.findLeaderboardPositionFromDB(entry)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Score submitted successfully!",
		"position": position,
		"entry":    entry,
	})
}

// TestScraper godoc
// @Summary Test the car scraper directly (Admin Only)
// @Description Tests the Bonhams scraper and returns up to 10 cars with full details. This is an expensive operation. Requires admin authentication.
// @Tags admin
// @Security AdminKey
// @Produce json
// @Success 200 {object} map[string]interface{} "message, count, and cars array"
// @Failure 401 {object} map[string]string "error: Unauthorized - Admin key required"
// @Failure 429 {object} map[string]string "error: Too Many Requests - Rate limited"
// @Failure 500 {object} map[string]string "error and message"
// @Router /api/admin/test-scraper [get]
func (h *Handler) TestScraper(c *gin.Context) {
	cars, err := h.scraper.GetCarListings(10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"message": "Scraper failed",
		})
		return
	}

	if len(cars) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No cars found from scraper",
			"cars":    cars,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scraper working!",
		"count":   len(cars),
		"cars":    cars,
	})
}

// GetDataSource godoc
// @Summary Get current data source information
// @Description Returns information about both data sources (Bonhams and Lookers) including listing counts
// @Tags debug
// @Produce json
// @Success 200 {object} map[string]interface{} "data sources info with Easy/Hard mode listings"
// @Router /api/data-source [get]
func (h *Handler) GetDataSource(c *gin.Context) {
	h.mu.RLock()
	bonhamsListings := len(h.bonhamsListings)
	lookersListings := len(h.lookersListings)
	h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"hard_mode": gin.H{
			"data_source":    "bonhams_auctions",
			"total_listings": bonhamsListings,
			"description":    "Real Bonhams Car Auction results (Hard Mode)",
		},
		"easy_mode": gin.H{
			"data_source":    "lookers_dealership",
			"total_listings": lookersListings,
			"description":    "Lookers used car dealership listings (Easy Mode)",
		},
		"total_listings":  bonhamsListings + lookersListings,
		"modes_available": []string{"easy", "hard"},
	})
}

// GetAllListings godoc
// @Summary Get all available car listings (Admin Only)
// @Description Returns all car listings currently loaded in the system with full details including prices. This reveals all answers and is restricted to admin access.
// @Tags admin
// @Security AdminKey
// @Produce json
// @Success 200 {object} map[string]interface{} "count and cars array"
// @Failure 401 {object} map[string]string "error: Unauthorized - Admin key required"
// @Failure 429 {object} map[string]string "error: Too Many Requests - Rate limited"
// @Router /api/admin/listings [get]
func (h *Handler) GetAllListings(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	cars := make([]*models.BonhamsCar, 0, len(h.bonhamsListings))
	for _, car := range h.bonhamsListings {
		cars = append(cars, car)
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(cars),
		"cars":  cars,
	})
}

func (h *Handler) initializeBonhamsListings() {
	fmt.Println("🔄 Initializing Bonhams listings for Hard mode...")

	// Try to load from cache first
	if cachedListings, found := cache.LoadBonhamsFromCache(); found {
		h.loadBonhamsListingsFromCache(cachedListings)
		return
	}

	// Cache miss or expired, scrape fresh data (blocking to ensure Hard mode is ready)
	fmt.Println("📥 No Bonhams cache found - scraping fresh data before startup...")
	h.refreshBonhamsListings()
}

func (h *Handler) loadBonhamsListingsFromCache(cachedListings []*models.BonhamsCar) {
	h.mu.Lock()
	defer h.mu.Unlock()

	filteredCount := 0
	for _, bonhamsCar := range cachedListings {
		if bonhamsCar.Price != 700 {
			h.bonhamsListings[bonhamsCar.ID] = bonhamsCar
		} else {
			filteredCount++
			fmt.Printf("⚠️ Filtered cached car %s %s %d (£700 - no price found)\n", bonhamsCar.Make, bonhamsCar.Model, bonhamsCar.Year)
		}
	}

	if filteredCount > 0 {
		fmt.Printf("⚠️ Filtered %d cars with £700 price from cache\n", filteredCount)
	}
}

// loadLookersFromCache loads Lookers listings from cache
func (h *Handler) loadLookersFromCache(cachedListings []*models.LookersCar) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Load Lookers listings
	for _, lookersCar := range cachedListings {
		h.lookersListings[lookersCar.ID] = lookersCar
	}

	fmt.Printf("✅ Loaded %d Lookers cars from cache\n", len(cachedListings))
}

// verifyDataSourcesReady ensures both Hard and Easy modes have data available
func (h *Handler) verifyDataSourcesReady() {
	h.mu.RLock()
	bonhamsCount := len(h.bonhamsListings)
	lookersCount := len(h.lookersListings)
	h.mu.RUnlock()

	fmt.Printf("🔍 Data source verification:\n")
	fmt.Printf("   Hard Mode (Bonhams): %d cars loaded\n", bonhamsCount)
	fmt.Printf("   Easy Mode (Lookers): %d cars loaded\n", lookersCount)

	if bonhamsCount == 0 {
		fmt.Println("❌ WARNING: Hard Mode has no cars - users will see errors!")
	}
	if lookersCount == 0 {
		fmt.Println("❌ WARNING: Easy Mode has no cars - users will see errors!")
	}

	if bonhamsCount > 0 && lookersCount > 0 {
		fmt.Println("✅ Both game modes have sufficient data")
	}
}

// initializeLookersListings initializes Lookers listings for Easy mode
func (h *Handler) initializeLookersListings() {
	fmt.Println("🔄 Initializing Lookers listings for Easy mode...")

	// Try to load from cache first
	if cachedListings, found := cache.LoadLookersFromCache(); found {
		h.loadLookersFromCache(cachedListings)
		return
	}

	// Cache miss or expired, scrape fresh data (blocking to ensure Easy mode is ready)
	fmt.Println("📥 No Lookers cache found - scraping fresh data before startup...")
	h.refreshLookersListings()
}

func (h *Handler) refreshBonhamsListings() {
	fmt.Println("🔄 Refreshing listings from Bonhams Car Auctions...")

	// Check if USE_BONHAMS_CACHE_ONLY env var is set (temporary fix)
	useCacheOnly := os.Getenv("USE_BONHAMS_CACHE_ONLY")
	if useCacheOnly == "true" || useCacheOnly == "1" {
		fmt.Println("⚙️  USE_BONHAMS_CACHE_ONLY is set - loading from cache only")
		cachedListings, cacheErr := cache.LoadBonhamsFromCacheIgnoreExpiry()
		if cacheErr == nil && len(cachedListings) > 0 {
			h.mu.Lock()
			h.bonhamsListings = make(map[string]*models.BonhamsCar)
			for _, bonhamsCar := range cachedListings {
				if bonhamsCar.Price != 700 {
					h.bonhamsListings[bonhamsCar.ID] = bonhamsCar
				}
			}
			h.mu.Unlock()

			// Bump expiry
			if err := cache.BumpBonhamsExpiry(); err != nil {
				fmt.Printf("⚠️ Failed to bump Bonhams cache expiry: %v\n", err)
			}

			fmt.Printf("✅ Using %d cars from cache-only mode (extended expiry by 7 days)\n", len(h.bonhamsListings))
			return
		}
		fmt.Println("❌ USE_BONHAMS_CACHE_ONLY set but no cache available!")
	}

	// Get fresh Bonhams data (250 cars with parallel scraping)
	bonhamsCars, err := h.scraper.GetBonhamsListings(ListingAmount)
	if err == nil && len(bonhamsCars) > 0 {
		// Filter out listings with £700 price (indicates missing price data)
		var validCars []*models.BonhamsCar
		for _, car := range bonhamsCars {
			if car.Price != 700 {
				validCars = append(validCars, car)
			} else {
				fmt.Printf("⚠️ Filtered out %s %s %d (£700 - no price found)\n", car.Make, car.Model, car.Year)
			}
		}

		h.mu.Lock()
		// Clear existing listings
		h.bonhamsListings = make(map[string]*models.BonhamsCar)

		// Load valid listings only
		for _, bonhamsCar := range validCars {
			h.bonhamsListings[bonhamsCar.ID] = bonhamsCar
		}
		h.mu.Unlock()

		// Save filtered listings to cache
		if err := cache.SaveBonhamsToCache(validCars); err != nil {
			fmt.Printf("⚠️ Failed to save Bonhams cache: %v\n", err)
		}

		fmt.Printf("✅ Refreshed %d valid cars from Bonhams Car Auctions (%d filtered out)\n", len(validCars), len(bonhamsCars)-len(validCars))
		return
	}

	// Scraping failed - try to fall back to cached data (even if expired)
	fmt.Printf("❌ Bonhams scraper failed: %v\n", err)
	fmt.Println("🔄 Attempting to load from expired cache as fallback...")

	cachedListings, cacheErr := cache.LoadBonhamsFromCacheIgnoreExpiry()
	if cacheErr == nil && len(cachedListings) > 0 {
		// Successfully loaded from expired cache
		h.mu.Lock()
		h.bonhamsListings = make(map[string]*models.BonhamsCar)
		for _, bonhamsCar := range cachedListings {
			if bonhamsCar.Price != 700 {
				h.bonhamsListings[bonhamsCar.ID] = bonhamsCar
			}
		}
		h.mu.Unlock()

		// Bump expiry to give us another week before retry
		if err := cache.BumpBonhamsExpiry(); err != nil {
			fmt.Printf("⚠️ Failed to bump Bonhams cache expiry: %v\n", err)
		}

		fmt.Printf("✅ Using %d cars from fallback cache (extended expiry by 7 days)\n", len(h.bonhamsListings))
		return
	}

	// No cache available - use minimal mock data as last resort
	fmt.Println("❌ No cache available, creating minimal mock data for testing...")

	mockCars := []*models.BonhamsCar{
		{
			ID:            "mock1",
			Make:          "Volkswagen",
			Model:         "Golf GTI",
			Year:          2020,
			Price:         18995,
			Images:        []string{"https://images.unsplash.com/photo-1609521263047-f8f205293f24?w=600&h=400&fit=crop"},
			OriginalURL:   "https://example.com/mock1",
			Mileage:       "28,000 Miles",
			Engine:        "1.4 TSI",
			Gearbox:       "Manual",
			ExteriorColor: "Silver",
			InteriorColor: "Black",
			Steering:      "Right-hand drive",
			FuelType:      "Petrol",
			KeyFacts:      []string{"Low mileage", "Full service history", "Great condition"},
		},
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for _, mockCar := range mockCars {
		h.bonhamsListings[mockCar.ID] = mockCar
	}
}

// startAutoRefresh starts background goroutines that refresh listings on different schedules
func (h *Handler) startAutoRefresh() {
	// Create ticker for Bonhams (Hard mode) - 7-day intervals
	h.bonhamsRefreshTicker = time.NewTicker(7 * 24 * time.Hour)

	// Create ticker for Lookers (Easy mode) - 7-day intervals
	h.lookersRefreshTicker = time.NewTicker(7 * 24 * time.Hour)

	// Bonhams auto-refresh goroutine
	go func() {
		for range h.bonhamsRefreshTicker.C {
			fmt.Println("⏰ Bonhams auto-refresh triggered (7 days elapsed)")
			// Check for active games before refreshing
			if h.hasActiveGames() {
				fmt.Println("⚠️ Active games detected, postponing Bonhams refresh for 1 hour")
				time.AfterFunc(1*time.Hour, func() {
					go h.refreshBonhamsListingsAsync()
				})
			} else {
				// Run refresh in background to avoid blocking gameplay
				go h.refreshBonhamsListingsAsync()
			}
		}
	}()

	// Lookers auto-refresh goroutine
	go func() {
		for range h.lookersRefreshTicker.C {
			fmt.Println("⏰ Lookers auto-refresh triggered (7 days elapsed)")
			// Check for active games before refreshing
			if h.hasActiveGames() {
				fmt.Println("⚠️ Active games detected, postponing Lookers refresh for 1 hour")
				time.AfterFunc(1*time.Hour, func() {
					go h.refreshLookersListingsAsync()
				})
			} else {
				// Run refresh in background to avoid blocking gameplay
				go h.refreshLookersListingsAsync()
			}
		}
	}()
}

// hasActiveGames checks if there are any active game sessions or challenge sessions
func (h *Handler) hasActiveGames() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Check for active challenge sessions (incomplete sessions)
	for _, session := range h.challengeSessions {
		if !session.IsComplete {
			return true
		}
	}

	// Check for active streak/zero mode sessions (non-zero scores indicate active games)
	if len(h.streakScores) > 0 || len(h.zeroScores) > 0 {
		return true
	}

	return false
}

// refreshListingsAsync performs a non-blocking refresh that doesn't interrupt gameplay
func (h *Handler) refreshBonhamsListingsAsync() {
	fmt.Println("🔄 Starting background refresh (non-blocking)...")

	// Check if USE_BONHAMS_CACHE_ONLY env var is set (temporary fix)
	useCacheOnly := os.Getenv("USE_BONHAMS_CACHE_ONLY")
	if useCacheOnly == "true" || useCacheOnly == "1" {
		fmt.Println("⚙️  USE_BONHAMS_CACHE_ONLY is set - loading from cache only")
		cachedListings, cacheErr := cache.LoadBonhamsFromCacheIgnoreExpiry()
		if cacheErr == nil && len(cachedListings) > 0 {
			h.mu.Lock()
			oldCount := len(h.bonhamsListings)
			h.bonhamsListings = make(map[string]*models.BonhamsCar)
			for _, bonhamsCar := range cachedListings {
				if bonhamsCar.Price != 700 {
					h.bonhamsListings[bonhamsCar.ID] = bonhamsCar
				}
			}
			h.mu.Unlock()

			// Bump expiry
			if err := cache.BumpBonhamsExpiry(); err != nil {
				fmt.Printf("⚠️ Failed to bump Bonhams cache expiry: %v\n", err)
			}

			fmt.Printf("✅ Background refresh used cache-only mode: %d cars (was %d, extended expiry by 7 days)\n",
				len(h.bonhamsListings), oldCount)
			return
		}
		fmt.Println("❌ USE_BONHAMS_CACHE_ONLY set but no cache available - keeping existing data")
		return
	}

	// Get fresh Bonhams data (this may take a few minutes) - 250 cars with parallel scraping
	bonhamsCars, err := h.scraper.GetBonhamsListings(ListingAmount)
	if err != nil || len(bonhamsCars) == 0 {
		fmt.Printf("❌ Background refresh failed: %v\n", err)

		// Try to fall back to cached data (even if expired)
		fmt.Println("🔄 Attempting to load from expired cache as fallback...")
		cachedListings, cacheErr := cache.LoadBonhamsFromCacheIgnoreExpiry()
		if cacheErr == nil && len(cachedListings) > 0 {
			h.mu.Lock()
			oldCount := len(h.bonhamsListings)
			h.bonhamsListings = make(map[string]*models.BonhamsCar)
			for _, bonhamsCar := range cachedListings {
				if bonhamsCar.Price != 700 {
					h.bonhamsListings[bonhamsCar.ID] = bonhamsCar
				}
			}
			h.mu.Unlock()

			// Bump expiry to give us another week before retry
			if err := cache.BumpBonhamsExpiry(); err != nil {
				fmt.Printf("⚠️ Failed to bump Bonhams cache expiry: %v\n", err)
			}

			fmt.Printf("✅ Background refresh used fallback cache: %d cars (was %d, extended expiry by 7 days)\n",
				len(h.bonhamsListings), oldCount)
			return
		}

		fmt.Println("❌ No cache available for fallback - keeping existing data")
		return
	}

	// Filter out listings with £700 price (indicates missing price data)
	var validCars []*models.BonhamsCar
	for _, car := range bonhamsCars {
		if car.Price != 700 {
			validCars = append(validCars, car)
		}
	}

	// Quick atomic update - only lock briefly
	h.mu.Lock()
	oldCount := len(h.bonhamsListings)
	h.bonhamsListings = make(map[string]*models.BonhamsCar)
	for _, bonhamsCar := range validCars {
		h.bonhamsListings[bonhamsCar.ID] = bonhamsCar
	}
	h.mu.Unlock()

	// Save to cache (this can take time, but doesn't block gameplay)
	if err := cache.SaveBonhamsToCache(validCars); err != nil {
		fmt.Printf("⚠️ Failed to save Bonhams cache during background refresh: %v\n", err)
	}

	fmt.Printf("✅ Background refresh complete: %d cars updated (was %d, filtered %d)\n",
		len(validCars), oldCount, len(bonhamsCars)-len(validCars))
}

// refreshLookersListings refreshes Lookers listings for Easy mode
func (h *Handler) refreshLookersListings() {
	fmt.Println("🔄 Refreshing Lookers listings for Easy mode...")

	// Check if USE_LOOKERS_CACHE_ONLY env var is set (temporary fix)
	useCacheOnly := os.Getenv("USE_LOOKERS_CACHE_ONLY")
	if useCacheOnly == "true" || useCacheOnly == "1" {
		fmt.Println("⚙️  USE_LOOKERS_CACHE_ONLY is set - loading from cache only")
		cachedListings, cacheErr := cache.LoadLookersFromCacheIgnoreExpiry()
		if cacheErr == nil && len(cachedListings) > 0 {
			h.mu.Lock()
			h.lookersListings = make(map[string]*models.LookersCar)
			for _, lookersCar := range cachedListings {
				h.lookersListings[lookersCar.ID] = lookersCar
			}
			h.mu.Unlock()

			// Bump expiry
			if err := cache.BumpLookersExpiry(); err != nil {
				fmt.Printf("⚠️ Failed to bump Lookers cache expiry: %v\n", err)
			}

			fmt.Printf("✅ Using %d cars from cache-only mode (extended expiry by 7 days)\n", len(cachedListings))
			return
		}
		fmt.Println("❌ USE_LOOKERS_CACHE_ONLY set but no cache available!")
	}

	// Get fresh Lookers data (scraper is now stateless)
	lookersCars, err := h.scraper.GetLookersListings()
	if err != nil || len(lookersCars) == 0 {
		fmt.Printf("❌ Lookers scraper failed: %v\n", err)

		// Try to fall back to cached data (even if expired)
		fmt.Println("🔄 Attempting to load from expired cache as fallback...")
		cachedListings, cacheErr := cache.LoadLookersFromCacheIgnoreExpiry()
		if cacheErr == nil && len(cachedListings) > 0 {
			h.mu.Lock()
			h.lookersListings = make(map[string]*models.LookersCar)
			for _, lookersCar := range cachedListings {
				h.lookersListings[lookersCar.ID] = lookersCar
			}
			h.mu.Unlock()

			// Bump expiry to give us another week before retry
			if err := cache.BumpLookersExpiry(); err != nil {
				fmt.Printf("⚠️ Failed to bump Lookers cache expiry: %v\n", err)
			}

			fmt.Printf("✅ Using %d cars from fallback cache (extended expiry by 7 days)\n", len(cachedListings))
			return
		}

		fmt.Println("❌ No cache available for fallback")
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Clear existing Lookers listings
	h.lookersListings = make(map[string]*models.LookersCar)

	// Load Lookers listings
	for _, lookersCar := range lookersCars {
		h.lookersListings[lookersCar.ID] = lookersCar
	}

	// Save to cache
	if err := cache.SaveLookersToCache(lookersCars); err != nil {
		fmt.Printf("⚠️ Failed to save Lookers cache: %v\n", err)
	}

	fmt.Printf("✅ Refreshed %d cars from Lookers for Easy mode\n", len(lookersCars))
}

// refreshLookersListingsAsync performs a non-blocking Lookers refresh that doesn't interrupt gameplay
func (h *Handler) refreshLookersListingsAsync() {
	fmt.Println("🔄 Starting Lookers background refresh (non-blocking)...")

	// Check if USE_LOOKERS_CACHE_ONLY env var is set (temporary fix)
	useCacheOnly := os.Getenv("USE_LOOKERS_CACHE_ONLY")
	if useCacheOnly == "true" || useCacheOnly == "1" {
		fmt.Println("⚙️  USE_LOOKERS_CACHE_ONLY is set - loading from cache only")
		cachedListings, cacheErr := cache.LoadLookersFromCacheIgnoreExpiry()
		if cacheErr == nil && len(cachedListings) > 0 {
			h.mu.Lock()
			oldCount := len(h.lookersListings)
			h.lookersListings = make(map[string]*models.LookersCar)
			for _, lookersCar := range cachedListings {
				h.lookersListings[lookersCar.ID] = lookersCar
			}
			h.mu.Unlock()

			// Bump expiry
			if err := cache.BumpLookersExpiry(); err != nil {
				fmt.Printf("⚠️ Failed to bump Lookers cache expiry: %v\n", err)
			}

			fmt.Printf("✅ Lookers background refresh used cache-only mode: %d cars (was %d, extended expiry by 7 days)\n",
				len(cachedListings), oldCount)
			return
		}
		fmt.Println("❌ USE_LOOKERS_CACHE_ONLY set but no cache available - keeping existing data")
		return
	}

	// Get fresh Lookers data (scraper is now stateless)
	lookersCars, err := h.scraper.GetLookersListings()
	if err != nil || len(lookersCars) == 0 {
		fmt.Printf("❌ Lookers background refresh failed: %v\n", err)

		// Try to fall back to cached data (even if expired)
		fmt.Println("🔄 Attempting to load from expired cache as fallback...")
		cachedListings, cacheErr := cache.LoadLookersFromCacheIgnoreExpiry()
		if cacheErr == nil && len(cachedListings) > 0 {
			h.mu.Lock()
			oldCount := len(h.lookersListings)
			h.lookersListings = make(map[string]*models.LookersCar)
			for _, lookersCar := range cachedListings {
				h.lookersListings[lookersCar.ID] = lookersCar
			}
			h.mu.Unlock()

			// Bump expiry to give us another week before retry
			if err := cache.BumpLookersExpiry(); err != nil {
				fmt.Printf("⚠️ Failed to bump Lookers cache expiry: %v\n", err)
			}

			fmt.Printf("✅ Lookers background refresh used fallback cache: %d cars (was %d, extended expiry by 7 days)\n",
				len(cachedListings), oldCount)
			return
		}

		fmt.Println("❌ No cache available for fallback - keeping existing data")
		return
	}

	// Quick atomic update - only lock briefly
	h.mu.Lock()
	oldCount := len(h.lookersListings)
	h.lookersListings = make(map[string]*models.LookersCar)
	for _, lookersCar := range lookersCars {
		h.lookersListings[lookersCar.ID] = lookersCar
	}
	h.mu.Unlock()

	// Save to cache (this can take time, but doesn't block gameplay)
	if err := cache.SaveLookersToCache(lookersCars); err != nil {
		fmt.Printf("⚠️ Failed to save Lookers cache during background refresh: %v\n", err)
	}

	fmt.Printf("✅ Lookers background refresh complete: %d cars updated (was %d)\n",
		len(lookersCars), oldCount)
}

// StopAutoRefresh stops the automatic refresh tickers (useful for cleanup)
func (h *Handler) StopAutoRefresh() {
	if h.bonhamsRefreshTicker != nil {
		h.bonhamsRefreshTicker.Stop()
		fmt.Println("🛑 Bonhams auto-refresh stopped")
	}
	if h.lookersRefreshTicker != nil {
		h.lookersRefreshTicker.Stop()
		fmt.Println("🛑 Lookers auto-refresh stopped")
	}
}

// ManualRefresh godoc
// @Summary Manually refresh car listings (Admin Only)
// @Description Triggers a non-blocking background refresh of car listings. Supports mode query param (bonhams/lookers/both). Requires admin authentication and has a 30-minute cooldown between requests. Game continues normally during refresh.
// @Tags admin
// @Security AdminKey
// @Produce json
// @Param mode query string false "Refresh mode (bonhams, lookers, both)" Enums(bonhams, lookers, both) default(both)
// @Success 200 {object} map[string]interface{} "message: refresh started, status: refreshing, note: game continues normally"
// @Failure 401 {object} map[string]string "error: Unauthorized - Admin key required"
// @Failure 429 {object} map[string]string "error: Too Many Requests - Rate limited or refresh cooldown active"
// @Router /api/admin/refresh-listings [post]
func (h *Handler) ManualRefresh(c *gin.Context) {
	mode := c.DefaultQuery("mode", "both") // Default to refreshing both

	fmt.Printf("🔄 Manual refresh requested for mode: %s\n", mode)

	switch mode {
	case "bonhams":
		go func() {
			h.refreshBonhamsListingsAsync()
		}()
		c.JSON(http.StatusOK, gin.H{
			"message": "Bonhams refresh started in background (non-blocking)",
			"status":  "refreshing",
			"mode":    "bonhams",
			"note":    "Game will continue normally while refresh happens in background",
		})
	case "lookers":
		go func() {
			h.refreshLookersListingsAsync()
		}()
		c.JSON(http.StatusOK, gin.H{
			"message": "Lookers refresh started in background (non-blocking)",
			"status":  "refreshing",
			"mode":    "lookers",
			"note":    "Game will continue normally while refresh happens in background",
		})
	case "both":
		go func() {
			h.refreshBonhamsListingsAsync()
		}()
		go func() {
			h.refreshLookersListingsAsync()
		}()
		c.JSON(http.StatusOK, gin.H{
			"message": "Both Bonhams and Lookers refresh started in background (non-blocking)",
			"status":  "refreshing",
			"mode":    "both",
			"note":    "Game will continue normally while refresh happens in background",
		})
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid mode. Use: bonhams, lookers, or both",
		})
	}
}

// GetCacheStatus godoc
// @Summary Get cache status information (Admin Only)
// @Description Returns information about the current cache status, age, and listing counts. Requires admin authentication.
// @Tags admin
// @Security AdminKey
// @Produce json
// @Success 200 {object} map[string]interface{} "cache status information"
// @Failure 401 {object} map[string]string "error: Unauthorized - Admin key required"
// @Failure 429 {object} map[string]string "error: Too Many Requests - Rate limited"
// @Router /api/admin/cache-status [get]
func (h *Handler) GetCacheStatus(c *gin.Context) {
	bonhamsAge, bonhamsErr := cache.GetBonhamsCacheAge()
	lookersAge, lookersErr := cache.GetLookersCacheAge()
	bonhamsExpired := cache.IsBonhamsCacheExpired()
	lookersExpired := cache.IsLookersCacheExpired()

	h.mu.RLock()
	bonhamsListings := len(h.bonhamsListings)
	lookersListings := len(h.lookersListings)
	h.mu.RUnlock()

	status := gin.H{
		"total_listings": bonhamsListings + lookersListings,
		"bonhams": gin.H{
			"listings":        bonhamsListings,
			"cache_expired":   bonhamsExpired,
			"next_refresh_in": "up to 7 days",
		},
		"lookers": gin.H{
			"listings":        lookersListings,
			"cache_expired":   lookersExpired,
			"next_refresh_in": "up to 7 days",
		},
	}

	// Bonhams cache age
	if bonhamsErr == nil {
		status["bonhams"].(gin.H)["cache_age"] = bonhamsAge.Round(time.Hour).String()
		status["bonhams"].(gin.H)["cache_age_hours"] = bonhamsAge.Hours()
	} else {
		status["bonhams"].(gin.H)["cache_age"] = "no cache file"
	}

	// Lookers cache age
	if lookersErr == nil {
		status["lookers"].(gin.H)["cache_age"] = lookersAge.Round(time.Hour).String()
		status["lookers"].(gin.H)["cache_age_hours"] = lookersAge.Hours()
	} else {
		status["lookers"].(gin.H)["cache_age"] = "no cache file"
	}

	c.JSON(http.StatusOK, status)
}

// StartChallenge godoc
// @Summary Start a new Challenge Mode session
// @Description Starts a new 10-car challenge session with GeoGuessr-style scoring. Supports difficulty query param (easy/hard). Rate limited to 60 requests per minute per IP.
// @Tags challenge
// @Produce json
// @Param difficulty query string false "Difficulty mode (easy for Lookers, hard for Bonhams)" Enums(easy, hard)
// @Success 200 {object} models.ChallengeSession "sessionId, cars array (10 cars with prices hidden), currentCar: 0, totalScore: 0"
// @Failure 404 {object} map[string]string "error: Not enough cars available for challenge mode"
// @Failure 429 {object} map[string]string "error: Too Many Requests - Rate limited"
// @Router /api/challenge/start [post]
func (h *Handler) StartChallenge(c *gin.Context) {
	difficulty := c.DefaultQuery("difficulty", "hard") // Default to hard mode for backward compatibility

	h.mu.RLock()

	var selectedCars []*models.EnhancedCar

	if difficulty == "easy" {
		// Easy mode - use Lookers listings
		if len(h.lookersListings) < 10 {
			h.mu.RUnlock()
			c.JSON(http.StatusNotFound, gin.H{"error": "Not enough easy mode cars available for challenge mode"})
			return
		}

		// Select 10 random Lookers cars
		var allCars []*models.LookersCar
		for _, car := range h.lookersListings {
			allCars = append(allCars, car)
		}
		h.mu.RUnlock() // Release read lock before processing

		// Shuffle and select 10
		rand.Shuffle(len(allCars), func(i, j int) {
			allCars[i], allCars[j] = allCars[j], allCars[i]
		})

		selectedCars = make([]*models.EnhancedCar, 10)
		for i := 0; i < 10; i++ {
			enhancedCar := allCars[i].ToEnhancedCar()
			enhancedCar.Price = 0 // Hide price for guessing
			selectedCars[i] = enhancedCar
		}
	} else {
		// Hard mode - use Bonhams listings (default)
		if len(h.bonhamsListings) < 10 {
			h.mu.RUnlock()
			c.JSON(http.StatusNotFound, gin.H{"error": "Not enough hard mode cars available for challenge mode"})
			return
		}

		// Select 10 random Bonhams cars
		var allCars []*models.BonhamsCar
		for _, car := range h.bonhamsListings {
			allCars = append(allCars, car)
		}
		h.mu.RUnlock() // Release read lock before processing

		// Shuffle and select 10
		rand.Shuffle(len(allCars), func(i, j int) {
			allCars[i], allCars[j] = allCars[j], allCars[i]
		})

		selectedCars = make([]*models.EnhancedCar, 10)
		for i := 0; i < 10; i++ {
			enhancedCar := allCars[i].ToEnhancedCar()
			enhancedCar.Price = 0 // Hide price for guessing
			selectedCars[i] = enhancedCar
		}
	}

	// Create challenge session
	sessionID := generateSessionID()
	session := &models.ChallengeSession{
		SessionID:  sessionID,
		Difficulty: difficulty, // Add the missing difficulty field
		Cars:       selectedCars,
		CurrentCar: 0,
		Guesses:    make([]models.ChallengeGuess, 0),
		TotalScore: 0,
		IsComplete: false,
		StartTime:  time.Now().Format(time.RFC3339),
	}

	// Get user context if available
	var userID int
	if user, exists := c.Get("user"); exists {
		if u, ok := user.(*models.User); ok {
			userID = u.ID
			session.UserID = userID
		}
	}

	// Save to database
	if err := h.db.CreateChallengeSession(session); err != nil {
		log.Printf("Failed to save challenge session to database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create challenge session"})
		return
	}

	// Also store in memory for backward compatibility during transition
	h.mu.Lock()
	h.challengeSessions[sessionID] = session
	h.mu.Unlock()

	c.JSON(http.StatusOK, session)
}

// GetChallengeSession godoc
// @Summary Get current challenge session
// @Description Returns the current state of a challenge session
// @Tags challenge
// @Produce json
// @Param sessionId path string true "Session ID"
// @Success 200 {object} models.ChallengeSession
// @Failure 404 {object} map[string]string "error: Session not found"
// @Router /api/challenge/{sessionId} [get]
func (h *Handler) GetChallengeSession(c *gin.Context) {
	sessionID := c.Param("sessionId")

	// Validate session ID
	if !isValidSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID format"})
		return
	}

	// Try to get session from database first
	session, err := h.db.GetChallengeSession(sessionID)
	if err != nil {
		log.Printf("Failed to get challenge session from database: %v", err)
		// Fallback to in-memory storage
		h.mu.RLock()
		sessionMem, exists := h.challengeSessions[sessionID]
		h.mu.RUnlock()

		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Challenge session not found"})
			return
		}
		session = sessionMem
	}

	if session == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Challenge session not found"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// SubmitChallengeGuess godoc
// @Summary Submit a guess for challenge mode
// @Description Submit a price guess for the current car in challenge mode. Returns points based on accuracy (max 5000 points). Rate limited to 60 requests per minute per IP.
// @Tags challenge
// @Accept json
// @Produce json
// @Param sessionId path string true "Challenge Session ID (16 alphanumeric characters)"
// @Param guess body models.ChallengeGuessRequest true "Price guess (max price: £10,000,000)"
// @Success 200 {object} models.ChallengeResponse "points earned, totalScore, isLastCar, message, originalUrl"
// @Failure 400 {object} map[string]string "error: Invalid request, session complete, or price exceeds maximum"
// @Failure 404 {object} map[string]string "error: Session not found"
// @Failure 429 {object} map[string]string "error: Too Many Requests - Rate limited"
// @Router /api/challenge/{sessionId}/guess [post]
func (h *Handler) SubmitChallengeGuess(c *gin.Context) {
	sessionID := c.Param("sessionId")

	// Validate session ID
	if !isValidSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID format"})
		return
	}

	var req models.ChallengeGuessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Additional security validation - allow higher values via text input
	if req.GuessedPrice > 10000000 { // £10 million max
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid price", "message": "Price cannot exceed £10,000,000"})
		return
	}

	// Get session from database first
	session, err := h.db.GetChallengeSession(sessionID)
	if err != nil {
		log.Printf("Failed to get challenge session from database: %v", err)
		// Fallback to in-memory storage
		h.mu.Lock()
		sessionMem, exists := h.challengeSessions[sessionID]
		if !exists {
			h.mu.Unlock()
			c.JSON(http.StatusNotFound, gin.H{"error": "Challenge session not found"})
			return
		}
		session = sessionMem
		h.mu.Unlock()
	}

	if session == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Challenge session not found"})
		return
	}

	h.mu.Lock() // Lock for updates

	if session.IsComplete {
		h.mu.Unlock()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Challenge session is already complete"})
		return
	}

	if session.CurrentCar >= len(session.Cars) {
		h.mu.Unlock()
		c.JSON(http.StatusBadRequest, gin.H{"error": "No more cars in this challenge"})
		return
	}

	// Get the current car and its original price
	currentCar := session.Cars[session.CurrentCar]
	var actualPrice float64
	var originalURL string

	// Find the original car with the actual price (check both difficulty modes)
	found := false

	// Try Bonhams listings first (hard mode)
	for _, bonhamsCar := range h.bonhamsListings {
		if bonhamsCar.ID == currentCar.ID {
			actualPrice = bonhamsCar.Price
			originalURL = bonhamsCar.OriginalURL
			found = true
			break
		}
	}

	// If not found in Bonhams, try Lookers listings (easy mode)
	if !found {
		for _, lookersCar := range h.lookersListings {
			if lookersCar.ID == currentCar.ID {
				actualPrice = lookersCar.Price
				originalURL = lookersCar.OriginalURL
				found = true
				break
			}
		}
	}

	if !found || actualPrice == 0 {
		h.mu.Unlock()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not find actual price for this car"})
		return
	}

	// Calculate difference and percentage
	difference := math.Abs(actualPrice - req.GuessedPrice)
	percentage := (difference / actualPrice) * 100

	// Calculate points using Geoguessr-like scoring
	points := h.calculateChallengePoints(percentage)

	// Create guess record
	guess := models.ChallengeGuess{
		CarID:        currentCar.ID,
		GuessedPrice: req.GuessedPrice,
		ActualPrice:  actualPrice,
		Difference:   difference,
		Percentage:   percentage,
		Points:       points,
	}

	// Add to session
	session.Guesses = append(session.Guesses, guess)
	session.TotalScore += points
	session.CurrentCar++

	// Check if challenge is complete
	isLastCar := session.CurrentCar >= len(session.Cars)
	if isLastCar {
		session.IsComplete = true
		session.CompletedTime = time.Now().Format(time.RFC3339)
	}

	// Save guess to database
	if err := h.db.AddChallengeGuess(sessionID, &guess); err != nil {
		log.Printf("Failed to save challenge guess to database: %v", err)
	}

	// Update session in database
	if err := h.db.UpdateChallengeSession(session); err != nil {
		log.Printf("Failed to update challenge session in database: %v", err)
	}

	// Also update in-memory storage for backward compatibility
	h.challengeSessions[sessionID] = session

	h.mu.Unlock()

	// Create response
	response := models.ChallengeResponse{
		ChallengeGuess:  guess,
		IsLastCar:       isLastCar,
		TotalScore:      session.TotalScore,
		SessionComplete: session.IsComplete,
		OriginalURL:     originalURL,
	}

	if !isLastCar {
		response.NextCarNumber = session.CurrentCar + 1
		response.Message = fmt.Sprintf("Car %d/10 - %d points! Moving to next car...", session.CurrentCar, points)
	} else {
		response.Message = fmt.Sprintf("Challenge Complete! Final Score: %d points", session.TotalScore)
	}

	c.JSON(http.StatusOK, response)
}

// calculateChallengePoints calculates points based on guess accuracy (Geoguessr-style)
func (h *Handler) calculateChallengePoints(percentage float64) int {
	// Points scale: 5000 max points for perfect guess, decreasing with error percentage
	// Perfect guess (0% error): 5000 points
	// 1% error: ~4950 points
	// 5% error: ~4750 points
	// 10% error: ~4500 points
	// 25% error: ~3750 points
	// 50% error: ~2500 points
	// 100% error or more: 0 points

	if percentage >= 100 {
		return 0
	}

	// Use exponential decay for scoring
	// Points = 5000 * e^(-percentage/20)
	points := 5000 * math.Exp(-percentage/20)

	// Round to nearest integer
	return int(math.Round(points))
}

// CreateTemplateChallenge creates a challenge session template for friend challenges
func (h *Handler) CreateTemplateChallenge(difficulty string, userID int) (*models.ChallengeSession, error) {
	h.mu.RLock()
	var selectedCars []*models.EnhancedCar

	if difficulty == "easy" {
		// Easy mode - use Lookers listings
		if len(h.lookersListings) < 10 {
			h.mu.RUnlock()
			return nil, fmt.Errorf("not enough easy mode cars available")
		}

		var allCars []*models.LookersCar
		for _, car := range h.lookersListings {
			allCars = append(allCars, car)
		}
		h.mu.RUnlock()

		// Shuffle and select 10
		rand.Shuffle(len(allCars), func(i, j int) {
			allCars[i], allCars[j] = allCars[j], allCars[i]
		})

		selectedCars = make([]*models.EnhancedCar, 10)
		for i := 0; i < 10; i++ {
			enhancedCar := allCars[i].ToEnhancedCar()
			enhancedCar.Price = 0 // Hide price for guessing
			selectedCars[i] = enhancedCar
		}
	} else {
		// Hard mode - use Bonhams listings
		if len(h.bonhamsListings) < 10 {
			h.mu.RUnlock()
			return nil, fmt.Errorf("not enough hard mode cars available")
		}

		var allCars []*models.BonhamsCar
		for _, car := range h.bonhamsListings {
			allCars = append(allCars, car)
		}
		h.mu.RUnlock()

		// Shuffle and select 10
		rand.Shuffle(len(allCars), func(i, j int) {
			allCars[i], allCars[j] = allCars[j], allCars[i]
		})

		selectedCars = make([]*models.EnhancedCar, 10)
		for i := 0; i < 10; i++ {
			enhancedCar := allCars[i].ToEnhancedCar()
			enhancedCar.Price = 0 // Hide price for guessing
			selectedCars[i] = enhancedCar
		}
	}

	sessionID := generateSessionID()

	session := &models.ChallengeSession{
		SessionID:  sessionID,
		UserID:     userID,
		Difficulty: difficulty,
		Cars:       selectedCars,
		CurrentCar: 0,
		Guesses:    []models.ChallengeGuess{},
		TotalScore: 0,
		IsComplete: false,
		StartTime:  time.Now().Format(time.RFC3339),
	}

	// Save the template session to database
	if err := h.db.CreateChallengeSession(session); err != nil {
		return nil, fmt.Errorf("failed to create template session: %w", err)
	}

	return session, nil
}

// generateSessionID returns a short, URL-safe identifier used to track anonymous sessions.
func generateSessionID() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// addToRecentlyShown adds a car ID to the recently shown list for a session
func (h *Handler) addToRecentlyShown(sessionID, carID string) {
	const maxRecentCars = 10 // Keep track of last 10 cars

	if h.recentlyShown[sessionID] == nil {
		h.recentlyShown[sessionID] = make([]string, 0, maxRecentCars)
	}

	// Add to front of list
	h.recentlyShown[sessionID] = append([]string{carID}, h.recentlyShown[sessionID]...)

	// Keep only last maxRecentCars
	if len(h.recentlyShown[sessionID]) > maxRecentCars {
		h.recentlyShown[sessionID] = h.recentlyShown[sessionID][:maxRecentCars]
	}
}

// isRecentlyShown checks if a car was recently shown to this session
func (h *Handler) isRecentlyShown(sessionID, carID string) bool {
	recentCars := h.recentlyShown[sessionID]
	for _, recentID := range recentCars {
		if recentID == carID {
			return true
		}
	}
	return false
}

// selectRandomCarWithHistory selects a random car that hasn't been recently shown
func (h *Handler) selectRandomCarWithHistory(sessionID string, allIDs []string) string {
	const maxAttempts = 20 // Prevent infinite loops

	for attempt := 0; attempt < maxAttempts; attempt++ {
		randomID := allIDs[rand.Intn(len(allIDs))]
		if !h.isRecentlyShown(sessionID, randomID) {
			return randomID
		}
	}

	// If we can't find a non-recent car after maxAttempts, just return any random car
	// This ensures the game doesn't break if user plays longer than available cars
	return allIDs[rand.Intn(len(allIDs))]
}

// isValidListingID validates listing ID format
func isValidListingID(id string) bool {
	// Allow alphanumeric characters, hyphens, and underscores
	// Max length 100 characters
	if len(id) == 0 || len(id) > 100 {
		return false
	}

	validID := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	return validID.MatchString(id)
}

// isValidSessionID validates session ID format
func isValidSessionID(id string) bool {
	// Session IDs should be 16 characters, alphanumeric only
	if len(id) != 16 {
		return false
	}

	validID := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	return validID.MatchString(id)
}

// sanitizeName removes potentially harmful content from names
func sanitizeName(name string) string {
	// Remove any HTML/JS content and limit to basic characters
	validName := regexp.MustCompile(`[^a-zA-Z0-9\s\-_.]`)
	return validName.ReplaceAllString(strings.TrimSpace(name), "")
}

// parseInt safely parses an integer string
func parseInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}

// trimLeaderboard keeps only top entries per game mode and difficulty combination

// findLeaderboardPositionFromDB calculates the 1-based rank for the submitted entry.
func (h *Handler) findLeaderboardPositionFromDB(entry models.LeaderboardEntry) int {
	// Get all entries for the same game mode and difficulty, ordered by score
	allEntries, err := h.db.GetLeaderboard(entry.GameMode, entry.Difficulty, 0) // 0 = no limit
	if err != nil {
		log.Printf("Failed to get leaderboard for position calculation: %v", err)
		return -1
	}

	// Find position of the entry (entries are already sorted by score in descending order)
	for i, e := range allEntries {
		if e.Score <= entry.Score {
			return i + 1
		}
	}

	// If entry score is lower than all existing entries
	return len(allEntries) + 1
}

// GetLeaderboardStatus godoc
// @Summary Get leaderboard status information (Admin Only)
// @Description Returns information about the leaderboard file, entry counts, and storage details. Requires admin authentication.
// @Tags admin
// @Security AdminKey
// @Produce json
// @Success 200 {object} map[string]interface{} "leaderboard status information"
// @Failure 401 {object} map[string]string "error: Unauthorized - Admin key required"
// @Failure 429 {object} map[string]string "error: Too Many Requests - Rate limited"
// @Router /api/admin/leaderboard-status [get]
func (h *Handler) GetLeaderboardStatus(c *gin.Context) {
	// Get counts from database for each mode and difficulty combination
	counts := make(map[string]int)
	totalEntries := 0

	modes := []string{"challenge", "streak", "zero"}
	difficulties := []string{"easy", "hard"}

	for _, mode := range modes {
		for _, difficulty := range difficulties {
			entries, err := h.db.GetLeaderboard(mode, difficulty, 0) // 0 = no limit
			if err == nil {
				key := mode + "_" + difficulty
				counts[key] = len(entries)
				totalEntries += len(entries)
			}
		}
	}

	status := gin.H{
		"total_entries": totalEntries,
		"breakdown": gin.H{
			"challenge_easy": counts["challenge_easy"],
			"challenge_hard": counts["challenge_hard"],
			"streak_easy":    counts["streak_easy"],
			"streak_hard":    counts["streak_hard"],
			"zero_easy":      counts["zero_easy"],
			"zero_hard":      counts["zero_hard"],
		},
		"storage":       "database",
		"database_path": "./data/carguessr.db",
	}

	c.JSON(http.StatusOK, status)
}
