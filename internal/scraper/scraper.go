package scraper

import (
	"fmt"

	"autotraderguesser/internal/models"
)

type Scraper struct {
	motorsScraper     *MotorsUKScraper
	collectingCars    *CollectingCarsScraper
	bonhamsScraper    *BonhamsScraper
	useCollectingCars bool
	useBonhams        bool
}

func New() *Scraper {
	return &Scraper{
		motorsScraper:     NewMotorsUKScraper(),
		collectingCars:    NewCollectingCarsScraper(),
		bonhamsScraper:    NewBonhamsScraper(),
		useCollectingCars: false, // Disabled
		useBonhams:        true,  // Use Bonhams as primary scraper
	}
}

// SetCollectingCarsCredentials sets the login credentials for Collecting Cars
func (s *Scraper) SetCollectingCarsCredentials(email, password string) {
	s.collectingCars.SetCredentials(email, password)
}

// GetBonhamsListings gets Bonhams car listings directly
func (s *Scraper) GetBonhamsListings(maxListings int) ([]*models.BonhamsCar, error) {
	if s.useBonhams {
		fmt.Println("Fetching Bonhams data directly...")
		return s.bonhamsScraper.ScrapeCarListings(maxListings)
	}
	return nil, fmt.Errorf("bonhams scraper is not enabled")
}

// GetCarListings gets car listings from the configured source
func (s *Scraper) GetCarListings(maxListings int) ([]*models.Car, error) {
	if s.useBonhams {
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

	if s.useCollectingCars {
		fmt.Println("Fetching data from Collecting Cars...")
		return s.collectingCars.ScrapeCarListings(maxListings)
	}

	s.motorsScraper.Enable()
	fmt.Println("Fetching real Motors.co.uk data...")
	return s.motorsScraper.ScrapeCarListings(maxListings)
}

// ScrapeMotors wraps the Motors scraper for backward compatibility
func (s *Scraper) ScrapeMotors() ([]*models.Car, error) {
	return s.GetCarListings(20)
}

// Close closes any open browser connections
func (s *Scraper) Close() {
	// The MotorsUKScraper manages its own browser lifecycle
}
