package models

// Car represents a vehicle listing
type Car struct {
	ID          string   `json:"id"`
	Make        string   `json:"make"`
	Model       string   `json:"model"`
	Year        int      `json:"year"`
	Price       float64  `json:"price"`
	Images      []string `json:"images"`
	OriginalURL string   `json:"originalUrl,omitempty"`

	// Overview section fields
	Mileage       int    `json:"mileage"`
	Registration  string `json:"registration,omitempty"`
	Owners        string `json:"owners,omitempty"`
	FuelType      string `json:"fuelType"`
	BodyType      string `json:"bodyType,omitempty"`
	Engine        string `json:"engine,omitempty"`
	Gearbox       string `json:"gearbox,omitempty"`
	Doors         string `json:"doors,omitempty"`
	Seats         string `json:"seats,omitempty"`
	BodyColour    string `json:"bodyColour,omitempty"`
	EmissionClass string `json:"emissionClass,omitempty"`

	// Placeholder for future fields
}

// BonhamsCar represents a comprehensive car model with all Bonhams auction data
type BonhamsCar struct {
	// Basic info
	ID          string   `json:"id"`
	Make        string   `json:"make"`
	Model       string   `json:"model"`
	Year        int      `json:"year"`
	Price       float64  `json:"price"`
	Images      []string `json:"images"`
	OriginalURL string   `json:"originalUrl,omitempty"`

	// Specifications from data-qa attributes
	Mileage       string `json:"mileage,omitempty"`       // Keep as string to preserve formatting like "41,565 Miles"
	Engine        string `json:"engine,omitempty"`        // e.g., "5999cc"
	Gearbox       string `json:"gearbox,omitempty"`       // e.g., "semi", "manual", "automatic"
	ExteriorColor string `json:"exteriorColor,omitempty"` // e.g., "Blue"
	InteriorColor string `json:"interiorColor,omitempty"` // e.g., "Crema"
	Steering      string `json:"steering,omitempty"`      // e.g., "Right-hand drive"
	FuelType      string `json:"fuelType,omitempty"`      // e.g., "Petrol"

	// Key facts (description points)
	KeyFacts []string `json:"keyFacts,omitempty"`

	// Additional extracted info
	MileageNumeric int    `json:"mileageNumeric,omitempty"` // Numeric version for calculations
	Description    string `json:"description,omitempty"`    // Combined description
}

// ToStandardCar converts BonhamsCar to the standard Car model for compatibility
func (bc *BonhamsCar) ToStandardCar() *Car {
	return &Car{
		ID:          bc.ID,
		Make:        bc.Make,
		Model:       bc.Model,
		Year:        bc.Year,
		Price:       bc.Price,
		Images:      bc.Images,
		OriginalURL: bc.OriginalURL,
		Mileage:     bc.MileageNumeric,
		FuelType:    bc.FuelType,
		Engine:      bc.Engine,
		Gearbox:     bc.Gearbox,
		BodyColour:  bc.ExteriorColor,
	}
}

// ToEnhancedCar converts BonhamsCar to EnhancedCar with all characteristics
func (bc *BonhamsCar) ToEnhancedCar() *EnhancedCar {
	return &EnhancedCar{
		// Standard fields
		ID:          bc.ID,
		Make:        bc.Make,
		Model:       bc.Model,
		Year:        bc.Year,
		Price:       bc.Price,
		Images:      bc.Images,
		OriginalURL: bc.OriginalURL,

		// Standard overview fields
		Mileage:    bc.MileageNumeric,
		FuelType:   bc.FuelType,
		Engine:     bc.Engine,
		Gearbox:    bc.Gearbox,
		BodyColour: bc.ExteriorColor,

		// Enhanced Bonhams fields
		MileageFormatted: bc.Mileage,
		ExteriorColor:    bc.ExteriorColor,
		InteriorColor:    bc.InteriorColor,
		Steering:         bc.Steering,
		KeyFacts:         bc.KeyFacts,
		AuctionDetails:   true,
	}
}

