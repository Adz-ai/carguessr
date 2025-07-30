package game

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"autotraderguesser/internal/cache"
	"autotraderguesser/internal/models"
	"autotraderguesser/internal/scraper"
)

type Handler struct {
	scraper         *scraper.Scraper
	listings        map[string]*models.Car
	bonhamsListings map[string]*models.BonhamsCar // Store original Bonhams data
	leaderboard     []models.LeaderboardEntry
	mu              sync.RWMutex
	zeroScores      map[string]float64
	streakScores    map[string]int
	refreshTicker   *time.Ticker
}

func NewHandler() *Handler {
	h := &Handler{
		scraper:         scraper.New(),
		listings:        make(map[string]*models.Car),
		bonhamsListings: make(map[string]*models.BonhamsCar),
		leaderboard:     make([]models.LeaderboardEntry, 0),
		zeroScores:      make(map[string]float64),
		streakScores:    make(map[string]int),
	}

	// Initialize with cached or fresh data
	h.initializeListings()

	// Start automatic refresh timer (every 12 hours)
	h.startAutoRefresh()

	return h
}

// GetRandomListing godoc
// @Summary Get a random car listing for the game
// @Description Returns a random car listing with the price hidden (set to 0) for the guessing game
// @Tags game
// @Produce json
// @Success 200 {object} models.Car
// @Failure 404 {object} map[string]string "error: No listings available"
// @Router /api/random-listing [get]
func (h *Handler) GetRandomListing(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Get all listing IDs
	ids := make([]string, 0, len(h.listings))
	for id := range h.listings {
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No listings available"})
		return
	}

	// Select random listing
	randomID := ids[rand.Intn(len(ids))]
	listing := h.listings[randomID]

	// Create a copy without the price
	displayListing := *listing
	displayListing.Price = 0

	c.JSON(http.StatusOK, displayListing)
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
// @Description Submit a price guess and get feedback on accuracy, score, and game status
// @Tags game
// @Accept json
// @Produce json
// @Param guess body models.GuessRequest true "Price guess data"
// @Success 200 {object} models.GuessResponse
// @Failure 400 {object} map[string]string "error: Invalid request"
// @Failure 404 {object} map[string]string "error: Listing not found"
// @Router /api/check-guess [post]
func (h *Handler) CheckGuess(c *gin.Context) {
	var req models.GuessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.mu.RLock()
	listing, exists := h.listings[req.ListingID]
	h.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found"})
		return
	}

	// Calculate difference and percentage
	difference := math.Abs(listing.Price - req.GuessedPrice)
	percentage := (difference / listing.Price) * 100

	response := models.GuessResponse{
		ActualPrice:  listing.Price,
		GuessedPrice: req.GuessedPrice,
		Difference:   difference,
		Percentage:   percentage,
		OriginalURL:  listing.OriginalURL,
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
// @Summary Test the car scraper directly
// @Description Tests the AutoTrader scraper and returns up to 10 cars with full details
// @Tags debug
// @Produce json
// @Success 200 {object} map[string]interface{} "message, count, and cars array"
// @Failure 500 {object} map[string]string "error and message"
// @Router /api/test-scraper [get]
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
// @Description Returns information about the current data source being used for car listings
// @Tags debug
// @Produce json
// @Success 200 {object} map[string]interface{} "data_source, total_listings, and description"
// @Router /api/data-source [get]
func (h *Handler) GetDataSource(c *gin.Context) {
	h.mu.RLock()
	totalListings := len(h.listings)
	h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"data_source":    "bonhams_auctions",
		"total_listings": totalListings,
		"description":    "Real Bonhams Car Auction results",
	})
}

