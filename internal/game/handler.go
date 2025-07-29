package game

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"autotraderguesser/internal/models"
	"autotraderguesser/internal/scraper"
)

type Handler struct {
	scraper      *scraper.Scraper
	listings     map[string]*models.Car
	leaderboard  []models.LeaderboardEntry
	mu           sync.RWMutex
	zeroScores   map[string]float64
	streakScores map[string]int
}

func NewHandler() *Handler {
	h := &Handler{
		scraper:      scraper.New(),
		listings:     make(map[string]*models.Car),
		leaderboard:  make([]models.LeaderboardEntry, 0),
		zeroScores:   make(map[string]float64),
		streakScores: make(map[string]int),
	}

	// Initialize with some mock data
	h.initializeMockData()

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
		"data_source":    "motors_live",
		"total_listings": totalListings,
		"description":    "Real Motors.co.uk car listings",
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

func (h *Handler) initializeMockData() {
	// Load car data using the unified scraper interface
	fmt.Println("Starting Motors.co.uk scraper...")
	cars, err := h.scraper.GetCarListings(50) // Get 50 cars with variety across makes
	if err == nil && len(cars) > 0 {
		h.mu.Lock()
		defer h.mu.Unlock()
		for _, car := range cars {
			h.listings[car.ID] = car
		}
		fmt.Printf("✅ Loaded %d real cars from Motors.co.uk\n", len(cars))
		return
	}

	fmt.Printf("❌ Motors scraper failed: %v\n", err)
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

func generateSessionID() string {
	rand.Seed(time.Now().UnixNano())
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
