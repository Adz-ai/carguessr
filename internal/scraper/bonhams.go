package scraper

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"autotraderguesser/internal/models"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

const (
	// Maximum number of concurrent detail page scrapers
	maxConcurrentScrapers = 5
	// Timeout for individual page loads
	pageLoadTimeout = 10 * time.Second
	// Minimum delay between requests to same domain
	minRequestDelay = 500 * time.Millisecond
)

// BonhamsScraper handles scraping from Bonhams Car Auctions
type BonhamsScraper struct {
	browser *rod.Browser
	enabled bool
}

// NewBonhamsScraper creates a new Bonhams scraper
func NewBonhamsScraper() *BonhamsScraper {
	return &BonhamsScraper{
		enabled: true,
	}
}

// Enable enables the scraper
func (s *BonhamsScraper) Enable() {
	s.enabled = true
}

// ScrapeCarListings scrapes car listings from Bonhams auction results
func (s *BonhamsScraper) ScrapeCarListings(maxListings int) ([]*models.BonhamsCar, error) {
	if !s.enabled {
		return nil, fmt.Errorf("bonhams scraping is disabled")
	}

	// Initialize browser for scraping
	err := s.initBrowser()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize browser: %v", err)
	}
	defer s.closeBrowser()

	return s.scrapeWithBrowser(maxListings)
}

