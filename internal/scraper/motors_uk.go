package scraper

import (
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"

	"autotraderguesser/internal/models"
)

// MotorsUKScraper handles scraping from Motors.co.uk
type MotorsUKScraper struct {
	browser *rod.Browser
	enabled bool
}

// NewMotorsUKScraper creates a new Motors UK scraper
func NewMotorsUKScraper() *MotorsUKScraper {
	return &MotorsUKScraper{
		enabled: true, // Enabled by default
	}
}

// Enable enables the scraper
func (s *MotorsUKScraper) Enable() {
	s.enabled = true
}

// ScrapeCarListings scrapes car listings from Motors.co.uk with variety by make
func (s *MotorsUKScraper) ScrapeCarListings(maxListings int) ([]*models.Car, error) {
	if !s.enabled {
		return nil, fmt.Errorf("motors UK scraping is disabled")
	}

	err := s.initBrowser()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize browser: %v", err)
	}
	defer s.closeBrowser()

	// Calculate how many listings per make (aim for 5 per make)
	listingsPerMake := 5
	targetMakes := maxListings / listingsPerMake
	if targetMakes < 3 {
		targetMakes = 3 // Minimum 3 different makes
		listingsPerMake = maxListings / targetMakes
	}

	fmt.Printf("Targeting %d makes with %d listings each (total ~%d)\n", targetMakes, listingsPerMake, targetMakes*listingsPerMake)

	return s.scrapeWithVariety(targetMakes, listingsPerMake)
}

// scrapeWithVariety scrapes listings ensuring variety across different makes
func (s *MotorsUKScraper) scrapeWithVariety(targetMakes, listingsPerMake int) ([]*models.Car, error) {
	var allCars []*models.Car
	makeCount := make(map[string]int)

	// Different search locations for variety
	searchLocations := []string{
		"?postcode=SW1A1AA", // London
		"?postcode=M11AA",   // Manchester
		"?postcode=B11AA",   // Birmingham
		"?postcode=LE11AA",  // Leicester
		"?postcode=LS11AA",  // Leeds
		"?postcode=BN11AA",  // Brighton
		"?postcode=CF11AA",  // Cardiff
		"?postcode=EH11AA",  // Edinburgh
	}

	attempts := 0
	maxAttempts := len(searchLocations) * 2 // Try each location twice if needed

	for len(makeCount) < targetMakes && attempts < maxAttempts {
		// Pick a search location
		location := searchLocations[attempts%len(searchLocations)]
		attempts++

		fmt.Printf("\\n🔍 Searching location %s (attempt %d, found %d makes so far)\\n", location, attempts, len(makeCount))

		cars, err := s.scrapeSingleLocation(location, 30) // Get more cars to filter from
		if err != nil {
			fmt.Printf("⚠️  Location %s failed: %v\\n", location, err)
			continue
		}

		// Process cars and maintain variety
		for _, car := range cars {
			if car.Make == "" || car.Price <= 0 {
				continue
			}

			// Check if we need more of this make
			if makeCount[car.Make] < listingsPerMake {
				allCars = append(allCars, car)
				makeCount[car.Make]++
				fmt.Printf("  ✅ Added %s %s (£%.0f) - %s count: %d/%d\\n",
					car.Make, car.Model, car.Price, car.Make, makeCount[car.Make], listingsPerMake)

				// Stop if we have enough cars total
				if len(allCars) >= targetMakes*listingsPerMake {
					break
				}
			}
		}

		// Print current variety status
		fmt.Printf("📊 Current variety: ")
		for make, count := range makeCount {
			fmt.Printf("%s(%d) ", make, count)
		}
		fmt.Println()

		// Break if we have good variety
		if len(makeCount) >= targetMakes {
			fmt.Printf("✅ Achieved target variety: %d different makes\\n", len(makeCount))
			break
		}
	}

	fmt.Printf("\\n🏁 Final collection: %d cars across %d makes\\n", len(allCars), len(makeCount))
	return allCars, nil
}

