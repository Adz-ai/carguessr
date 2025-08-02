package scraper

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"autotraderguesser/internal/models"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
)

// LookersScraper handles scraping from Lookers.co.uk
type LookersScraper struct {
	browser *rod.Browser
}

// NewLookersScraper creates a new Lookers scraper
func NewLookersScraper() *LookersScraper {
	return &LookersScraper{}
}

// Close closes the browser connection
func (s *LookersScraper) Close() {
	if s.browser != nil {
		s.browser.MustClose()
		s.browser = nil
	}
}

// initBrowser initializes the browser if not already initialized
func (s *LookersScraper) initBrowser() error {
	if s.browser != nil {
		return nil // Already initialized
	}

	path, _ := launcher.LookPath()
	u := launcher.New().
		Bin(path).
		Headless(true).
		MustLaunch()

	s.browser = rod.New().
		ControlURL(u).
		MustConnect()

	return nil
}

type LinkConfig struct {
	URL          string `json:"url"`
	SortOrder    string `json:"sortOrder"`
	CarsToScrape int    `json:"carsToScrape"`
}

type BodyTypeConfig struct {
	BodyType string       `json:"bodyType"`
	Links    []LinkConfig `json:"links"`
}

type CarJob struct {
	Car   models.LookersCar
	Index int
}

type CarResult struct {
	Car   models.LookersCar
	Index int
	Error error
}

// ScrapeCarListings scrapes car listings from Lookers.co.uk (consistent interface with Bonhams)
func (s *LookersScraper) ScrapeCarListings() ([]*models.LookersCar, error) {
	log.Println("Starting Lookers.co.uk scraper (Easy mode)...")

	// Initialize browser if needed
	if err := s.initBrowser(); err != nil {
		return nil, fmt.Errorf("failed to initialize browser: %v", err)
	}

	// Read configuration from JSON file
	configData, err := os.ReadFile("data/lookers-links.json")
	if err != nil {
		return nil, fmt.Errorf("error reading lookers-links.json: %v", err)
	}

	var bodyTypeConfigs []BodyTypeConfig
	if err := json.Unmarshal(configData, &bodyTypeConfigs); err != nil {
		return nil, fmt.Errorf("error parsing lookers-links.json: %v", err)
	}

	// Use the struct's browser (already initialized)

	// Scrape from all body types and URLs
	var allCars []*models.LookersCar
	totalExpected := 0

	// Calculate total expected cars
	for _, bodyTypeConfig := range bodyTypeConfigs {
		for _, linkConfig := range bodyTypeConfig.Links {
			totalExpected += linkConfig.CarsToScrape
		}
	}

	log.Printf("📊 Starting scrape for %d body types with %d total expected cars", len(bodyTypeConfigs), totalExpected)

	for _, bodyTypeConfig := range bodyTypeConfigs {
		log.Printf("\n🚗 Processing body type: %s", bodyTypeConfig.BodyType)

		for _, linkConfig := range bodyTypeConfig.Links {
			log.Printf("  📈 Scraping %d cars with %s sorting", linkConfig.CarsToScrape, linkConfig.SortOrder)

			cars, err := scrapeLookersURL(s.browser, linkConfig.URL, linkConfig.CarsToScrape, bodyTypeConfig.BodyType, linkConfig.SortOrder)
			if err != nil {
				log.Printf("❌ Error scraping %s %s: %v", bodyTypeConfig.BodyType, linkConfig.SortOrder, err)
				continue
			}

			allCars = append(allCars, cars...)
			log.Printf("  ✅ Added %d cars from %s %s (Total: %d/%d)", len(cars), bodyTypeConfig.BodyType, linkConfig.SortOrder, len(allCars), totalExpected)
		}
	}

	log.Printf("\n🎉 Successfully scraped %d cars from %d body types", len(allCars), len(bodyTypeConfigs))

	// Deduplicate cars based on ID (URL-based)
	log.Printf("🔍 Deduplicating cars...")
	uniqueCars := deduplicateCars(allCars)
	duplicatesRemoved := len(allCars) - len(uniqueCars)
	if duplicatesRemoved > 0 {
		log.Printf("✅ Removed %d duplicate cars, %d unique cars remaining", duplicatesRemoved, len(uniqueCars))
	} else {
		log.Printf("✅ No duplicates found, %d unique cars", len(uniqueCars))
	}

	// Print summary by body type
	bodyTypeSummary := make(map[string]int)
	for _, car := range uniqueCars {
		bodyTypeSummary[car.BodyType]++
	}

	log.Println("\n📊 Cars by body type (after deduplication):")
	for bodyType, count := range bodyTypeSummary {
		log.Printf("  %s: %d cars", bodyType, count)
	}

	return uniqueCars, nil
}

