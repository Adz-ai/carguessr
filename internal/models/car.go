package models

// Car represents a vehicle listing
type Car struct {
	ID             string   `json:"id"`
	Make           string   `json:"make"`
	Model          string   `json:"model"`
	Year           int      `json:"year"`
	Price          float64  `json:"price"`
	Images         []string `json:"images"`
	OriginalURL    string   `json:"originalUrl,omitempty"`
	
	// Overview section fields
	Mileage        int      `json:"mileage"`
	Registration   string   `json:"registration,omitempty"`
	Owners         string   `json:"owners,omitempty"`
	FuelType       string   `json:"fuelType"`
	BodyType       string   `json:"bodyType,omitempty"`
	Engine         string   `json:"engine,omitempty"`
	Gearbox        string   `json:"gearbox,omitempty"`
	Doors          string   `json:"doors,omitempty"`
	Seats          string   `json:"seats,omitempty"`
	BodyColour     string   `json:"bodyColour,omitempty"`
	EmissionClass  string   `json:"emissionClass,omitempty"`
}

// GuessRequest represents a price guess from the user
type GuessRequest struct {
	ListingID    string  `json:"listingId" binding:"required"`
	GuessedPrice float64 `json:"guessedPrice" binding:"required,min=0"`
	GameMode     string  `json:"gameMode" binding:"required,oneof=zero streak"`
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

// LeaderboardEntry represents a high score entry
type LeaderboardEntry struct {
	Name     string `json:"name"`
	Score    int    `json:"score"`
	GameMode string `json:"gameMode"`
	Date     string `json:"date"`
}