// scrapeSingleLocation scrapes a single Motors location
func (s *MotorsUKScraper) scrapeSingleLocation(searchParam string, maxListings int) ([]*models.Car, error) {
	page := s.browser.MustPage()
	defer page.Close()

	searchURL := "https://www.motors.co.uk/search/car/" + searchParam
	page.MustNavigate(searchURL).MustWaitLoad()

	// Handle postcode requirement if present
	postcodeInput, err := page.ElementX("//input[@placeholder*='postcode' or @name*='postcode']")
	if err == nil && postcodeInput != nil {
		postcodeInput.MustInput("SW1A1AA")

		submitButton, err := page.ElementX("//button[contains(text(), 'Search') or contains(text(), 'Find')]")
		if err == nil && submitButton != nil {
			submitButton.MustClick()
			page.MustWaitLoad()
		}
	}

	// Wait for content to load
	time.Sleep(3 * time.Second)

	// Look for car listing elements
	listingSelectors := []string{
		".result-card",
		".result-item",
		".search-result",
		".vehicle-card",
		".car-card",
		"[data-testid*='listing']",
		"[data-testid*='result']",
		".listing-item",
		".car-listing",
		".vehicle-listing",
		"article",
		".card",
	}

	var cars []*models.Car

	for _, selector := range listingSelectors {
		elements, err := page.Elements(selector)
		if err != nil || len(elements) == 0 {
			continue
		}

		// Process listings from this location
		for i, element := range elements {
			if i >= maxListings {
				break
			}

			car := s.parseListingElement(element)
			if car != nil && car.Make != "" && car.Price > 0 {
				// Try to enhance with detail page data
				if car.OriginalURL != "" {
					err := s.enhanceCarWithDetailPage(car)
					if err != nil {
						fmt.Printf("  ⚠️  Could not enhance %s %s: %v\\n", car.Make, car.Model, err)
					}
				}
				cars = append(cars, car)
			}
		}
		break // Found listings, stop trying selectors
	}

	return cars, nil
}