// ScrapeLookersListings is a backward-compatible function that creates a temporary scraper
// Deprecated: Use LookersScraper.ScrapeCarListings() instead for better performance
func ScrapeLookersListings() ([]*models.LookersCar, error) {
	scraper := NewLookersScraper()
	defer scraper.Close()
	return scraper.ScrapeCarListings()
}

func scrapeLookersURL(browser *rod.Browser, url string, maxCars int, bodyType string, sortOrder string) ([]*models.LookersCar, error) {
	log.Printf("Fetching listings from: %s", url)

	// Create stealth page
	page := stealth.MustPage(browser)
	defer page.MustClose()

	// Navigate to Lookers Used Cars
	page.MustNavigate(url)
	page.MustWaitLoad()

	// Wait for content to load
	time.Sleep(3 * time.Second)

	// Handle cookie consent if present
	handleCookieConsent(page)

	// Load more cars by clicking "Load More" button until we have enough listings
	loadMoreCars(page, maxCars)

	// Find car listings - use the correct Lookers selector
	var carElements []*rod.Element
	selectors := []string{
		".vehicle-result",
		".vehicle-result.vehicle-result--new-search",
		"div[class*='vehicle-result']",
	}

	for _, selector := range selectors {
		elements, err := page.Elements(selector)
		if err == nil && len(elements) > 0 {
			log.Printf("Found %d elements with selector: %s", len(elements), selector)
			carElements = elements
			break
		}
	}

	if len(carElements) == 0 {
		return []*models.LookersCar{}, fmt.Errorf("no car listings found on page")
	}

	// Extract basic car info from all elements first (fast)
	var carJobs []CarJob
	for i, element := range carElements {
		if i >= maxCars {
			break
		}

		car := models.LookersCar{
			BodyType:        bodyType,
			SortOrder:       sortOrder,
			Characteristics: make(map[string]string),
			Images:          []string{},
		}

		// Extract basic info from the listing card
		extractBasicInfo(element, &car)

		if car.Title != "" && car.OriginalURL != "" {
			// Generate ID from URL
			parts := strings.Split(car.OriginalURL, "/")
			if len(parts) > 0 {
				car.ID = parts[len(parts)-1]
			}
			carJobs = append(carJobs, CarJob{Car: car, Index: i})
		}
	}

	log.Printf("Found %d valid cars to process concurrently", len(carJobs))

	// Process cars concurrently with worker pool
	return processCarsConc(browser, carJobs)
}

func handleCookieConsent(page *rod.Page) {
	log.Println("Handling cookie consent...")
	cookieSelectors := []string{
		"button[data-testid='cookie-accept']",
		"button.cookie-accept",
		"button#cookie-accept",
		"button[id*='accept']",
		"button[class*='accept']",
		"button[onclick*='accept']",
		".cookie-banner button",
		".cookie-consent button",
		"#cookie-consent button",
		"button[aria-label*='Accept']",
	}

	cookieHandled := false
	for _, selector := range cookieSelectors {
		cookieButton, err := page.Timeout(3 * time.Second).Element(selector)
		if err == nil && cookieButton != nil {
			log.Printf("Found cookie button with selector: %s", selector)
			cookieButton.MustClick()
			log.Println("Cookie consent accepted")
			time.Sleep(2 * time.Second)
			cookieHandled = true
			break
		}
	}

	if !cookieHandled {
		log.Println("No cookie consent button found, continuing...")
	}
}

func loadMoreCars(page *rod.Page, maxCars int) {
	log.Println("Loading more car listings...")
	loadMoreAttempts := 0
	maxLoadMoreAttempts := 20 // Limit to prevent infinite loops

	for loadMoreAttempts < maxLoadMoreAttempts {
		// Check current number of car elements
		currentElements, _ := page.Elements(".vehicle-result")
		log.Printf("Currently loaded: %d car elements", len(currentElements))

		if len(currentElements) >= maxCars*2 { // Load extra to account for filtering
			log.Printf("Loaded enough cars (%d), stopping load more", len(currentElements))
			break
		}

		// Look for Load More button
		loadMoreSelectors := []string{
			"button[data-testid='load-more']",
			".load-more-button",
			"button.btn-load-more",
		}

		loadMoreFound := false
		for _, selector := range loadMoreSelectors {
			loadMoreBtn, err := page.Timeout(3 * time.Second).Element(selector)
			if err == nil && loadMoreBtn != nil {
				log.Printf("Found Load More button with selector: %s", selector)

				// Scroll to the button first
				loadMoreBtn.MustScrollIntoView()
				time.Sleep(1 * time.Second)

				// Click the button
				loadMoreBtn.MustClick()
				log.Println("Clicked Load More button")
				loadMoreFound = true

				// Wait for new content to load
				time.Sleep(3 * time.Second)
				break
			}
		}

		if !loadMoreFound {
			log.Println("No Load More button found, continuing...")
			break
		}

		loadMoreAttempts++
	}
}

