# Motors Price Guesser Game 🚗💸

A fun web-based game where players guess the prices of real cars from Motors.co.uk listings. Built with Go/Gin backend and vanilla JavaScript frontend.

## Features

- **Real Motors.co.uk data** - Live car listings with actual prices, images, and details
- **Fast scraping** - Data extracted directly from search results (no detail page navigation)
- **Headless operation** - Runs silently without opening browser windows
- Two game modes:
  - **Stay at Zero**: Accumulate the lowest total price difference
  - **Streak Mode**: Guess within 10% of the actual price or game over
- Beautiful, responsive UI with clean car details
- Real car images from Motors CDN
- Direct links to original Motors.co.uk listings

## Quick Start

### Using Make (Recommended)
```bash
git clone <your-repo-url>
cd autotraderguesser
make setup    # Install dependencies and generate docs
make run      # Start the server
```

### Manual Setup
```bash
git clone <your-repo-url>
cd autotraderguesser
go mod download
go run cmd/server/main.go
```

Open your browser to `http://localhost:8080`

The game automatically fetches real car data from Motors.co.uk (takes 5-10 seconds on first load). If scraping fails, it falls back to sample data.

## Game Modes

### 🎯 Stay at Zero
- Start with a score of 0
- Each guess adds the absolute difference to your score
- Goal: Keep your cumulative score as low as possible

### 🔥 Streak Mode  
- Guess within 10% of the actual price to continue
- Each correct guess adds 1 to your streak
- One wrong guess ends the game

## Example Real Cars

From live Motors.co.uk data:
- **2018 Hyundai TUCSON** - £8,495 (63k miles, 1.6L Petrol, Manual, SUV)
- **2017 Toyota Yaris** - £7,995 (42k miles, 1.5L Hybrid, Auto, Hatchback)
- **2016 Honda Jazz** - £7,450 (42k miles, 1.3L Petrol, Auto, Hatchback)
- **2016 BMW X3** - £6,495 (103k miles, 2L Diesel, Manual, SUV)

Each car includes detailed specs: year, engine size, mileage, fuel type, gearbox, body type, real images, and links to the original Motors listing!

## Car Details Displayed

- **Make & Model** (title)
- **Year**
- **Engine Size** (1.6L, 2L, etc.)
- **Mileage** 
- **Fuel Type** (Petrol, Diesel, Hybrid, Electric)
- **Gearbox** (Manual, Auto, CVT)
- **Body Type** (SUV, Hatchback, Saloon, Estate)

## Available Commands

```bash
make run           # Start the server
make dev           # Run in development mode  
make build         # Build binary
make test          # Run tests
make test-scraper  # Test Motors scraper
make check-listings# Check loaded cars
make swagger       # Generate API docs
make clean         # Clean build files
make kill          # Kill running servers
make help          # Show all commands
```

## API Endpoints

```bash
# Check data source status
curl http://localhost:8080/api/data-source

# Get all available cars with full details
curl http://localhost:8080/api/listings

# Get a random car for the game (price hidden)
curl http://localhost:8080/api/random-listing

# Test the Motors scraper directly
curl http://localhost:8080/api/test-scraper

# View API documentation
open http://localhost:8080/swagger/
```

## Project Structure

```
autotraderguesser/
├── cmd/server/          # Main application 
├── internal/
│   ├── game/           # Game logic and handlers
│   ├── models/         # Data models (Car, GuessRequest, etc.)
│   └── scraper/        # Motors.co.uk scraping
├── static/
│   ├── css/            # Stylesheets
│   ├── js/             # Frontend JavaScript
│   └── index.html      # Main HTML file
├── docs/               # Swagger API documentation
├── Makefile           # Build and development commands
└── README.md          # This file
```

## Development

The scraper uses Go Rod for headless browser automation to extract car data from Motors.co.uk search results. It includes:

- **Variety**: Searches different UK locations for diverse car listings
- **Data extraction**: Make, model, year, price, engine size, mileage, fuel type, transmission, body type
- **Image handling**: Extracts real car images from Motors CDN
- **Error handling**: Falls back to sample data if scraping fails
- **Performance**: Fast extraction from search results page (no detail page navigation)

## Troubleshooting

**Server won't start?**
```bash
make kill  # Kill any running instances
make run   # Start fresh
```

**No cars loading?**
- Browser automation may take 5-10 seconds on first run
- Check network connection to Motors.co.uk
- Server automatically falls back to sample data if scraping fails

**Want to see what cars are loaded?**
```bash
make check-listings
```

## Tech Stack

- **Backend**: Go 1.24, Gin web framework
- **Scraping**: Go Rod (headless Chrome automation)
- **Frontend**: Vanilla JavaScript, CSS3
- **API**: RESTful with Swagger documentation
- **Build**: Make, Go modules

## License

MIT