// scrapeWithBrowser uses rod browser for scraping with pagination support
func (s *BonhamsScraper) scrapeWithBrowser(maxListings int) ([]*models.BonhamsCar, error) {
	page := s.browser.MustPage()
	defer page.Close()

	var cars []*models.BonhamsCar
	var allFoundLinks []string

	// Calculate how many pages we need (18 listings per page)
	// Since we're filtering for SOLD only at the listing level, we need more pages
	listingsPerPage := 18
	pagesNeeded := (maxListings + listingsPerPage - 1) / listingsPerPage // Round up

	// Add extra pages since many cars will be filtered out (didn't meet reserve)
	pagesNeeded = pagesNeeded * 3 // Triple the pages to account for filtering
	if pagesNeeded > 50 {
		pagesNeeded = 50 // Increased cap for 250 cars
	}

	fmt.Printf("🔍 Attempting to scrape %d sold listings across %d pages\n", maxListings, pagesNeeded)

	// Scrape multiple pages
	for pageNum := 1; pageNum <= pagesNeeded && len(allFoundLinks) < maxListings; pageNum++ {
		searchURL := fmt.Sprintf("https://carsonline.bonhams.com/en/auctions/results?page=%d", pageNum)

		fmt.Printf("📍 Navigating to page %d: %s\n", pageNum, searchURL)
		page.MustNavigate(searchURL)

		// Use WaitStable for smarter waiting
		page.MustWaitStable()

		// Look for car listing elements - Bonhams uses /listings/ URLs
		linkSelectors := []string{
			"a[href*='/listings/']",
			".listing-card a",
			".listing-card",
			"a[href*='/en/listings/']",
		}

		fmt.Printf("🔍 Looking for car listings on page %d...\n", pageNum)

		var pageFoundLinks []string
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
					if strings.Contains(*href, "/listings/") && !strings.Contains(*href, "#") {
						// Check if this is a sold item by examining the card content
						cardText, _ := element.Text()
						cardTextLower := strings.ToLower(cardText)

						// Skip items that show "Bid to" as these didn't meet reserve
						if strings.Contains(cardTextLower, "bid to") {
							fmt.Printf("Skipping unsold item (bid to): %s\n", *href)
							continue
						}

						// Look for indicators that the item sold
						isSold := strings.Contains(cardTextLower, "sold for") || strings.Contains(cardTextLower, "hammer price")

						if !isSold {
							fmt.Printf("Skipping item (no sold indicator): %s\n", *href)
							continue
						}

						fullURL := *href
						if !strings.HasPrefix(fullURL, "http") {
							fullURL = "https://carsonline.bonhams.com" + fullURL
						}

						// Avoid duplicates within this page
						isDuplicate := false
						for _, existing := range pageFoundLinks {
							if existing == fullURL {
								isDuplicate = true
								break
							}
						}

						if !isDuplicate {
							pageFoundLinks = append(pageFoundLinks, fullURL)
							fmt.Printf("✅ Added sold item: %s\n", fullURL)
						}
					}
				}
			}
		}

		// Add page links to total, avoiding cross-page duplicates
		for _, link := range pageFoundLinks {
			isDuplicate := false
			for _, existing := range allFoundLinks {
				if existing == link {
					isDuplicate = true
					break
				}
			}

			if !isDuplicate {
				allFoundLinks = append(allFoundLinks, link)
				if len(allFoundLinks) >= maxListings {
					break
				}
			}
		}

		fmt.Printf("✅ Page %d: Found %d new links (total: %d)\n", pageNum, len(pageFoundLinks), len(allFoundLinks))

		// Stop if we have enough listings
		if len(allFoundLinks) >= maxListings {
			break
		}

		// Minimal delay between page requests
		if pageNum < pagesNeeded {
			time.Sleep(minRequestDelay)
		}
	}

	fmt.Printf("🎉 Found %d unique car listing URLs across %d pages\n", len(allFoundLinks), pagesNeeded)

	// If no links found with selectors, try JavaScript to find links
	if len(allFoundLinks) == 0 {
		fmt.Println("No links found with CSS selectors, trying JavaScript approach...")

		linkResult, err := page.Eval(`() => {
			const links = [];
			const allLinks = document.querySelectorAll('a[href]');
			for (let link of allLinks) {
				const href = link.href;
				if (href.includes('/listings/') && 
					!href.includes('#') && 
					(href.includes('bonhams.com') || href.startsWith('/'))) {
					links.push(href);
				}
			}
			return JSON.stringify([...new Set(links)]);
		}`)

		if err == nil {
			linkJSON := fmt.Sprintf("%v", linkResult.Value)
			if linkJSON != "" && linkJSON != "[]" {
				linkJSON = strings.Trim(linkJSON, `"`)
				var links []string
				err := json.Unmarshal([]byte(linkJSON), &links)
				if err == nil {
					allFoundLinks = links
					if len(allFoundLinks) > maxListings {
						allFoundLinks = allFoundLinks[:maxListings]
					}
					fmt.Printf("Found %d links via JavaScript\n", len(allFoundLinks))
				}
			}
		}
	}

	// Limit to requested number of listings
	if len(allFoundLinks) > maxListings {
		allFoundLinks = allFoundLinks[:maxListings]
	}

	// Scrape detail pages in parallel
	fmt.Printf("🚀 Starting parallel scraping of %d cars with %d workers\n", len(allFoundLinks), maxConcurrentScrapers)
	startTime := time.Now()

	type result struct {
		car *models.BonhamsCar
		url string
		err error
	}

	// Create channels for work distribution
	urlChan := make(chan string, len(allFoundLinks))
	resultChan := make(chan result, len(allFoundLinks))

	// Create worker pool
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrentScrapers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for url := range urlChan {
				fmt.Printf("[Worker %d] Scraping: %s\n", workerID, url)
				car := s.scrapeDetailPageConcurrent(url)
				if car != nil && car.Price > 0 {
					resultChan <- result{car: car, url: url}
					fmt.Printf("[Worker %d] ✅ Success: %s %s %d (£%.0f)\n",
						workerID, car.Make, car.Model, car.Year, car.Price)
				} else {
					resultChan <- result{car: nil, url: url, err: fmt.Errorf("failed to scrape")}
					fmt.Printf("[Worker %d] ❌ Failed: %s\n", workerID, url)
				}
				// Small delay to avoid overwhelming the server
				time.Sleep(minRequestDelay)
			}
		}(i)
	}

	// Queue all URLs
	for _, url := range allFoundLinks {
		urlChan <- url
	}
	close(urlChan)

	// Wait for all workers to complete
	wg.Wait()
	close(resultChan)

	// Collect results
	for result := range resultChan {
		if result.car != nil {
			cars = append(cars, result.car)
		}
	}

	elapsed := time.Since(startTime)
	fmt.Printf("⏱️  Scraping completed in %.2f seconds (%.2f cars/second)\n",
		elapsed.Seconds(), float64(len(cars))/elapsed.Seconds())

	if len(cars) == 0 {
		return nil, fmt.Errorf("no sold cars could be found from Bonhams")
	}

	fmt.Printf("🎉 Successfully scraped %d sold cars from Bonhams\n", len(cars))
	return cars, nil
}

