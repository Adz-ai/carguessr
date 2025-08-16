package scraper

import (
	"fmt"
	"runtime"

	"autotraderguesser/internal/models"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
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

// CreateBrowser creates a standardized browser instance that works on both Mac and Linux
// This provides a consistent configuration for all scrapers
func CreateBrowser() (*rod.Browser, error) {
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	l := launcher.New().
		Headless(true).
		Set("user-agent", userAgent).
		Set("disable-blink-features", "AutomationControlled").
		Set("disable-dev-shm-usage") // Use /tmp instead of /dev/shm

	// Add Linux-specific flags to handle sandbox issues
	if runtime.GOOS == "linux" {
		l = l.
			Set("no-sandbox").             // Disable sandbox for Linux compatibility
			Set("disable-setuid-sandbox"). // Additional sandbox disable for older kernels
			Set("disable-gpu").            // Disable GPU hardware acceleration on Linux
			Set("single-process")          // Run Chrome in single process mode (helps with containers)
	}

	// Add stealth flags for better anti-detection
	l = l.
		Set("disable-web-security").
		Set("allow-running-insecure-content").
		Set("disable-features", "IsolateOrigins,site-per-process").
		Set("flag-switches-begin").
		Set("flag-switches-end")

	url, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %v", err)
	}

	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %v", err)
	}

	return browser, nil
}