// parseListingElement extracts car data from a listing element
func (s *MotorsUKScraper) parseListingElement(element *rod.Element) *models.Car {
	car := &models.Car{
		ID: fmt.Sprintf("motors-%d", time.Now().UnixNano()),
	}

	// Try to get the parent listing card
	listingCard := element
	if element.MustHTML() != "" {
		// Try to find the parent result card
		parent, err := element.Element(".result-card")
		if err == nil && parent != nil {
			listingCard = parent
		}
	}

	// Get all text content
	text, err := listingCard.Text()
	if err != nil {
		return nil
	}

	fmt.Printf("Listing text: %s\n", truncateString(text, 300))

	// Extract price
	priceRegex := regexp.MustCompile(`£([0-9,]+)`)
	if matches := priceRegex.FindStringSubmatch(text); len(matches) > 1 {
		priceStr := strings.ReplaceAll(matches[1], ",", "")
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			car.Price = price
		}
	}

	// Try to find make/model from title or heading elements
	titleElements := []string{
		".result-card__title",
		".vehicle-title",
		".car-title",
		"h3",
		"h2",
		".title",
	}

	for _, selector := range titleElements {
		titleEl, err := listingCard.Element(selector)
		if err == nil && titleEl != nil {
			titleText, err := titleEl.Text()
			if err == nil && titleText != "" {
				// Parse make/model from title (handles "Make, Model" format)
				titleText = strings.TrimSpace(titleText)
				if strings.Contains(titleText, ",") {
					parts := strings.Split(titleText, ",")
					if len(parts) >= 2 {
						car.Make = strings.TrimSpace(parts[0])
						car.Model = strings.TrimSpace(strings.Join(parts[1:], ","))
						break
					}
				} else {
					// Fallback to space-separated parsing
					words := strings.Fields(titleText)
					if len(words) >= 2 {
						car.Make = words[0]
						car.Model = strings.Join(words[1:], " ")
						break
					}
				}
			}
		}
	}

	// If no make/model found from title, try to parse from main text
	if car.Make == "" {
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.Contains(line, "£") ||
				strings.Contains(line, "Finance") || strings.Contains(line, "pm") ||
				strings.Contains(line, "Miles") || strings.Contains(line, "Diesel") ||
				strings.Contains(line, "Petrol") || strings.Contains(line, "Auto") ||
				strings.Contains(line, "Manual") {
				continue
			}

			// Look for make/model pattern
			words := strings.Fields(line)
			if len(words) >= 2 && len(words[0]) > 2 {
				car.Make = words[0]
				car.Model = strings.Join(words[1:], " ")
				break
			}
		}
	}

	// Extract mileage
	mileageRegex := regexp.MustCompile(`([0-9]+\.?[0-9]*)\s*k?\s*Miles?`)
	if matches := mileageRegex.FindStringSubmatch(text); len(matches) > 1 {
		mileageStr := matches[1]
		if strings.Contains(text, "k") && !strings.Contains(mileageStr, ".") {
			// Convert k to actual number
			if mileage, err := strconv.ParseFloat(mileageStr, 64); err == nil {
				car.Mileage = int(mileage * 1000)
			}
		} else if strings.Contains(mileageStr, ".") {
			// Handle decimal k format (e.g., "19.5k")
			if mileage, err := strconv.ParseFloat(mileageStr, 64); err == nil {
				car.Mileage = int(mileage * 1000)
			}
		} else {
			if mileage, err := strconv.Atoi(mileageStr); err == nil {
				car.Mileage = mileage
			}
		}
	}

	// Extract fuel type
	if strings.Contains(text, "Diesel") {
		car.FuelType = "Diesel"
	} else if strings.Contains(text, "Petrol") {
		car.FuelType = "Petrol"
	} else if strings.Contains(text, "Electric") {
		car.FuelType = "Electric"
	} else if strings.Contains(text, "Hybrid") {
		car.FuelType = "Hybrid"
	}

	// Extract transmission
	if strings.Contains(text, "Auto") {
		car.Gearbox = "Automatic"
	} else if strings.Contains(text, "Manual") {
		car.Gearbox = "Manual"
	}

	// Extract year with multiple patterns for robustness
	yearPatterns := []string{
		`(20\d{2})\s*\(\d+\)\s*-`, // "2018 (68) -"
		`(20\d{2})\s*\(\d+\)`,     // "2018 (68)"
		`(20\d{2})\s*-`,           // "2018 -"
		`\b(20\d{2})\b`,           // Any standalone year 2000-2099
		`^(20\d{2})`,              // Year at start of text
	}

	for _, pattern := range yearPatterns {
		yearRegex := regexp.MustCompile(pattern)
		if matches := yearRegex.FindStringSubmatch(text); len(matches) > 1 {
			if year, err := strconv.Atoi(matches[1]); err == nil && year >= 2000 && year <= 2025 {
				car.Year = year
				fmt.Printf("  ✓ Extracted year: %d (pattern: %s)\n", year, pattern)
				break
			}
		}
	}

	// Extract engine size (e.g., "1.4L", "2L", "1.5L", "3L")
	engineRegex := regexp.MustCompile(`(\d+\.?\d*)\s*L`)
	if matches := engineRegex.FindStringSubmatch(text); len(matches) > 1 {
		car.Engine = matches[1] + "L"
	}

	// Extract body type
	bodyTypes := []string{"SUV", "Hatchback", "Saloon", "Estate", "Coupe", "Convertible", "MPV"}
	for _, bodyType := range bodyTypes {
		if strings.Contains(text, bodyType) {
			car.BodyType = bodyType
			break
		}
	}

	// Extract images - try to get multiple if available
	imageElements, err := listingCard.Elements("img")
	if err == nil {
		var images []string
		for _, imgEl := range imageElements {
			src, err := imgEl.Attribute("src")
			if err == nil && src != nil && *src != "" &&
				!strings.Contains(*src, "placeholder") &&
				(strings.Contains(*src, "cdn.images") ||
					strings.Contains(*src, "autoexpress") ||
					strings.Contains(*src, "autosonshow") ||
					strings.Contains(*src, "motors")) {
				images = append(images, *src)
			}
		}
		if len(images) > 0 {
			car.Images = images
		}
	}

	// Extract additional details from search result text
	// This is now handled in the main parsing above

	// Extract original URL from the listing link
	linkElement, err := listingCard.Element("a.result-card__link")
	if err == nil && linkElement != nil {
		href, err := linkElement.Attribute("href")
		if err == nil && href != nil && *href != "" {
			if strings.HasPrefix(*href, "/") {
				car.OriginalURL = "https://www.motors.co.uk" + *href
			} else {
				car.OriginalURL = *href
			}
		}
	}

	return car
}