// scrapeDetailPage scrapes a single car detail page from Bonhams
func (s *BonhamsScraper) scrapeDetailPage(url string) *models.BonhamsCar {
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

	car := &models.BonhamsCar{
		ID:          fmt.Sprintf("bonhams-%d", time.Now().UnixNano()),
		OriginalURL: url,
	}

	// 1. Extract title (usually contains Year Make Model)
	fmt.Println("Extracting title...")
	titleSelectors := []string{
		"h1",
		".lot-title",
		".auction-title",
		"[data-testid='lot-title']",
		".page-title",
	}

	for _, selector := range titleSelectors {
		titleResult, err := page.Eval(fmt.Sprintf(`() => {
			const element = document.querySelector('%s');
			return element ? element.textContent.trim() : '';
		}`, selector))

		if err == nil {
			titleText := strings.TrimSpace(fmt.Sprintf("%v", titleResult.Value))
			if titleText != "" {
				fmt.Printf("Title: %s\n", titleText)
				s.parseTitle(titleText, car)
				break
			}
		}
	}

	// 2. Extract price using precise selector
	fmt.Println("Extracting price...")
	priceResult, err := page.Eval(`() => {
		// Use the precise selector provided by the user
		const priceElement = document.querySelector('.listing-state__value.listing-final-price p[data-qa="listing highest bid value"]');
		if (priceElement) {
			const priceText = priceElement.textContent.trim();
			const priceMatch = priceText.match(/£\s*([\d,]+)/);
			if (priceMatch) {
				return priceMatch[0];
			}
		}
		
		// Fallback to broader search if precise selector doesn't work
		const fallbackSelectors = [
			'.listing-state__value p',
			'[data-qa="listing highest bid value"]',
			'.listing-final-price',
			'.sold-price',
			'.hammer-price'
		];
		
		for (let selector of fallbackSelectors) {
			const el = document.querySelector(selector);
			if (el) {
				const text = el.textContent || '';
				const priceMatch = text.match(/£\s*([\d,]+)/);
				if (priceMatch) {
					return priceMatch[0];
				}
			}
		}
		
		return '';
	}`)

	if err == nil {
		priceText := strings.TrimSpace(fmt.Sprintf("%v", priceResult.Value))
		if priceText != "" {
			fmt.Printf("Found price text: %s\n", priceText)
			s.parsePrice(priceText, car)
		}
	}

	// 3. Extract sale date using the provided selector
	fmt.Println("Extracting sale date...")
	saleDateResult, err := page.Eval(`() => {
		// Use the precise selector provided by the user
		const dateElement = document.querySelector('.countdown__wrapper .end-date[data-qa="listing end date"]');
		if (dateElement) {
			const dateText = dateElement.textContent.trim();
			// Extract just the date part, removing the time
			const dateMatch = dateText.match(/(\d{1,2}\s+\w+\s+\d{4})/);
			if (dateMatch) {
				return dateMatch[1];
			}
		}
		
		// Fallback selectors
		const fallbackSelectors = [
			'[data-qa="listing end date"]',
			'.end-date',
			'.countdown__wrapper span'
		];
		
		for (let selector of fallbackSelectors) {
			const el = document.querySelector(selector);
			if (el) {
				const text = el.textContent || '';
				const dateMatch = text.match(/(\d{1,2}\s+\w+\s+\d{4})/);
				if (dateMatch) {
					return dateMatch[1];
				}
			}
		}
		
		return '';
	}`)

	if err == nil {
		saleDateText := strings.TrimSpace(fmt.Sprintf("%v", saleDateResult.Value))
		if saleDateText != "" {
			car.SaleDate = saleDateText
			fmt.Printf("Found sale date: %s\n", saleDateText)
		}
	}

	// 4. Extract vehicle location
	fmt.Println("Extracting vehicle location...")
	locationResult, err := page.Eval(`() => {
		// Use the precise selector provided by the user
		const locationElement = document.querySelector('.auction-info-details__location .text[data-v-0a8ab3c9]');
		if (locationElement) {
			return locationElement.textContent.trim();
		}
		
		// Fallback selectors
		const fallbackSelectors = [
			'[data-qa="auction information location"]',
			'.auction-info-details__location .text',
			'.icon-text[data-qa="auction information location"] .text'
		];
		
		for (let selector of fallbackSelectors) {
			const el = document.querySelector(selector);
			if (el) {
				return el.textContent.trim();
			}
		}
		
		return '';
	}`)

	if err == nil {
		locationText := strings.TrimSpace(fmt.Sprintf("%v", locationResult.Value))
		if locationText != "" {
			car.Location = locationText
			fmt.Printf("Found location: %s\n", locationText)
		}
	}

	// 5. Extract specifications from data-qa attributes
	fmt.Println("Extracting specifications from data-qa attributes...")
	specsResult, err := page.Eval(`() => {
		const specs = {};
		
		// Extract from auction information stats
		const qaSelectors = [
			'li[data-qa="auction information stat chassis"]',
			'li[data-qa="auction information stat mileage"]',
			'li[data-qa="auction information stat engine"]',
			'li[data-qa="auction information stat gearbox"]',
			'li[data-qa="auction information stat color"]',
			'li[data-qa="auction information stat interior"]',
			'li[data-qa="auction information stat steering"]',
			'li[data-qa="auction information stat fuel_type"]'
		];
		
		for (let selector of qaSelectors) {
			const element = document.querySelector(selector);
			if (element) {
				const textElement = element.querySelector('.text');
				if (textElement) {
					const qaType = element.getAttribute('data-qa').split(' ').pop(); // Gets the last part (chassis, mileage, etc.)
					specs[qaType] = textElement.textContent.trim();
				}
			}
		}
		
		return JSON.stringify(specs);
	}`)

	if err == nil {
		specsJSON := strings.TrimSpace(fmt.Sprintf("%v", specsResult.Value))
		if specsJSON != "" && specsJSON != "{}" {
			fmt.Printf("Specifications JSON: %s\n", specsJSON)
			s.parseSpecsFromDataQA(specsJSON, car)
		}
	}

	// 6. Extract key facts from data-qa="auction information key fact"
	fmt.Println("Extracting key facts...")
	keyFactsResult, err := page.Eval(`() => {
		const keyFactElements = document.querySelectorAll('li[data-qa="auction information key fact"]');
		const keyFacts = [];
		
		for (let element of keyFactElements) {
			const fact = element.textContent.trim();
			if (fact) {
				keyFacts.push(fact);
			}
		}
		
		return JSON.stringify(keyFacts);
	}`)

	if err == nil {
		keyFactsJSON := strings.TrimSpace(fmt.Sprintf("%v", keyFactsResult.Value))
		if keyFactsJSON != "" && keyFactsJSON != "[]" {
			keyFactsJSON = strings.Trim(keyFactsJSON, `"`)
			var keyFacts []string
			err := json.Unmarshal([]byte(keyFactsJSON), &keyFacts)
			if err == nil {
				car.KeyFacts = keyFacts
				// Create description from key facts
				car.Description = strings.Join(keyFacts, " • ")
				fmt.Printf("Found %d key facts: %v\n", len(keyFacts), keyFacts)
			}
		}
	}

	// 7. Extract images
	fmt.Println("Extracting images...")
	imageResult, err := page.Eval(`() => {
		const images = [];
		const imgs = document.querySelectorAll('img');
		for (let img of imgs) {
			if (img.src && 
				(img.src.includes('bonhams') || 
				 img.src.includes('twic.pics') ||
				 img.src.includes('cloudinary') ||
				 img.src.includes('imgix') ||
				 img.src.includes('amazonaws')) &&
				!img.src.includes('logo') &&
				!img.src.includes('icon') &&
				img.width > 100) {
				
				let enhancedUrl = img.src;
				// For twic.pics URLs, simply append /cover=900x700 for high quality
				if (img.src.includes('twic.pics')) {
					// Remove any existing resize or cover parameters first
					enhancedUrl = img.src.replace(/\/resize=\d+/g, '').replace(/\/cover=[^/]*/g, '');
					// Add high quality cover parameter
					enhancedUrl += '/cover=900x700';
				}
				
				images.push(enhancedUrl);
			}
		}
		return JSON.stringify([...new Set(images.slice(0, 10))]);
	}`)

	if err == nil {
		imageJSON := fmt.Sprintf("%v", imageResult.Value)
		if imageJSON != "" && imageJSON != "[]" {
			imageJSON = strings.Trim(imageJSON, `"`)
			var images []string
			err := json.Unmarshal([]byte(imageJSON), &images)
			if err == nil {
				car.Images = images
				fmt.Printf("Found %d images\n", len(images))
			}
		}
	}

	fmt.Printf("Scraped car: Make=%s, Model=%s, Year=%d, Price=%.0f, Mileage=%s\n",
		car.Make, car.Model, car.Year, car.Price, car.Mileage)
	return car
}

