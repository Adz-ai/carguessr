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
	if pagesNeeded > 15 {
		pagesNeeded = 15 // Cap at 15 pages to be reasonable
	}

	fmt.Printf("🔍 Attempting to scrape %d sold listings across %d pages\n", maxListings, pagesNeeded)

	// Scrape multiple pages
	for pageNum := 1; pageNum <= pagesNeeded && len(allFoundLinks) < maxListings; pageNum++ {
		searchURL := fmt.Sprintf("https://carsonline.bonhams.com/en/auctions/results?page=%d", pageNum)

		fmt.Printf("📍 Navigating to page %d: %s\n", pageNum, searchURL)
		page.MustNavigate(searchURL).MustWaitLoad()

		// Wait for the page to fully load
		time.Sleep(3 * time.Second)
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
						isSold := strings.Contains(cardTextLower, "sold for") ||
							strings.Contains(cardTextLower, "hammer price") ||
							(strings.Contains(cardTextLower, "sold") && !strings.Contains(cardTextLower, "bid to"))

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

		// Don't spam the server - small delay between pages
		if pageNum < pagesNeeded {
			time.Sleep(2 * time.Second)
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

	// Scrape each detail page (already filtered for sold items)
	for i, detailURL := range allFoundLinks {
		fmt.Printf("Scraping sold car %d/%d: %s\n", i+1, len(allFoundLinks), detailURL)

		car := s.scrapeDetailPage(detailURL)
		if car != nil && car.Price > 0 {
			cars = append(cars, car)
			fmt.Printf("✅ Sold car added: %s %s %d (£%.0f, %s)\n",
				car.Make, car.Model, car.Year, car.Price, car.Mileage)
		} else {
			fmt.Printf("❌ Failed to scrape car details: %s\n", detailURL)
		}

		// Add delay between requests to be respectful
		if i < len(allFoundLinks)-1 {
			time.Sleep(2 * time.Second)
		}
	}

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

	// 2. Extract price AND sale status (must be SOLD, not just bid)
	fmt.Println("Extracting price and sale status...")
	priceResult, err := page.Eval(`() => {
		const result = { price: '', status: '', fullText: '' };
		
		const priceSelectors = [
			'.price',
			'.sold-price',
			'.hammer-price',
			'.estimate',
			'*[class*="price"]',
			'*[class*="sold"]',
			'*[class*="hammer"]',
			'*[data-qa*="price"]',
			'*[data-qa*="sold"]'
		];
		
		// Look for price and status information
		for (let selector of priceSelectors) {
			const elements = document.querySelectorAll(selector);
			for (let el of elements) {
				const text = (el.textContent || '').toLowerCase();
				const originalText = el.textContent || '';
				
				// Check if this element contains price information
				if (text.includes('£') && text.match(/£\s*[\d,]+/)) {
					result.fullText = originalText;
					
					// Extract the price
					const priceMatch = originalText.match(/£\s*([\d,]+)/);
					if (priceMatch) {
						result.price = priceMatch[0];
					}
					
					// Determine sale status - CRITICAL: only accept SOLD items
					if (text.includes('sold') && !text.includes('bid to')) {
						result.status = 'sold';
						return JSON.stringify(result);
					} else if (text.includes('bid to') || text.includes('bidding')) {
						result.status = 'bid_to'; // Did not meet reserve
						return JSON.stringify(result);
					} else if (text.includes('hammer') && !text.includes('bid to')) {
						result.status = 'sold'; // Hammer price usually means sold
						return JSON.stringify(result);
					}
				}
			}
		}
		
		// Broader search for sale status in the entire page
		const allElements = document.querySelectorAll('*');
		for (let el of allElements) {
			const text = (el.textContent || '').toLowerCase();
			const originalText = el.textContent || '';
			
			// Look for explicit sale status
			if (text.includes('sold for £') || 
				text.includes('hammer price £') ||
				(text.includes('sold') && text.includes('£') && text.match(/£\s*[\d,]+/))) {
				
				const priceMatch = originalText.match(/£\s*([\d,]+)/);
				if (priceMatch) {
					result.price = priceMatch[0];
					result.fullText = originalText;
					result.status = 'sold';
					return JSON.stringify(result);
				}
			}
			
			// Check for "bid to" which means reserve not met
			if (text.includes('bid to £') || 
				text.includes('bidding to £') ||
				(text.includes('bid to') && text.includes('£'))) {
				
				const priceMatch = originalText.match(/£\s*([\d,]+)/);
				if (priceMatch) {
					result.price = priceMatch[0];
					result.fullText = originalText;
					result.status = 'bid_to';
					return JSON.stringify(result);
				}
			}
		}
		
		return JSON.stringify(result);
	}`)

	if err == nil {
		priceText := strings.TrimSpace(fmt.Sprintf("%v", priceResult.Value))
		if priceText != "" {
			fmt.Printf("Found price text: %s\n", priceText)
			s.parsePrice(priceText, car)
		}
	}

	// 3. Extract specifications from data-qa attributes
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

	// 4. Extract key facts from data-qa="auction information key fact"
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

	// 5. Extract images
	fmt.Println("Extracting images...")
	imageResult, err := page.Eval(`() => {
		const images = [];
		const imgs = document.querySelectorAll('img');
		for (let img of imgs) {
			if (img.src && 
				(img.src.includes('bonhams') || 
				 img.src.includes('cloudinary') ||
				 img.src.includes('imgix') ||
				 img.src.includes('amazonaws')) &&
				!img.src.includes('logo') &&
				!img.src.includes('icon') &&
				img.width > 100) {
				images.push(img.src);
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
	// Look for patterns like:
	// "Sold for £123,456"
	// "Hammer price: £98,765"
	// "£45,000 - £65,000 (estimate)"

	priceRegex := regexp.MustCompile(`£\s*([0-9,]+)`)
	matches := priceRegex.FindAllStringSubmatch(priceText, -1)

	if len(matches) > 0 {
		// Take the first price found (usually the sold price)
		priceStr := strings.ReplaceAll(matches[0][1], ",", "")
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
		Set("allow-running-insecure-content")

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
