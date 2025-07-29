package scraper

import (
	"fmt"

	"autotraderguesser/internal/models"
)

type Scraper struct {
	motorsScraper *MotorsUKScraper
}

func New() *Scraper {
	return &Scraper{
		motorsScraper: NewMotorsUKScraper(),
	}
}

// GetCarListings gets car listings from Motors.co.uk
func (s *Scraper) GetCarListings(maxListings int) ([]*models.Car, error) {
	s.motorsScraper.Enable()
	fmt.Println("Fetching real Motors.co.uk data...")
	return s.motorsScraper.ScrapeCarListings(maxListings)
}

// ScrapeAutotrader wraps the Motors scraper for backward compatibility
func (s *Scraper) ScrapeAutotrader() ([]*models.Car, error) {
	return s.GetCarListings(20)
}

// Close closes any open browser connections
func (s *Scraper) Close() {
	// The MotorsUKScraper manages its own browser lifecycle
}