// scrapeDetailPageConcurrent is an optimized version for concurrent scraping
func (s *BonhamsScraper) scrapeDetailPageConcurrent(url string) *models.BonhamsCar {
	// Create a new page for concurrent scraping
	page := s.browser.MustPage()
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Detail page panic recovered: %v\n", err)
		}
		page.Close()
	}()

	// Set timeout for page operations
	page = page.Timeout(pageLoadTimeout)

	if err := page.Navigate(url); err != nil {
		fmt.Printf("Failed to navigate to %s: %v\n", url, err)
		return nil
	}

	// Use WaitStable for faster page loading
	page.MustWaitStable()

	car := &models.BonhamsCar{
		ID:          fmt.Sprintf("bonhams-%d", time.Now().UnixNano()),
		OriginalURL: url,
	}

	// Execute all extractions in parallel using a single JavaScript evaluation
	extractionScript := `() => {
		const result = {
			title: '',
			price: '',
			saleDate: '',
			location: '',
			specs: {},
			keyFacts: [],
			images: []
		};

		// Extract title
		const titleSelectors = ['h1', '.lot-title', '.auction-title', '[data-testid="lot-title"]', '.page-title'];
		for (let selector of titleSelectors) {
			const element = document.querySelector(selector);
			if (element && element.textContent.trim()) {
				result.title = element.textContent.trim();
				break;
			}
		}

		// Extract price
		const priceElement = document.querySelector('.listing-state__value.listing-final-price p[data-qa="listing highest bid value"]');
		if (priceElement) {
			const priceText = priceElement.textContent.trim();
			const priceMatch = priceText.match(/£\s*([\d,]+)/);
			if (priceMatch) {
				result.price = priceMatch[0];
			}
		}

		// Extract sale date
		const dateElement = document.querySelector('.countdown__wrapper .end-date[data-qa="listing end date"]');
		if (dateElement) {
			const dateText = dateElement.textContent.trim();
			const dateMatch = dateText.match(/(\d{1,2}\s+\w+\s+\d{4})/);
			if (dateMatch) {
				result.saleDate = dateMatch[1];
			}
		}

		// Extract location
		const locationElement = document.querySelector('.auction-info-details__location .text[data-v-0a8ab3c9]');
		if (locationElement) {
			result.location = locationElement.textContent.trim();
		}

		// Extract specifications
		const qaSelectors = [
			'li[data-qa="auction information stat chassis"]',
			'li[data-qa="auction information stat mileage"]',
			'li[data-qa="auction information stat engine"]',
			'li[data-qa="auction information stat gearbox"]',
			'li[data-qa="auction information stat color"]',
			'li[data-qa="auction information stat interior"]',
			'li[data-qa="auction information stat steering"]',
			'li[data-qa="auction information stat fuel_type"]'
		];
		
		for (let selector of qaSelectors) {
			const element = document.querySelector(selector);
			if (element) {
				const textElement = element.querySelector('.text');
				if (textElement) {
					const qaType = element.getAttribute('data-qa').split(' ').pop();
					result.specs[qaType] = textElement.textContent.trim();
				}
			}
		}

		// Extract key facts
		const keyFactElements = document.querySelectorAll('li[data-qa="auction information key fact"]');
		for (let element of keyFactElements) {
			const fact = element.textContent.trim();
			if (fact) {
				result.keyFacts.push(fact);
			}
		}

		// Extract images
		const imgs = document.querySelectorAll('img');
		for (let img of imgs) {
			if (img.src && 
				(img.src.includes('bonhams') || img.src.includes('twic.pics') || 
				 img.src.includes('cloudinary') || img.src.includes('imgix') || 
				 img.src.includes('amazonaws')) &&
				!img.src.includes('logo') && !img.src.includes('icon') && img.width > 100) {
				
				let enhancedUrl = img.src;
				// For twic.pics URLs, simply append /cover=800x600 for high quality
				if (img.src.includes('twic.pics')) {
					// Remove any existing resize or cover parameters first
					enhancedUrl = img.src.replace(/\/resize=\d+/g, '').replace(/\/cover=[^/]*/g, '');
					// Add high quality cover parameter
					enhancedUrl += '/cover=800x600';
				}
				
				result.images.push(enhancedUrl);
			}
		}
		result.images = [...new Set(result.images.slice(0, 10))];

		return JSON.stringify(result);
	}`

	extractResult := page.MustEval(extractionScript)

	// Parse the extraction results
	var extracted struct {
		Title    string            `json:"title"`
		Price    string            `json:"price"`
		SaleDate string            `json:"saleDate"`
		Location string            `json:"location"`
		Specs    map[string]string `json:"specs"`
		KeyFacts []string          `json:"keyFacts"`
		Images   []string          `json:"images"`
	}

	if err := json.Unmarshal([]byte(extractResult.String()), &extracted); err != nil {
		fmt.Printf("Failed to parse extraction results: %v\n", err)
		return nil
	}

	// Process extracted data
	s.parseTitle(extracted.Title, car)
	s.parsePrice(extracted.Price, car)
	car.SaleDate = extracted.SaleDate
	car.Location = extracted.Location
	car.Images = extracted.Images
	car.KeyFacts = extracted.KeyFacts
	car.Description = strings.Join(extracted.KeyFacts, " • ")

	// Process specifications
	for key, value := range extracted.Specs {
		switch key {
		case "mileage":
			car.Mileage = value
			s.extractNumericMileage(value, car)
		case "engine":
			car.Engine = value
		case "gearbox":
			car.Gearbox = value
		case "color":
			car.ExteriorColor = value
		case "interior":
			car.InteriorColor = value
		case "steering":
			car.Steering = value
		case "fuel_type":
			car.FuelType = value
		}
	}

	return car
}