// GetAllListings godoc
// @Summary Get all available car listings
// @Description Returns all car listings currently loaded in the system with full details including prices
// @Tags listings
// @Produce json
// @Success 200 {object} map[string]interface{} "count and cars array"
// @Router /api/listings [get]
func (h *Handler) GetAllListings(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	cars := make([]*models.Car, 0, len(h.listings))
	for _, car := range h.listings {
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

	for _, bonhamsCar := range cachedListings {
		h.bonhamsListings[bonhamsCar.ID] = bonhamsCar
		h.listings[bonhamsCar.ID] = bonhamsCar.ToStandardCar()
	}
}

func (h *Handler) refreshListings() {
	fmt.Println("🔄 Refreshing listings from Bonhams Car Auctions...")

	// Get fresh Bonhams data
	bonhamsCars, err := h.scraper.GetBonhamsListings(50)
	if err == nil && len(bonhamsCars) > 0 {
		h.mu.Lock()
		// Clear existing listings
		h.listings = make(map[string]*models.Car)
		h.bonhamsListings = make(map[string]*models.BonhamsCar)

		// Load new listings
		for _, bonhamsCar := range bonhamsCars {
			h.bonhamsListings[bonhamsCar.ID] = bonhamsCar
			h.listings[bonhamsCar.ID] = bonhamsCar.ToStandardCar()
		}
		h.mu.Unlock()

		// Save to cache
		if err := cache.SaveToCache(bonhamsCars); err != nil {
			fmt.Printf("⚠️ Failed to save cache: %v\n", err)
		}

		fmt.Printf("✅ Refreshed %d cars from Bonhams Car Auctions\n", len(bonhamsCars))
		return
	}

	// Fallback to standard scraper
	cars, err := h.scraper.GetCarListings(50)
	if err == nil && len(cars) > 0 {
		h.mu.Lock()
		h.listings = make(map[string]*models.Car)
		for _, car := range cars {
			h.listings[car.ID] = car
		}
		h.mu.Unlock()
		fmt.Printf("✅ Loaded %d cars from fallback scraper\n", len(cars))
		return
	}

	fmt.Printf("❌ All scrapers failed: %v\n", err)
	fmt.Println("Loading fallback cars for testing...")

	// Fallback to static mock data
	mockCars := []models.Car{
		{
			ID:           "mock1",
			Make:         "Volkswagen",
			Model:        "Golf",
			Year:         2020,
			Price:        18995,
			Mileage:      28000,
			FuelType:     "Petrol",
			Images:       []string{"https://images.unsplash.com/photo-1609521263047-f8f205293f24?w=600&h=400&fit=crop"},
			Registration: "70 VWG",
			Owners:       "2 previous owners",
			BodyType:     "Hatchback",
			Engine:       "1.4 TSI",
			Gearbox:      "Manual",
			Doors:        "5",
			Seats:        "5",
			BodyColour:   "Silver",
		},
		{
			ID:           "mock2",
			Make:         "Nissan",
			Model:        "Qashqai",
			Year:         2019,
			Price:        22750,
			Mileage:      34000,
			FuelType:     "Diesel",
			Images:       []string{"https://images.unsplash.com/photo-1606611013016-969c19ba1fbb?w=600&h=400&fit=crop"},
			Registration: "69 NIS",
			Owners:       "1 previous owner",
			BodyType:     "SUV",
			Engine:       "1.5 dCi",
			Gearbox:      "Automatic",
			Doors:        "5",
			Seats:        "5",
			BodyColour:   "Blue",
		},
		{
			ID:           "mock3",
			Make:         "BMW",
			Model:        "3 Series",
			Year:         2021,
			Price:        34995,
			Mileage:      19000,
			FuelType:     "Petrol",
			Images:       []string{"https://images.unsplash.com/photo-1555215695-3004980ad54e?w=600&h=400&fit=crop"},
			Registration: "21 BMW",
			Owners:       "1 previous owner",
			BodyType:     "Saloon",
			Engine:       "2.0 TwinPower Turbo",
			Gearbox:      "Automatic",
			Doors:        "4",
			Seats:        "5",
			BodyColour:   "White",
		},
		{
			ID:           "mock4",
			Make:         "Ford",
			Model:        "Focus",
			Year:         2018,
			Price:        12495,
			Mileage:      45000,
			FuelType:     "Petrol",
			Images:       []string{"https://images.unsplash.com/photo-1610647752706-3bb12232b3ab?w=600&h=400&fit=crop"},
			Registration: "68 FOR",
			Owners:       "2 previous owners",
			BodyType:     "Hatchback",
			Engine:       "1.0 EcoBoost",
			Gearbox:      "Manual",
			Doors:        "5",
			Seats:        "5",
			BodyColour:   "Black",
		},
		{
			ID:           "mock5",
			Make:         "Audi",
			Model:        "A4",
			Year:         2020,
			Price:        28950,
			Mileage:      25000,
			FuelType:     "Diesel",
			Images:       []string{"https://images.unsplash.com/photo-1606664515524-ed2f786a0bd6?w=600&h=400&fit=crop"},
			Registration: "70 AUD",
			Owners:       "1 previous owner",
			BodyType:     "Saloon",
			Engine:       "2.0 TDI",
			Gearbox:      "Automatic",
			Doors:        "4",
			Seats:        "5",
			BodyColour:   "Grey",
		},
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for i := range mockCars {
		h.listings[mockCars[i].ID] = &mockCars[i]
	}
}

// startAutoRefresh starts a background goroutine that refreshes listings every 12 hours
func (h *Handler) startAutoRefresh() {
	// Create ticker for 12-hour intervals
	h.refreshTicker = time.NewTicker(12 * time.Hour)

	go func() {
		for range h.refreshTicker.C {
			fmt.Println("⏰ Auto-refresh triggered (12 hours elapsed)")
			h.refreshListings()
		}
	}()

	fmt.Println("🔄 Auto-refresh scheduled every 12 hours")
}

// StopAutoRefresh stops the automatic refresh ticker (useful for cleanup)
func (h *Handler) StopAutoRefresh() {
	if h.refreshTicker != nil {
		h.refreshTicker.Stop()
		fmt.Println("🛑 Auto-refresh stopped")
	}
}

// ManualRefresh godoc
// @Summary Manually refresh car listings
// @Description Triggers a manual refresh of car listings from Bonhams
// @Tags admin
// @Produce json
// @Success 200 {object} map[string]interface{} "message and count"
// @Failure 500 {object} map[string]string "error message"
// @Router /api/refresh-listings [post]
func (h *Handler) ManualRefresh(c *gin.Context) {
	fmt.Println("🔄 Manual refresh requested")

	go func() {
		h.refreshListings()
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Manual refresh started in background",
		"status":  "refreshing",
	})
}

// GetCacheStatus godoc
// @Summary Get cache status information
// @Description Returns information about the current cache status and age
// @Tags debug
// @Produce json
// @Success 200 {object} map[string]interface{} "cache status information"
// @Router /api/cache-status [get]
func (h *Handler) GetCacheStatus(c *gin.Context) {
	age, err := cache.GetCacheAge()
	expired := cache.IsCacheExpired()

	h.mu.RLock()
	totalListings := len(h.listings)
	bonhamsListings := len(h.bonhamsListings)
	h.mu.RUnlock()

	status := gin.H{
		"cache_expired":    expired,
		"total_listings":   totalListings,
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

func generateSessionID() string {
	rand.Seed(time.Now().UnixNano())
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