func extractBasicInfo(element *rod.Element, car *models.LookersCar) {
	// Extract title from vehicle-result structure
	titleEl, err := element.Element(".vehicle-result__model")
	if err == nil && titleEl != nil {
		car.Title = strings.TrimSpace(titleEl.MustText())

		// Parse make and model from title
		parseMakeModel(car)
	}

	// Extract additional details
	detailsEl, err := element.Element(".vehicle-result__details")
	if err == nil && detailsEl != nil {
		details := strings.TrimSpace(detailsEl.MustText())
		if car.Title != "" && details != "" {
			car.Title = car.Title + " - " + details
		}

		// Extract year from details (usually first number in details)
		re := regexp.MustCompile(`(\d{4})`)
		if match := re.FindString(details); match != "" {
			if year, err := strconv.Atoi(match); err == nil {
				car.Year = year
			}
		}
	}

	// Extract price
	priceEl, err := element.Element(".vehicle-result__price")
	if err == nil && priceEl != nil {
		priceText := strings.TrimSpace(priceEl.MustText())
		car.Price = parsePrice(priceText)
	}

	// Extract location (dealership)
	locationEl, err := element.Element(".vehicle-result__location span")
	if err == nil && locationEl != nil {
		car.Location = strings.TrimSpace(locationEl.MustText())
	}

	// Extract URL - look for the "View" button link
	linkEl, err := element.Element("a.button")
	if err == nil && linkEl != nil {
		href := linkEl.MustAttribute("href")
		if href != nil {
			if strings.HasPrefix(*href, "/") {
				car.OriginalURL = "https://www.lookers.co.uk" + *href
			} else {
				car.OriginalURL = *href
			}
		}
	}
}

func parseMakeModel(car *models.LookersCar) {
	// Extract make from title (usually first word in uppercase)
	words := strings.Fields(car.Title)
	if len(words) > 0 {
		car.Make = strings.ToUpper(words[0])

		// Model is usually the second word
		if len(words) > 1 {
			car.Model = words[1]
		}
	}
}

func parsePrice(priceStr string) float64 {
	// Remove currency symbol and commas
	priceStr = strings.ReplaceAll(priceStr, "£", "")
	priceStr = strings.ReplaceAll(priceStr, ",", "")
	priceStr = strings.TrimSpace(priceStr)

	// Extract numeric value
	re := regexp.MustCompile(`\d+`)
	matches := re.FindString(priceStr)

	price, _ := strconv.ParseFloat(matches, 64)
	return price
}

func processCarsConc(browser *rod.Browser, carJobs []CarJob) ([]*models.LookersCar, error) {
	const numWorkers = 5

	// Create channels
	jobChan := make(chan CarJob, len(carJobs))
	resultChan := make(chan CarResult, len(carJobs))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go carWorker(browser, jobChan, resultChan, &wg)
	}

	// Send jobs
	for _, job := range carJobs {
		jobChan <- job
	}
	close(jobChan)

	// Wait for workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var validCars []*models.LookersCar
	processedCount := 0
	totalJobs := len(carJobs)

	for result := range resultChan {
		processedCount++

		if result.Error != nil {
			log.Printf("❌ Error processing car %d: %v", result.Index+1, result.Error)
			continue
		}

		// Filter out cars with insufficient images (less than 10)
		if len(result.Car.Images) < 10 {
			log.Printf("⏭️  Skipping %s - only %d images (need minimum 10)", result.Car.Title, len(result.Car.Images))
			continue
		}

		log.Printf("✅ Car approved (%d/%d): %s (%d images)", len(validCars)+1, processedCount, result.Car.Title, len(result.Car.Images))
		validCars = append(validCars, &result.Car)
	}

	log.Printf("🎉 Concurrent processing complete: %d approved cars from %d processed", len(validCars), totalJobs)
	return validCars, nil
}

func carWorker(browser *rod.Browser, jobChan <-chan CarJob, resultChan chan<- CarResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobChan {
		car := job.Car
		result := CarResult{Index: job.Index}

		// Get detailed information from the car's detail page
		if err := scrapeCarDetails(browser, &car); err != nil {
			result.Error = err
		} else {
			result.Car = car
		}

		resultChan <- result
	}
}