// parseSpecsFromDataQA parses specifications extracted from data-qa attributes
func (s *BonhamsScraper) parseSpecsFromDataQA(specsJSON string, car *models.BonhamsCar) {
	var specs map[string]interface{}
	err := json.Unmarshal([]byte(specsJSON), &specs)
	if err != nil {
		fmt.Printf("Error parsing specifications JSON: %v\n", err)
		return
	}

	for key, value := range specs {
		valueStr := fmt.Sprintf("%v", value)

		switch key {
		case "chassis":
			// Skip chassis as requested - don't store it
			continue
		case "mileage":
			car.Mileage = valueStr
			// Also extract numeric version for calculations
			s.extractNumericMileage(valueStr, car)
		case "engine":
			car.Engine = valueStr
		case "gearbox":
			car.Gearbox = valueStr
		case "color":
			car.ExteriorColor = valueStr
		case "interior":
			car.InteriorColor = valueStr
		case "steering":
			car.Steering = valueStr
		case "fuel_type":
			car.FuelType = valueStr
		}
	}
}

// extractNumericMileage extracts numeric mileage for calculations
func (s *BonhamsScraper) extractNumericMileage(mileageStr string, car *models.BonhamsCar) {
	// Remove common words and extract numbers
	cleanStr := strings.ToLower(mileageStr)
	cleanStr = strings.ReplaceAll(cleanStr, "miles", "")
	cleanStr = strings.ReplaceAll(cleanStr, "mile", "")
	cleanStr = strings.ReplaceAll(cleanStr, ",", "")
	cleanStr = strings.TrimSpace(cleanStr)

	// Extract first number found
	mileageRegex := regexp.MustCompile(`(\d+)`)
	if matches := mileageRegex.FindStringSubmatch(cleanStr); len(matches) > 1 {
		if mileage, err := strconv.Atoi(matches[1]); err == nil {
			car.MileageNumeric = mileage
		}
	}
}

