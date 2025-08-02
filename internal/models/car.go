package models

import (
	"regexp"
	"strconv"
	"strings"
)

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
	SaleDate       string `json:"saleDate,omitempty"`       // Sale date in format "29 Jul 2025"
	Location       string `json:"location,omitempty"`       // Vehicle location e.g. "Bournemouth, Dorset, United Kingdom"
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
		SaleDate:         bc.SaleDate,
		Location:         bc.Location,
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
	SaleDate         string   `json:"saleDate,omitempty"`         // Sale date in format "29 Jul 2025"
	Location         string   `json:"location,omitempty"`         // Vehicle location e.g. "Bournemouth, Dorset, United Kingdom"

	// Enhanced Lookers fields
	Trim      string `json:"trim,omitempty"`      // Trim level and engine details from title
	FullTitle string `json:"fullTitle,omitempty"` // Complete original title for Easy mode

	AuctionDetails bool `json:"auctionDetails"` // Flag indicating this is auction data
}

// LookersCar represents a car from Lookers.co.uk (Easy mode)
type LookersCar struct {
	// Basic info
	ID          string   `json:"id"`
	Make        string   `json:"make"`
	Model       string   `json:"model"`
	Year        int      `json:"year"`
	Price       float64  `json:"price"`
	Images      []string `json:"images"`
	OriginalURL string   `json:"originalUrl,omitempty"`

	// Lookers specific fields
	Title           string            `json:"title"`
	Location        string            `json:"location"`
	BodyType        string            `json:"bodyType"`
	SortOrder       string            `json:"sortOrder"`
	Characteristics map[string]string `json:"characteristics"`
}

// ToEnhancedCar converts LookersCar to EnhancedCar for the UI
func (lc *LookersCar) ToEnhancedCar() *EnhancedCar {
	// Parse mileage
	mileageNumeric := 0
	if mileageStr, ok := lc.Characteristics["Mileage"]; ok {
		// Extract numeric value from "75,851 miles"
		re := regexp.MustCompile(`[\d,]+`)
		if match := re.FindString(mileageStr); match != "" {
			cleanedMileage := strings.ReplaceAll(match, ",", "")
			if val, err := strconv.Atoi(cleanedMileage); err == nil {
				mileageNumeric = val
			}
		}
	}

	// Parse year if not set
	year := lc.Year
	if year == 0 {
		if yearStr, ok := lc.Characteristics["Year"]; ok {
			if val, err := strconv.Atoi(yearStr); err == nil {
				year = val
			}
		}
	}

	// Extract trim information from title
	// Format: "LAND ROVER RANGE ROVER ESTATE - 2025 3.0 P460E Autobiography 4Dr Auto"
	var trim string
	if strings.Contains(lc.Title, " - ") {
		parts := strings.SplitN(lc.Title, " - ", 2)
		if len(parts) == 2 {
			trim = parts[1]
			// Remove year from trim if it's at the beginning (avoid redundancy)
			trimParts := strings.Fields(trim)
			if len(trimParts) > 0 {
				// Check if first part is a year (4 digits)
				if len(trimParts[0]) == 4 {
					if _, err := strconv.Atoi(trimParts[0]); err == nil {
						// Remove the year and rejoin
						if len(trimParts) > 1 {
							trim = strings.Join(trimParts[1:], " ")
						}
					}
				}
			}
		}
	}

	return &EnhancedCar{
		// Standard fields
		ID:          lc.ID,
		Make:        lc.Make,
		Model:       lc.Model,
		Year:        year,
		Price:       lc.Price,
		Images:      lc.Images,
		OriginalURL: lc.OriginalURL,

		// Standard overview fields
		Mileage:      mileageNumeric,
		Registration: lc.Characteristics["Registration"],
		Owners:       lc.Characteristics["Owners"],
		FuelType:     lc.Characteristics["Fuel Type"],
		BodyType:     lc.BodyType,
		Engine:       lc.Characteristics["Engine Size"],
		Gearbox:      lc.Characteristics["Transmission"],
		Doors:        lc.Characteristics["Doors"],
		BodyColour:   lc.Characteristics["Color"],

		// Enhanced fields
		MileageFormatted: lc.Characteristics["Mileage"],
		Location:         lc.Location,
		FullTitle:        lc.Title, // Store complete original title
		Trim:             trim,     // Extract trim from after hyphen
		AuctionDetails:   false,    // This is not auction data
	}
}

// GuessRequest represents a price guess from the user
type GuessRequest struct {
	ListingID    string  `json:"listingId" binding:"required"`
	GuessedPrice float64 `json:"guessedPrice" binding:"required,min=0"`
	GameMode     string  `json:"gameMode" binding:"required,oneof=zero streak challenge"`
	Difficulty   string  `json:"difficulty" binding:"omitempty,oneof=easy hard"` // Default to hard for backward compatibility
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
	Name       string `json:"name" binding:"required,min=1,max=20"`
	Score      int    `json:"score"`
	GameMode   string `json:"gameMode"`
	Difficulty string `json:"difficulty,omitempty"` // "easy" or "hard", defaults to "hard" for backward compatibility
	Date       string `json:"date"`
	ID         string `json:"id,omitempty"`
}

// LeaderboardSubmissionRequest represents a request to submit a score to the leaderboard
type LeaderboardSubmissionRequest struct {
	Name       string `json:"name" binding:"required,min=1,max=20"`
	Score      int    `json:"score" binding:"required,min=0"`
	GameMode   string `json:"gameMode" binding:"required,oneof=streak challenge"`
	Difficulty string `json:"difficulty,omitempty" binding:"omitempty,oneof=easy hard"` // Default to hard for backward compatibility
	SessionID  string `json:"sessionId,omitempty"`
}
