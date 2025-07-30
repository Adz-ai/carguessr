package scraper

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"autotraderguesser/internal/models"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// CollectingCarsScraper handles scraping from Collecting Cars
type CollectingCarsScraper struct {
	browser  *rod.Browser
	enabled  bool
	email    string
	password string
}

// NewCollectingCarsScraper creates a new Collecting Cars scraper
func NewCollectingCarsScraper() *CollectingCarsScraper {
	return &CollectingCarsScraper{
		enabled:  true,
		email:    "", // Set these via environment variables or config
		password: "",
	}
}

// Enable enables the scraper
func (s *CollectingCarsScraper) Enable() {
	s.enabled = true
}

// SetCredentials sets the login credentials
func (s *CollectingCarsScraper) SetCredentials(email, password string) {
	s.email = email
	s.password = password
}

// ScrapeCarListings scrapes car listings from Collecting Cars
func (s *CollectingCarsScraper) ScrapeCarListings(maxListings int) ([]*models.Car, error) {
	if !s.enabled {
		return nil, fmt.Errorf("collecting cars scraping is disabled")
	}

	// Initialize browser for scraping
	err := s.initBrowser()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize browser: %v", err)
	}
	defer s.closeBrowser()

	return s.scrapeWithBrowser(maxListings)
}

// login handles the login process for Collecting Cars
func (s *CollectingCarsScraper) login() error {
	if s.email == "" || s.password == "" {
		return fmt.Errorf("credentials not set - please set email and password")
	}

	page := s.browser.MustPage()
	defer page.Close()

	// Navigate to login page
	fmt.Println("Navigating to login page...")
	page.MustNavigate("https://collectingcars.com/login").MustWaitLoad()
	time.Sleep(3 * time.Second)

	// Fill in login form
	fmt.Println("Filling login form...")

	// Find and fill email input
	emailInput, err := page.Element("input[type='email'], input[name='email'], input[placeholder*='email']")
	if err == nil && emailInput != nil {
		emailInput.MustInput(s.email)
	}

	// Find and fill password input
	passwordInput, err := page.Element("input[type='password'], input[name='password']")
	if err == nil && passwordInput != nil {
		passwordInput.MustInput(s.password)
	}

	// Submit the form
	submitButton, err := page.Element("button[type='submit'], button:contains('Sign in'), button:contains('Log in')")
	if err == nil && submitButton != nil {
		submitButton.MustClick()
	}

	// Wait for login to complete
	time.Sleep(5 * time.Second)

	// Check if login was successful by looking for user menu or profile elements
	fmt.Println("Login completed, checking if successful...")

	return nil
}

// scrapeWithBrowser uses rod browser for scraping
func (s *CollectingCarsScraper) scrapeWithBrowser(maxListings int) ([]*models.Car, error) {
	// First, attempt to login
	if err := s.login(); err != nil {
		fmt.Printf("Warning: Login failed: %v\n", err)
		// Continue anyway as some content might be accessible
	}

	page := s.browser.MustPage()
	defer page.Close()

	// Navigate to sold cars in UK
	searchURL := "https://collectingcars.com/buy?refinementList%5BlistingStage%5D%5B0%5D=sold&refinementList%5BregionCode%5D%5B0%5D=UK&refinementList%5BlotType%5D%5B0%5D=car"

	fmt.Printf("Navigating to: %s\n", searchURL)
	page.MustNavigate(searchURL).MustWaitLoad()

	// Wait for the page to fully load
	time.Sleep(5 * time.Second)

	// Try to wait for specific elements that indicate the page has loaded
	page.MustWaitStable()

	var cars []*models.Car

	// Look for car listing links more specifically
	// Collecting Cars uses Next.js with specific link patterns
	linkSelectors := []string{
		"a[href*='/for-sale/']",
		"a[href^='/for-sale/']",
		".ListingCard a",
		"div[data-testid='listing-card'] a",
		"article a[href*='/for-sale/']",
	}

	fmt.Println("Looking for car listings...")

	var foundLinks []string
	for _, selector := range linkSelectors {
		elements, err := page.Elements(selector)
		if err != nil {
			fmt.Printf("Error with selector %s: %v\n", selector, err)
			continue
		}

		fmt.Printf("Found %d elements with selector: %s\n", len(elements), selector)

		for _, element := range elements {
			href, err := element.Attribute("href")
			if err == nil && href != nil && *href != "" {
				// Only add links that look like car listings
				if strings.Contains(*href, "/for-sale/") && !strings.Contains(*href, "#") {
					fullURL := *href
					if !strings.HasPrefix(fullURL, "http") {
						fullURL = "https://collectingcars.com" + fullURL
					}

					// Avoid duplicates
					isDuplicate := false
					for _, existing := range foundLinks {
						if existing == fullURL {
							isDuplicate = true
							break
						}
					}

					if !isDuplicate {
						foundLinks = append(foundLinks, fullURL)
						if len(foundLinks) >= maxListings {
							break
						}
					}
				}
			}
		}

		if len(foundLinks) >= maxListings {
			break
		}
	}

	fmt.Printf("Found %d unique car listing URLs\n", len(foundLinks))

	// Scrape each detail page
	for i, detailURL := range foundLinks {
		fmt.Printf("Scraping car %d/%d: %s\n", i+1, len(foundLinks), detailURL)

		car := s.scrapeDetailPage(detailURL)
		if car != nil && car.Price > 0 {
			cars = append(cars, car)
			fmt.Printf("✓ Scraped: %s %s (£%.0f, %d miles)\n",
				car.Make, car.Model, car.Price, car.Mileage)
		} else {
			fmt.Printf("✗ Failed to scrape car details from: %s\n", detailURL)
		}

		// Add delay between requests to be respectful
		if i < len(foundLinks)-1 {
			time.Sleep(2 * time.Second)
		}
	}

	if len(cars) == 0 {
		return nil, fmt.Errorf("no cars could be scraped from Collecting Cars")
	}

	return cars, nil
}