// EnhancedCar represents a car with all Bonhams characteristics for the UI
type EnhancedCar struct {
	// Standard Car fields
	ID          string   `json:"id"`
	Make        string   `json:"make"`
	Model       string   `json:"model"`
	Year        int      `json:"year"`
	Price       float64  `json:"price"` // Will be 0 for guessing
	Images      []string `json:"images"`
	OriginalURL string   `json:"originalUrl,omitempty"`

	// Standard overview fields
	Mileage       int    `json:"mileage"`
	Registration  string `json:"registration,omitempty"`
	Owners        string `json:"owners,omitempty"`
	FuelType      string `json:"fuelType"`
	BodyType      string `json:"bodyType,omitempty"`
	Engine        string `json:"engine,omitempty"`
	Gearbox       string `json:"gearbox,omitempty"`
	Doors         string `json:"doors,omitempty"`
	Seats         string `json:"seats,omitempty"`
	BodyColour    string `json:"bodyColour,omitempty"`
	EmissionClass string `json:"emissionClass,omitempty"`

	// Enhanced Bonhams fields
	MileageFormatted string   `json:"mileageFormatted,omitempty"` // "41,565 Miles"
	ExteriorColor    string   `json:"exteriorColor,omitempty"`    // More detailed color
	InteriorColor    string   `json:"interiorColor,omitempty"`    // Interior details
	Steering         string   `json:"steering,omitempty"`         // "Right-hand drive"
	KeyFacts         []string `json:"keyFacts,omitempty"`         // Auction key facts
	Description      string   `json:"description,omitempty"`      // Combined description
	AuctionDetails   bool     `json:"auctionDetails"`             // Flag indicating this is auction data
}

// GuessRequest represents a price guess from the user
type GuessRequest struct {
	ListingID    string  `json:"listingId" binding:"required"`
	GuessedPrice float64 `json:"guessedPrice" binding:"required,min=0"`
	GameMode     string  `json:"gameMode" binding:"required,oneof=zero streak"`
}

// ChallengeGuessRequest represents a price guess in challenge mode
type ChallengeGuessRequest struct {
	GuessedPrice float64 `json:"guessedPrice" binding:"required,min=0"`
}

// GuessResponse represents the result of a guess
type GuessResponse struct {
	Correct      bool    `json:"correct"`
	ActualPrice  float64 `json:"actualPrice"`
	GuessedPrice float64 `json:"guessedPrice"`
	Difference   float64 `json:"difference"`
	Percentage   float64 `json:"percentage"`
	Score        int     `json:"score"`
	GameOver     bool    `json:"gameOver"`
	Message      string  `json:"message"`
	OriginalURL  string  `json:"originalUrl,omitempty"`
}

// ChallengeSession represents a 10-car challenge game session
type ChallengeSession struct {
	SessionID     string           `json:"sessionId"`
	Cars          []*EnhancedCar   `json:"cars"`
	CurrentCar    int              `json:"currentCar"`
	Guesses       []ChallengeGuess `json:"guesses"`
	TotalScore    int              `json:"totalScore"`
	IsComplete    bool             `json:"isComplete"`
	StartTime     string           `json:"startTime"`
	CompletedTime string           `json:"completedTime,omitempty"`
}

// ChallengeGuess represents a single guess in challenge mode
type ChallengeGuess struct {
	CarID        string  `json:"carId"`
	GuessedPrice float64 `json:"guessedPrice"`
	ActualPrice  float64 `json:"actualPrice"`
	Difference   float64 `json:"difference"`
	Percentage   float64 `json:"percentage"`
	Points       int     `json:"points"`
}

// ChallengeResponse represents the response after submitting a challenge guess
type ChallengeResponse struct {
	ChallengeGuess
	IsLastCar       bool   `json:"isLastCar"`
	NextCarNumber   int    `json:"nextCarNumber,omitempty"`
	TotalScore      int    `json:"totalScore"`
	SessionComplete bool   `json:"sessionComplete"`
	Message         string `json:"message"`
	OriginalURL     string `json:"originalUrl,omitempty"`
}

// LeaderboardEntry represents a high score entry
type LeaderboardEntry struct {
	Name     string `json:"name"`
	Score    int    `json:"score"`
	GameMode string `json:"gameMode"`
	Date     string `json:"date"`
}
