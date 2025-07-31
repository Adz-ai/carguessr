package game

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"autotraderguesser/internal/cache"
	"autotraderguesser/internal/models"
	"autotraderguesser/internal/scraper"
)

const ListingAmount int = 250

type Handler struct {
	scraper           *scraper.Scraper
	bonhamsListings   map[string]*models.BonhamsCar // Only store Bonhams data
	leaderboard       []models.LeaderboardEntry
	mu                sync.RWMutex
	zeroScores        map[string]float64
	streakScores      map[string]int
	challengeSessions map[string]*models.ChallengeSession // Store challenge sessions
	refreshTicker     *time.Ticker
}

func NewHandler() *Handler {
	h := &Handler{
		scraper:           scraper.New(),
		bonhamsListings:   make(map[string]*models.BonhamsCar),
		leaderboard:       make([]models.LeaderboardEntry, 0),
		zeroScores:        make(map[string]float64),
		streakScores:      make(map[string]int),
		challengeSessions: make(map[string]*models.ChallengeSession),
	}

	// Initialize with cached or fresh data
	h.initializeListings()

	// Start automatic refresh timer (every 7 days)
	h.startAutoRefresh()
	fmt.Printf("🔄 Auto-refresh scheduled every 7 days (next refresh: %s)\n",
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
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Get all Bonhams listing IDs
	ids := make([]string, 0, len(h.bonhamsListings))
	for id := range h.bonhamsListings {
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No listings available"})
		return
	}

	// Select random listing
	randomID := ids[rand.Intn(len(ids))]
	bonhamsListing := h.bonhamsListings[randomID]

	// Convert to enhanced format and hide price
	enhancedListing := bonhamsListing.ToEnhancedCar()
	enhancedListing.Price = 0

	c.JSON(http.StatusOK, enhancedListing)
}

// GetRandomEnhancedListing godoc
// @Summary Get a random car listing with all Bonhams characteristics
// @Description Returns a random car listing with full auction details and characteristics, price hidden for guessing
// @Tags game
// @Produce json
// @Success 200 {object} models.EnhancedCar
// @Failure 404 {object} map[string]string "error: No listings available"
// @Router /api/random-enhanced-listing [get]
func (h *Handler) GetRandomEnhancedListing(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Get all Bonhams listing IDs
	ids := make([]string, 0, len(h.bonhamsListings))
	for id := range h.bonhamsListings {
		ids = append(ids, id)
	}

	// Select random listing
	randomID := ids[rand.Intn(len(ids))]
	bonhamsListing := h.bonhamsListings[randomID]

	// Convert to enhanced format and hide price
	enhancedListing := bonhamsListing.ToEnhancedCar()
	enhancedListing.Price = 0

	c.JSON(http.StatusOK, enhancedListing)
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
	bonhamsListing, exists := h.bonhamsListings[req.ListingID]
	h.mu.RUnlock()

	if !exists {
		log.Printf("CheckGuess: Listing not found - ID: %s", req.ListingID)
		log.Printf("CheckGuess: Available listings: %d", len(h.bonhamsListings))
		c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found", "requestedId": req.ListingID})
		return
	}

	// Calculate difference and percentage
	difference := math.Abs(bonhamsListing.Price - req.GuessedPrice)
	percentage := (difference / bonhamsListing.Price) * 100

	response := models.GuessResponse{
		ActualPrice:  bonhamsListing.Price,
		GuessedPrice: req.GuessedPrice,
		Difference:   difference,
		Percentage:   percentage,
		OriginalURL:  bonhamsListing.OriginalURL,
	}

	// Handle game mode logic
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		sessionID = generateSessionID()
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
// @Description Returns the leaderboard optionally filtered by game mode
// @Tags game
// @Produce json
// @Param mode query string false "Game mode filter (zero or streak)"
// @Success 200 {array} models.LeaderboardEntry
// @Router /api/leaderboard [get]
func (h *Handler) GetLeaderboard(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	gameMode := c.Query("mode")

	// Filter leaderboard by game mode if specified
	filtered := make([]models.LeaderboardEntry, 0)
	for _, entry := range h.leaderboard {
		if gameMode == "" || entry.GameMode == gameMode {
			filtered = append(filtered, entry)
		}
	}

	c.JSON(http.StatusOK, filtered)
}

// TestScraper godoc
// @Summary Test the car scraper directly (Admin Only)
// @Description Tests the AutoTrader scraper and returns up to 10 cars with full details. This is an expensive operation. Requires admin authentication.
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
// @Description Returns information about the data source (Bonhams Car Auctions) including total listings count
// @Tags debug
// @Produce json
// @Success 200 {object} map[string]interface{} "data_source: bonhams_auctions, total_listings: count, description: Real Bonhams Car Auction results"
// @Router /api/data-source [get]
func (h *Handler) GetDataSource(c *gin.Context) {
	h.mu.RLock()
	totalListings := len(h.bonhamsListings)
	h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"data_source":    "bonhams_auctions",
		"total_listings": totalListings,
		"description":    "Real Bonhams Car Auction results",
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

func (h *Handler) initializeListings() {
	// Try to load from cache first
	if cachedListings, found := cache.LoadFromCache(); found {
		h.loadListingsFromCache(cachedListings)
		return
	}

	// Cache miss or expired, scrape fresh data
	h.refreshListings()
}

func (h *Handler) loadListingsFromCache(cachedListings []*models.BonhamsCar) {
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

func (h *Handler) refreshListings() {
	fmt.Println("🔄 Refreshing listings from Bonhams Car Auctions...")

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
		if err := cache.SaveToCache(validCars); err != nil {
			fmt.Printf("⚠️ Failed to save cache: %v\n", err)
		}

		fmt.Printf("✅ Refreshed %d valid cars from Bonhams Car Auctions (%d filtered out)\n", len(validCars), len(bonhamsCars)-len(validCars))
		return
	}

	fmt.Printf("❌ Bonhams scraper failed: %v\n", err)
	fmt.Println("Creating minimal mock data for testing...")

	// Fallback to static mock data
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

// startAutoRefresh starts a background goroutine that refreshes listings every 7 days
func (h *Handler) startAutoRefresh() {
	// Create ticker for 7-day intervals
	h.refreshTicker = time.NewTicker(7 * 24 * time.Hour)

	go func() {
		for range h.refreshTicker.C {
			fmt.Println("⏰ Auto-refresh triggered (7 days elapsed)")
			// Run refresh in background to avoid blocking gameplay
			go h.refreshListingsAsync()
		}
	}()

	fmt.Println("🔄 Auto-refresh scheduled every 24 hours")
}

// refreshListingsAsync performs a non-blocking refresh that doesn't interrupt gameplay
func (h *Handler) refreshListingsAsync() {
	fmt.Println("🔄 Starting background refresh (non-blocking)...")

	// Get fresh Bonhams data (this may take a few minutes) - 250 cars with parallel scraping
	bonhamsCars, err := h.scraper.GetBonhamsListings(ListingAmount)
	if err != nil {
		fmt.Printf("❌ Background refresh failed: %v\n", err)
		return
	}

	if len(bonhamsCars) == 0 {
		fmt.Println("❌ Background refresh returned no cars")
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
	if err := cache.SaveToCache(validCars); err != nil {
		fmt.Printf("⚠️ Failed to save cache during background refresh: %v\n", err)
	}

	fmt.Printf("✅ Background refresh complete: %d cars updated (was %d, filtered %d)\n",
		len(validCars), oldCount, len(bonhamsCars)-len(validCars))
}

// StopAutoRefresh stops the automatic refresh ticker (useful for cleanup)
func (h *Handler) StopAutoRefresh() {
	if h.refreshTicker != nil {
		h.refreshTicker.Stop()
		fmt.Println("🛑 Auto-refresh stopped")
	}
}

// ManualRefresh godoc
// @Summary Manually refresh car listings (Admin Only)
// @Description Triggers a non-blocking background refresh of car listings from Bonhams. Requires admin authentication and has a 30-minute cooldown between requests. Game continues normally during refresh.
// @Tags admin
// @Security AdminKey
// @Produce json
// @Success 200 {object} map[string]interface{} "message: refresh started, status: refreshing, note: game continues normally"
// @Failure 401 {object} map[string]string "error: Unauthorized - Admin key required"
// @Failure 429 {object} map[string]string "error: Too Many Requests - Rate limited or refresh cooldown active"
// @Router /api/admin/refresh-listings [post]
func (h *Handler) ManualRefresh(c *gin.Context) {
	fmt.Println("🔄 Manual refresh requested")

	go func() {
		h.refreshListingsAsync()
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Manual refresh started in background (non-blocking)",
		"status":  "refreshing",
		"note":    "Game will continue normally while refresh happens in background",
	})
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
	age, err := cache.GetCacheAge()
	expired := cache.IsCacheExpired()

	h.mu.RLock()
	bonhamsListings := len(h.bonhamsListings)
	h.mu.RUnlock()

	status := gin.H{
		"cache_expired":    expired,
		"total_listings":   bonhamsListings,
		"bonhams_listings": bonhamsListings,
		"next_refresh_in":  "up to 12 hours",
	}

	if err == nil {
		status["cache_age"] = age.Round(time.Minute).String()
		status["cache_age_hours"] = age.Hours()
	} else {
		status["cache_age"] = "no cache file"
	}

	c.JSON(http.StatusOK, status)
}

// StartChallenge godoc
// @Summary Start a new Challenge Mode session
// @Description Starts a new 10-car challenge session with GeoGuessr-style scoring. Players get up to 5000 points per car based on guess accuracy. Rate limited to 60 requests per minute per IP.
// @Tags challenge
// @Produce json
// @Success 200 {object} models.ChallengeSession "sessionId, cars array (10 cars with prices hidden), currentCar: 0, totalScore: 0"
// @Failure 404 {object} map[string]string "error: Not enough cars available for challenge mode"
// @Failure 429 {object} map[string]string "error: Too Many Requests - Rate limited"
// @Router /api/challenge/start [post]
func (h *Handler) StartChallenge(c *gin.Context) {
	h.mu.RLock()

	// Need at least 10 cars for a challenge
	if len(h.bonhamsListings) < 10 {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "Not enough cars available for challenge mode"})
		return
	}

	// Select 10 random cars
	var allCars []*models.BonhamsCar
	for _, car := range h.bonhamsListings {
		allCars = append(allCars, car)
	}
	h.mu.RUnlock() // Release read lock before processing

	// Shuffle and select 10
	rand.Shuffle(len(allCars), func(i, j int) {
		allCars[i], allCars[j] = allCars[j], allCars[i]
	})

	selectedCars := make([]*models.EnhancedCar, 10)
	for i := 0; i < 10; i++ {
		enhancedCar := allCars[i].ToEnhancedCar()
		enhancedCar.Price = 0 // Hide price for guessing
		selectedCars[i] = enhancedCar
	}

	// Create challenge session
	sessionID := generateSessionID()
	session := &models.ChallengeSession{
		SessionID:  sessionID,
		Cars:       selectedCars,
		CurrentCar: 0,
		Guesses:    make([]models.ChallengeGuess, 0),
		TotalScore: 0,
		IsComplete: false,
		StartTime:  time.Now().Format(time.RFC3339),
	}

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

	h.mu.RLock()
	session, exists := h.challengeSessions[sessionID]
	h.mu.RUnlock()

	if !exists {
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

	h.mu.Lock()
	session, exists := h.challengeSessions[sessionID]
	if !exists {
		h.mu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "Challenge session not found"})
		return
	}

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

	// Find the original car with the actual price
	for _, bonhamsCar := range h.bonhamsListings {
		if bonhamsCar.ID == currentCar.ID {
			actualPrice = bonhamsCar.Price
			originalURL = bonhamsCar.OriginalURL
			break
		}
	}

	if actualPrice == 0 {
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

func generateSessionID() string {
	rand.Seed(time.Now().UnixNano())
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
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