// scrapeDetailPage scrapes a single car detail page
func (s *CollectingCarsScraper) scrapeDetailPage(url string) *models.Car {
	page := s.browser.MustPage()
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Detail page panic recovered: %v\n", err)
		}
		page.Close()
	}()

	if err := page.Navigate(url); err != nil {
		fmt.Printf("Failed to navigate to %s: %v\n", url, err)
		return nil
	}

	page.MustWaitLoad()
	time.Sleep(5 * time.Second)
	page.MustWaitStable()

	car := &models.Car{
		ID:          fmt.Sprintf("cc-%d", time.Now().UnixNano()),
		OriginalURL: url,
	}

	// 1. Extract Year, Make, Model from listing title (h1)
	fmt.Println("Extracting title...")
	titleResult, err := page.Eval(`() => {
		const h1 = document.querySelector('h1');
		return h1 ? h1.textContent.trim() : '';
	}`)

	if err == nil {
		titleText := fmt.Sprintf("%v", titleResult.Value)
		if titleText != "" {
			fmt.Printf("Title: %s\n", titleText)
			// Parse "1995 BMW M3 Evolution Saloon" format
			parts := strings.Fields(titleText)
			if len(parts) >= 3 {
				// First part is year
				if year, err := strconv.Atoi(parts[0]); err == nil && year > 1900 && year < 2030 {
					car.Year = year
				}
				// Second part is make
				car.Make = parts[1]
				// Rest is model
				car.Model = strings.Join(parts[2:], " ")
			}
		}
	}

	// 2. Extract price from top bar (under headers with bids and date)
	fmt.Println("Extracting price from top bar...")
	priceResult, err := page.Eval(`() => {
		// Look for sold price in the header area
		const priceElements = document.querySelectorAll('*');
		for (let el of priceElements) {
			const text = el.textContent || '';
			// Look for pattern like "Sold for £XXX,XXX"
			if (text.includes('Sold for') && text.includes('£')) {
				// Get just this element's text, not children
				let ownText = Array.from(el.childNodes)
					.filter(node => node.nodeType === Node.TEXT_NODE)
					.map(node => node.textContent)
					.join('');
				if (ownText.includes('£')) {
					return ownText;
				}
				return text;
			}
		}
		return '';
	}`)

	if err == nil {
		soldPriceText := fmt.Sprintf("%v", priceResult.Value)
		if soldPriceText != "" {
			fmt.Printf("Found price text: %s\n", soldPriceText)
			// Extract price from "Sold for £123,456" format
			priceRegex := regexp.MustCompile(`£\s*([0-9,]+)`)
			if matches := priceRegex.FindStringSubmatch(soldPriceText); len(matches) > 1 {
				priceStr := strings.ReplaceAll(matches[1], ",", "")
				if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
					car.Price = price
				}
			}
		}
	}

	// 3. Extract Car Overview section (right sidebar)
	fmt.Println("Extracting car overview...")
	overviewResult, err := page.Eval(`() => {
		// Find the Car Overview section
		const headers = document.querySelectorAll('h2, h3, h4, h5');
		for (let header of headers) {
			if (header.textContent && header.textContent.toLowerCase().includes('car overview')) {
				// Get the container after this header
				let container = header.nextElementSibling;
				while (container && !container.textContent.includes('Mileage')) {
					container = container.nextElementSibling;
				}
				if (container) {
					return container.innerText;
				}
			}
		}
		// Alternative: look for the section containing car specs
		const sections = document.querySelectorAll('section, div');
		for (let section of sections) {
			const text = section.innerText || '';
			if (text.includes('Mileage') && text.includes('Transmission') && 
				(text.includes('RHD') || text.includes('LHD'))) {
				return text;
			}
		}
		return '';
	}`)

	if err == nil {
		overviewText := fmt.Sprintf("%v", overviewResult.Value)
		if overviewText != "" {
			fmt.Printf("Car Overview text:\n%s\n", overviewText)
			s.parseCarOverview(overviewText, car)
		}
	}

	// 4. Extract Lot Overview for location
	fmt.Println("Extracting lot overview...")
	lotResult, err := page.Eval(`() => {
		const headers = document.querySelectorAll('h2, h3, h4, h5');
		for (let header of headers) {
			if (header.textContent && header.textContent.toLowerCase().includes('lot overview')) {
				let container = header.nextElementSibling;
				if (container) {
					return container.innerText;
				}
			}
		}
		return '';
	}`)

	if err == nil {
		lotText := fmt.Sprintf("%v", lotResult.Value)
		if lotText != "" && strings.Contains(lotText, "Location") {
			fmt.Printf("Lot location: %s\n", lotText)
		}
	}

	// 5. Extract images
	fmt.Println("Extracting images...")
	imageResult, err := page.Eval(`() => {
		const images = [];
		const imgs = document.querySelectorAll('img');
		for (let img of imgs) {
			if (img.src && 
				(img.src.includes('collectingcars') || 
				 img.src.includes('cloudinary') ||
				 img.src.includes('imgix')) &&
				!img.src.includes('logo') &&
				!img.src.includes('icon') &&
				img.width > 100) {
				images.push(img.src);
			}
		}
		return JSON.stringify(images.slice(0, 10));
	}`)

	if err == nil {
		imageJSON := fmt.Sprintf("%v", imageResult.Value)
		if imageJSON != "" && imageJSON != "[]" {
			// Parse JSON array
			imageJSON = strings.Trim(imageJSON, `"`)
			var images []string
			err := json.Unmarshal([]byte(imageJSON), &images)
			if err == nil {
				car.Images = images
				fmt.Printf("Found %d images\n", len(images))
			}
		}
	}

	fmt.Printf("Scraped car: %+v\n", car)
	return car
}