// scrapeDetailPage navigates to a Motors detail page and extracts comprehensive data
func (s *MotorsUKScraper) scrapeDetailPage(page *rod.Page, detailURL string) (*models.Car, error) {
	// Navigate to detail page
	page.MustNavigate(detailURL).MustWaitLoad()

	// Wait for content to load
	time.Sleep(time.Duration(1+rand.Intn(2)) * time.Second)

	car := &models.Car{
		ID:          fmt.Sprintf("motors-%d", time.Now().UnixNano()),
		OriginalURL: detailURL,
	}

	// Extract title (make, model, year)
	titleSelectors := []string{
		"h1",
		".vehicle-title",
		".advert-title",
	}

	for _, selector := range titleSelectors {
		titleEl, err := page.Element(selector)
		if err == nil && titleEl != nil {
			titleText, err := titleEl.Text()
			if err == nil && titleText != "" {
				// Parse title like "2018 (18) - 1.4 TSI SE 5dr Petrol Hatchback"
				s.parseTitle(titleText, car)
				break
			}
		}
	}

	// Extract price
	priceSelectors := []string{
		".price",
		"[class*='price']",
		"span:contains('£')",
	}

	for _, selector := range priceSelectors {
		priceEl, err := page.Element(selector)
		if err == nil && priceEl != nil {
			priceText, err := priceEl.Text()
			if err == nil && priceText != "" {
				priceRegex := regexp.MustCompile(`£([0-9,]+)`)
				if matches := priceRegex.FindStringSubmatch(priceText); len(matches) > 1 {
					priceStr := strings.ReplaceAll(matches[1], ",", "")
					if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
						car.Price = price
						break
					}
				}
			}
		}
	}

	// Extract all gallery images
	galleryImages, err := s.extractGalleryImages(page)
	if err == nil && len(galleryImages) > 0 {
		car.Images = galleryImages
	}

	// Extract detailed specifications
	err = s.extractDetailedSpecs(page, car)
	if err != nil {
		fmt.Printf("  ⚠️  Warning: Could not extract detailed specs: %v\n", err)
	}

	return car, nil
}

// parseTitle extracts make, model, year from title text
func (s *MotorsUKScraper) parseTitle(title string, car *models.Car) {
	// Handle title format: "2018 (18) - 1.4 TSI SE 5dr Petrol Hatchback"
	title = strings.TrimSpace(title)

	// Extract year
	yearRegex := regexp.MustCompile(`^(20\d{2})\s*\(\d+\)`)
	if matches := yearRegex.FindStringSubmatch(title); len(matches) > 1 {
		if year, err := strconv.Atoi(matches[1]); err == nil {
			car.Year = year
		}
	}

	// For make/model, we'll rely on the search result data since detail page
	// titles are more descriptive but less clear about make/model boundaries
}

// extractGalleryImages extracts all images from the Motors detail page gallery
func (s *MotorsUKScraper) extractGalleryImages(page *rod.Page) ([]string, error) {
	var allImages []string

	// Gallery selectors based on our test
	gallerySelectors := []string{
		".gallery img",
		".image-gallery img",
		".photos img",
		"img[src*='autosonshow.tv']",
		"img[src*='cdn.images']",
		"img[src*='autoexposure']",
	}

	for _, selector := range gallerySelectors {
		elements, err := page.Elements(selector)
		if err == nil {
			for _, el := range elements {
				src, err := el.Attribute("src")
				if err == nil && src != nil && s.isValidCarImage(*src) {
					allImages = append(allImages, *src)
				}
			}
		}
	}

	// Remove duplicates and limit to reasonable number
	seen := make(map[string]bool)
	var uniqueImages []string
	for _, img := range allImages {
		if !seen[img] && len(uniqueImages) < 20 { // Limit to 20 images
			seen[img] = true
			uniqueImages = append(uniqueImages, img)
		}
	}

	return uniqueImages, nil
}

