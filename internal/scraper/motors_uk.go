package scraper

import (
	"fmt"
	"math/rand"
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

// ScrapeCarListings scrapes car listings from Motors.co.uk search results
func (s *MotorsUKScraper) ScrapeCarListings(maxListings int) ([]*models.Car, error) {
	if !s.enabled {
		return nil, fmt.Errorf("motors UK scraping is disabled")
	}

	err := s.initBrowser()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize browser: %v", err)
	}
	defer s.closeBrowser()

	// Navigate to Motors search page
	fmt.Println("Navigating to Motors.co.uk...")
	page := s.browser.MustPage()
	defer page.Close()

	// Use different search parameters for variety 
	searchParams := []string{
		"?postcode=SW1A1AA",
		"?postcode=M11AA", // Manchester
		"?postcode=B11AA", // Birmingham  
		"?postcode=LE11AA", // Leicester
		"?postcode=LS11AA", // Leeds
	}
	
	// Pick a random search location for variety
	selectedParam := searchParams[rand.Intn(len(searchParams))]
	searchURL := "https://www.motors.co.uk/search/car/" + selectedParam
	page.MustNavigate(searchURL).MustWaitLoad()

	fmt.Printf("Page loaded - Title: %s\n", page.MustInfo().Title)

	// Handle postcode requirement if present
	postcodeInput, err := page.ElementX("//input[@placeholder*='postcode' or @name*='postcode']")
	if err == nil && postcodeInput != nil {
		fmt.Println("Found postcode input, filling with SW1A1AA...")
		postcodeInput.MustInput("SW1A1AA")
		
		// Look for submit button
		submitButton, err := page.ElementX("//button[contains(text(), 'Search') or contains(text(), 'Find')]")
		if err == nil && submitButton != nil {
			submitButton.MustClick()
			page.MustWaitLoad()
		}
	}

	// Wait for content to load
	time.Sleep(5 * time.Second)

	// Look for car listing elements on search results - prioritize actual result cards
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
	found := false

	for _, selector := range listingSelectors {
		elements, err := page.Elements(selector)
		if err != nil || len(elements) == 0 {
			continue
		}

		fmt.Printf("Found %d potential listings with selector: %s\n", len(elements), selector)
		found = true

		// Process first few elements to understand structure
		for i, element := range elements {
			if i >= maxListings {
				break
			}

			fmt.Printf("\n=== Analyzing listing %d ===\n", i+1)
			
			// Get element HTML to analyze structure
			html, err := element.HTML()
			if err != nil {
				continue
			}

			fmt.Printf("Element HTML (first 500 chars): %s...\n", truncateString(html, 500))

			// Try to extract basic info
			car := s.parseListingElement(element)
			if car != nil && car.Make != "" {
				cars = append(cars, car)
				fmt.Printf("✅ Parsed: %s %s (£%.0f)\n", car.Make, car.Model, car.Price)
			}
		}
		break
	}

	if !found {
		// Debug: get page content
		bodyText, _ := page.Eval("() => document.body.innerText")
		fmt.Printf("Page content preview: %s...\n", truncateString(fmt.Sprintf("%v", bodyText.Value), 1000))
		return nil, fmt.Errorf("no car listings found on Motors search page")
	}

	fmt.Printf("✅ Motors scraper found %d cars\n", len(cars))
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

	// Extract year from subtitle (e.g., "2018 (68) - 1.6 GDi SE Nav 5dr 2WD")
	yearRegex := regexp.MustCompile(`(20\d{2})\s*\(\d+\)\s*-`)
	if matches := yearRegex.FindStringSubmatch(text); len(matches) > 1 {
		if year, err := strconv.Atoi(matches[1]); err == nil {
			car.Year = year
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

	// Extract images
	imageElements, err := listingCard.Elements("img")
	if err == nil {
		var images []string
		for _, imgEl := range imageElements {
			src, err := imgEl.Attribute("src")
			if err == nil && src != nil && *src != "" && 
			   !strings.Contains(*src, "placeholder") &&
			   (strings.Contains(*src, "cdn.images") || strings.Contains(*src, "autoexpress") || strings.Contains(*src, "motors")) {
				images = append(images, *src)
			}
		}
		if len(images) > 0 {
			car.Images = images
		}
	}

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

func (s *MotorsUKScraper) initBrowser() error {
	// Use headless browser
	l := launcher.New().
		Headless(true).
		Set("disable-blink-features", "AutomationControlled").
		Set("disable-dev-shm-usage").
		Set("no-sandbox").
		Set("disable-gpu").
		Set("disable-extensions").
		Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	
	url, err := l.Launch()
	if err != nil {
		return err
	}
	
	s.browser = rod.New().ControlURL(url)
	err = s.browser.Connect()
	if err != nil {
		return err
	}

	return nil
}

func (s *MotorsUKScraper) closeBrowser() {
	if s.browser != nil {
		s.browser.Close()
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}