// parseCarOverview parses the car overview section with the simple key-value format
func (s *CollectingCarsScraper) parseCarOverview(text string, car *models.Car) {
	// Split by newlines
	lines := strings.Split(text, "\n")

	// The format is typically:
	// Mileage
	// 45,000 miles
	// Transmission
	// Manual
	// etc.

	for i := 0; i < len(lines)-1; i++ {
		key := strings.TrimSpace(lines[i])
		value := ""
		if i+1 < len(lines) {
			value = strings.TrimSpace(lines[i+1])
		}

		keyLower := strings.ToLower(key)

		switch {
		case strings.Contains(keyLower, "mileage"):
			// Extract numeric mileage from strings like "45,000 miles" or "45000"
			mileageStr := strings.ReplaceAll(value, ",", "")
			mileageStr = strings.ReplaceAll(mileageStr, " ", "")
			mileageStr = strings.ReplaceAll(mileageStr, "miles", "")
			mileageStr = strings.ReplaceAll(mileageStr, "mile", "")
			if m, err := strconv.Atoi(strings.TrimSpace(mileageStr)); err == nil {
				car.Mileage = m
			}
			i++ // Skip the value line

		case strings.Contains(keyLower, "transmission"):
			car.Gearbox = value
			i++

		case strings.Contains(keyLower, "rhd") || strings.Contains(keyLower, "lhd"):
			// Store drive side if needed
			i++

		case strings.Contains(keyLower, "colour") && !strings.Contains(keyLower, "upholstery"):
			car.BodyColour = value
			i++

		case strings.Contains(keyLower, "upholstery"):
			// Interior color - we could add this field if needed
			i++

		case strings.Contains(keyLower, "engine type"):
			car.Engine = value
			i++

		case strings.Contains(keyLower, "fuel"):
			car.FuelType = value
			i++

		case strings.Contains(keyLower, "body"):
			car.BodyType = value
			i++

		case strings.Contains(keyLower, "doors"):
			// Extract number from "4 doors" or just "4"
			doorsRegex := regexp.MustCompile(`(\d+)`)
			if matches := doorsRegex.FindStringSubmatch(value); len(matches) > 1 {
				car.Doors = matches[1]
			}
			i++

		case strings.Contains(keyLower, "seats"):
			// Extract number from "5 seats" or just "5"
			seatsRegex := regexp.MustCompile(`(\d+)`)
			if matches := seatsRegex.FindStringSubmatch(value); len(matches) > 1 {
				car.Seats = matches[1]
			}
			i++
		}
	}
}

func (s *CollectingCarsScraper) initBrowser() error {
	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	// Use a persistent user data directory to store cookies/session
	userDataDir := "./collecting_cars_session"

	l := launcher.New().
		Headless(true).
		Set("user-agent", userAgent).
		UserDataDir(userDataDir) // This will persist cookies between sessions

	url, err := l.Launch()
	if err != nil {
		return err
	}

	s.browser = rod.New().ControlURL(url)
	return s.browser.Connect()
}

func (s *CollectingCarsScraper) closeBrowser() {
	if s.browser != nil {
		s.browser.Close()
	}
}