// extractDetailedSpecs extracts detailed specifications from the detail page
func (s *MotorsUKScraper) extractDetailedSpecs(page *rod.Page, car *models.Car) error {
	// Get all page text for parsing
	bodyText, err := page.Eval("() => document.body.innerText")
	if err != nil {
		return err
	}

	text := fmt.Sprintf("%v", bodyText.Value)
	lines := strings.Split(text, "\n")

	// Parse key specifications
	for _, line := range lines {
		line = strings.TrimSpace(line)
		lineLower := strings.ToLower(line)

		// Engine size (e.g., "1.4 litre")
		if strings.Contains(lineLower, "litre") {
			engineRegex := regexp.MustCompile(`(\d+\.?\d*)\s*litre`)
			if matches := engineRegex.FindStringSubmatch(lineLower); len(matches) > 1 {
				car.Engine = matches[1] + "L"
			}
		}

		// Mileage (e.g., "58,075 Miles")
		if strings.Contains(lineLower, "miles") && !strings.Contains(lineLower, "away") {
			mileageRegex := regexp.MustCompile(`([0-9,]+)\s*miles`)
			if matches := mileageRegex.FindStringSubmatch(lineLower); len(matches) > 1 {
				mileageStr := strings.ReplaceAll(matches[1], ",", "")
				if mileage, err := strconv.Atoi(mileageStr); err == nil {
					car.Mileage = mileage
				}
			}
		}

		// Seats (e.g., "5 Seats")
		if strings.Contains(lineLower, "seats") {
			seatsRegex := regexp.MustCompile(`(\d+)\s*seats`)
			if matches := seatsRegex.FindStringSubmatch(lineLower); len(matches) > 1 {
				car.Seats = matches[1]
			}
		}

		// Doors (e.g., "5 Door")
		if strings.Contains(lineLower, "door") && !strings.Contains(lineLower, "outdoor") {
			doorsRegex := regexp.MustCompile(`(\d+)\s*door`)
			if matches := doorsRegex.FindStringSubmatch(lineLower); len(matches) > 1 {
				car.Doors = matches[1]
			}
		}

		// Fuel type
		if car.FuelType == "" {
			if strings.Contains(lineLower, "petrol") && !strings.Contains(lineLower, "diesel") {
				car.FuelType = "Petrol"
			} else if strings.Contains(lineLower, "diesel") {
				car.FuelType = "Diesel"
			} else if strings.Contains(lineLower, "electric") {
				car.FuelType = "Electric"
			} else if strings.Contains(lineLower, "hybrid") {
				car.FuelType = "Hybrid"
			}
		}

		// Transmission
		if car.Gearbox == "" {
			if strings.Contains(lineLower, "manual") && !strings.Contains(lineLower, "automatic") {
				car.Gearbox = "Manual"
			} else if strings.Contains(lineLower, "automatic") || strings.Contains(lineLower, "auto") {
				car.Gearbox = "Automatic"
			}
		}

		// Body type
		if car.BodyType == "" {
			bodyTypes := []string{"hatchback", "saloon", "suv", "estate", "coupe", "convertible"}
			for _, bodyType := range bodyTypes {
				if strings.Contains(lineLower, bodyType) {
					car.BodyType = strings.ToUpper(bodyType[:1]) + bodyType[1:]
					break
				}
			}
		}
	}

	// Future: Extract additional details if needed

	return nil
}

// isValidCarImage checks if the URL is a valid car image
func (s *MotorsUKScraper) isValidCarImage(url string) bool {
	if url == "" || !strings.HasPrefix(url, "http") {
		return false
	}

	// Valid Motors/car image domains
	validDomains := []string{
		"autosonshow.tv",
		"cdn.images",
		"autoexposure",
		"motors.co.uk",
	}

	for _, domain := range validDomains {
		if strings.Contains(url, domain) && !strings.Contains(url, "placeholder") {
			return true
		}
	}

	return false
}

