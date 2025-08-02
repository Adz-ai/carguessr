package scraper

import (
	"fmt"

	"autotraderguesser/internal/models"
)

type Scraper struct {
	bonhamsScraper *BonhamsScraper
	lookersScraper *LookersScraper
}

func New() *Scraper {
	return &Scraper{
		bonhamsScraper: NewBonhamsScraper(),
		lookersScraper: NewLookersScraper(),
	}
}

// GetBonhamsListings gets Bonhams car listings directly
func (s *Scraper) GetBonhamsListings(maxListings int) ([]*models.BonhamsCar, error) {
	fmt.Println("Fetching Bonhams data directly...")
	return s.bonhamsScraper.ScrapeCarListings(maxListings)
}

// GetCarListings gets car listings from Bonhams (legacy method - use GetEnhancedListings instead)
func (s *Scraper) GetCarListings(maxListings int) ([]*models.Car, error) {
	fmt.Println("Fetching data from Bonhams Car Auctions...")
	bonhamsCars, err := s.bonhamsScraper.ScrapeCarListings(maxListings)
	if err != nil {
		return nil, err
	}

	// Convert BonhamsCar to standard Car models
	var cars []*models.Car
	for _, bonhamsCar := range bonhamsCars {
		cars = append(cars, bonhamsCar.ToStandardCar())
	}
	return cars, nil
}

// GetLookersListings gets Lookers car listings for Easy mode
func (s *Scraper) GetLookersListings() ([]*models.LookersCar, error) {
	fmt.Println("Fetching Lookers data for Easy mode...")
	return s.lookersScraper.ScrapeCarListings()
}

// GetEnhancedListings gets enhanced car listings based on difficulty
func (s *Scraper) GetEnhancedListings(difficulty string) ([]*models.EnhancedCar, error) {
	if difficulty == "easy" {
		fmt.Println("Fetching Easy mode data from Lookers...")
		lookersCars, err := s.GetLookersListings()
		if err != nil {
			return nil, err
		}

		// Convert LookersCar to EnhancedCar models
		var enhancedCars []*models.EnhancedCar
		for _, lookersCar := range lookersCars {
			enhancedCars = append(enhancedCars, lookersCar.ToEnhancedCar())
		}
		return enhancedCars, nil
	} else {
		// Default to hard mode (Bonhams)
		fmt.Println("Fetching Hard mode data from Bonhams...")
		bonhamsCars, err := s.GetBonhamsListings(50) // Get more for variety
		if err != nil {
			return nil, err
		}

		// Convert BonhamsCar to EnhancedCar models
		var enhancedCars []*models.EnhancedCar
		for _, bonhamsCar := range bonhamsCars {
			enhancedCars = append(enhancedCars, bonhamsCar.ToEnhancedCar())
		}
		return enhancedCars, nil
	}
}

// Close closes any open browser connections
func (s *Scraper) Close() {
	if s.bonhamsScraper != nil {
		s.bonhamsScraper.Close()
	}
	if s.lookersScraper != nil {
		s.lookersScraper.Close()
	}
}