// parseTitle extracts year, make, and model from title
func (s *BonhamsScraper) parseTitle(title string, car *models.BonhamsCar) {
	// Common formats:
	// "1965 Aston Martin DB5"
	// "1973 Porsche 911 Carrera RS 2.7"
	// "2019 McLaren 720S Spider"

	parts := strings.Fields(title)
	if len(parts) >= 3 {
		// First part is usually year
		if year, err := strconv.Atoi(parts[0]); err == nil && year > 1900 && year < 2030 {
			car.Year = year
		}

		// Second part is make
		if len(parts) > 1 {
			car.Make = parts[1]
		}

		// Rest is model
		if len(parts) > 2 {
			car.Model = strings.Join(parts[2:], " ")
		}
	}
}

// parsePrice extracts price from price text
func (s *BonhamsScraper) parsePrice(priceText string, car *models.BonhamsCar) {
	// Extract number from format like "£29,000"
	priceRegex := regexp.MustCompile(`£\s*([0-9,]+)`)
	matches := priceRegex.FindStringSubmatch(priceText)

	if len(matches) > 1 {
		priceStr := strings.ReplaceAll(matches[1], ",", "")
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			car.Price = price
		}
	}
}

func (s *BonhamsScraper) initBrowser() error {
	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	l := launcher.New().
		Headless(true).
		Set("user-agent", userAgent).
		Set("disable-blink-features", "AutomationControlled").
		Set("disable-web-security").
		Set("allow-running-insecure-content").
		Set("max-connections-per-host", "10"). // Increase concurrent connections
		Set("aggressive-cache-discard").       // Reduce memory usage
		Set("disable-dev-shm-usage")           // Use /tmp instead of /dev/shm

	url, err := l.Launch()
	if err != nil {
		return err
	}

	s.browser = rod.New().ControlURL(url)
	return s.browser.Connect()
}

func (s *BonhamsScraper) closeBrowser() {
	if s.browser != nil {
		s.browser.Close()
	}
}