func (s *MotorsUKScraper) initBrowser() error {
	// Configure browser for both local and Docker environments
	l := launcher.New().
		Headless(true).
		Set("disable-blink-features", "AutomationControlled").
		Set("disable-dev-shm-usage").
		Set("no-sandbox").
		Set("disable-gpu").
		Set("disable-extensions").
		Set("disable-background-timer-throttling").
		Set("disable-renderer-backgrounding").
		Set("disable-backgrounding-occluded-windows").
		Set("disable-ipc-flooding-protection").
		Set("window-size", "1920,1080").
		Set("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// Check for Chromium first (common in containers)
	if chromiumPath := findChromiumPath(); chromiumPath != "" {
		fmt.Printf("🔍 Using Chromium at: %s\n", chromiumPath)
		l = l.Bin(chromiumPath)
	}

	// Check if running in Docker and add additional flags
	if isDockerEnvironment() {
		fmt.Println("🐳 Docker environment detected, applying container-specific settings")
		l = l.Set("remote-debugging-port", "9222").
			Set("disable-setuid-sandbox").
			Set("no-first-run").
			Set("disable-default-apps").
			Set("single-process") // Use single process in containers
	}

	url, err := l.Launch()
	if err != nil {
		return err
	}

	s.browser = rod.New().ControlURL(url)
	err = s.browser.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to browser: %v", err)
	}

	fmt.Println("✅ Browser initialized successfully")
	return nil
}

func (s *MotorsUKScraper) closeBrowser() {
	if s.browser != nil {
		s.browser.Close()
	}
}

// enhanceCarWithDetailPage enhances car data by visiting the detail page
func (s *MotorsUKScraper) enhanceCarWithDetailPage(car *models.Car) error {
	// Create a new page for the detail view
	detailPage := s.browser.MustPage()
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("  ⚠️  Detail page panic recovered: %v\n", err)
		}
		detailPage.Close()
	}()

	// Set timeout for the entire operation
	ctx := detailPage.Timeout(15 * time.Second)

	// Navigate with error handling
	err := ctx.Navigate(car.OriginalURL)
	if err != nil {
		return fmt.Errorf("navigation failed: %v", err)
	}

	// Wait for the page to stabilize
	err = ctx.WaitLoad()
	if err != nil {
		return fmt.Errorf("page load failed: %v", err)
	}

	// Additional wait for dynamic content
	time.Sleep(2 * time.Second)

	// Check if we hit bot detection or error page
	title := detailPage.MustInfo().Title
	if strings.Contains(strings.ToLower(title), "not found") ||
		strings.Contains(strings.ToLower(title), "error") {
		return fmt.Errorf("page not available: %s", title)
	}

	fmt.Printf("  ✓ Detail page loaded: %s\n", title)

	// Extract gallery images using existing method
	galleryImages, err := s.extractGalleryImages(detailPage)
	if err == nil && len(galleryImages) > 0 {
		car.Images = galleryImages
		fmt.Printf("  ✓ Found %d gallery images\n", len(galleryImages))
	}

	// Extract detailed specs using existing method
	err = s.extractDetailedSpecs(detailPage, car)
	if err != nil {
		fmt.Printf("  ⚠️  Could not extract all specs: %v\n", err)
	}

	return nil
}

// findChromiumPath looks for Chromium/Chrome binary in common locations
func findChromiumPath() string {
	// Check environment variable first
	if chromeBin := os.Getenv("CHROME_BIN"); chromeBin != "" {
		if _, err := os.Stat(chromeBin); err == nil {
			return chromeBin
		}
	}

	// Common paths for Chrome/Chromium
	paths := []string{
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/snap/bin/chromium",
		"/opt/google/chrome/chrome",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// isDockerEnvironment checks if running inside Docker
func isDockerEnvironment() bool {
	// Check for common Docker environment indicators
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check cgroup for docker
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		return strings.Contains(string(data), "docker") || strings.Contains(string(data), "containerd")
	}

	return false
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