func scrapeCarDetails(browser *rod.Browser, car *models.LookersCar) error {
	log.Printf("Fetching details from: %s", car.OriginalURL)

	// Create new page for car details
	detailPage := stealth.MustPage(browser)
	defer detailPage.MustClose()

	detailPage = detailPage.Timeout(15 * time.Second)

	err := detailPage.Navigate(car.OriginalURL)
	if err != nil {
		return fmt.Errorf("failed to navigate: %v", err)
	}

	err = detailPage.WaitLoad()
	if err != nil {
		return fmt.Errorf("failed to load: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Extract characteristics from used-specs__data-col
	specElements, err := detailPage.Elements(".used-specs__data-col .used-specs__icon-data-container")
	if err == nil {
		for _, element := range specElements {
			iconUse, err := element.Element("use")
			dataSpan, err2 := element.Element(".used-specs__vehicle-data")

			if err == nil && err2 == nil && iconUse != nil && dataSpan != nil {
				iconHref := iconUse.MustAttribute("xlink:href")
				dataValue := strings.TrimSpace(dataSpan.MustText())

				if iconHref != nil && *iconHref != "" && dataValue != "" {
					// Extract the icon type from the href
					iconType := extractIconType(*iconHref)
					if iconType != "" {
						car.Characteristics[iconType] = dataValue
					}
				}
			}
		}
	}

	// Extract images from carousel
	carouselSelectors := []string{
		".flickity-slider .carousel__cell .carousel__image",
		".image-carousel img",
		".vehicle-images img",
		".gallery img",
		"[data-testid*='image']",
	}

	for _, selector := range carouselSelectors {
		imageElements, err := detailPage.Elements(selector)
		if err == nil && len(imageElements) > 0 {
			log.Printf("Found %d images with selector: %s", len(imageElements), selector)

			for _, element := range imageElements {
				// Try different ways to get image URLs
				if style := element.MustAttribute("style"); style != nil && *style != "" {
					if imageURL := extractImageFromStyle(*style); imageURL != "" {
						car.Images = append(car.Images, imageURL)
					}
				}

				if lazyLoad := element.MustAttribute("data-flickity-bg-lazyload"); lazyLoad != nil && *lazyLoad != "" {
					car.Images = append(car.Images, *lazyLoad)
				}

				if src := element.MustAttribute("src"); src != nil && *src != "" {
					car.Images = append(car.Images, *src)
				}

				if dataSrc := element.MustAttribute("data-src"); dataSrc != nil && *dataSrc != "" {
					car.Images = append(car.Images, *dataSrc)
				}
			}
			break
		}
	}

	// Remove duplicate images
	car.Images = removeDuplicateStrings(car.Images)

	// Only keep high-quality images (not thumbnails)
	var highQualityImages []string
	for _, img := range car.Images {
		if !strings.Contains(img, "120x90") && !strings.Contains(img, "missing.png") {
			highQualityImages = append(highQualityImages, img)
		}
	}
	car.Images = highQualityImages

	log.Printf("Extracted %d characteristics and %d images for %s", len(car.Characteristics), len(car.Images), car.Title)

	return nil
}

func extractIconType(iconHref string) string {
	// Extract icon type from href like "/assets/images/iconography.svg#VehicleOdometer"
	parts := strings.Split(iconHref, "#")
	if len(parts) != 2 {
		return ""
	}

	iconName := parts[1]
	// Convert camelCase to readable format
	switch iconName {
	case "VehicleOdometer":
		return "Mileage"
	case "Calendar":
		return "Year"
	case "VehicleFuelType":
		return "Fuel Type"
	case "VehicleFuelConsumption":
		return "MPG"
	case "VehicleEngineSize":
		return "Engine Size"
	case "VehicleOwner":
		return "Owners"
	case "VehicleEngineType":
		return "Transmission"
	case "VehicleDoors":
		return "Doors"
	case "VehicleRegistration":
		return "Registration"
	case "Droplet":
		return "Color"
	default:
		return iconName
	}
}

func extractImageFromStyle(style string) string {
	// Extract URL from style="background-image: url("https://...")"
	re := regexp.MustCompile(`url\("([^"]+)"\)`)
	matches := re.FindStringSubmatch(style)
	if len(matches) > 1 {
		return matches[1]
	}

	// Try without quotes
	re = regexp.MustCompile(`url\(([^)]+)\)`)
	matches = re.FindStringSubmatch(style)
	if len(matches) > 1 {
		return strings.Trim(matches[1], `"'`)
	}

	return ""
}

func removeDuplicateStrings(slice []string) []string {
	keys := make(map[string]bool)
	result := []string{}

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// deduplicateCars removes duplicate cars based on their ID (which is generated from URL)
func deduplicateCars(cars []*models.LookersCar) []*models.LookersCar {
	seen := make(map[string]bool)
	var uniqueCars []*models.LookersCar

	for _, car := range cars {
		// Use car ID as the deduplication key (ID is generated from URL)
		if car.ID != "" && !seen[car.ID] {
			seen[car.ID] = true
			uniqueCars = append(uniqueCars, car)
		} else if car.ID != "" {
			// Log duplicate found for debugging
			log.Printf("🔄 Duplicate car found: %s (ID: %s)", car.Title, car.ID)
		}
	}

	return uniqueCars
}
