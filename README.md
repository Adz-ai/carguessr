# CarGuessr

A car price guessing game powered by Go/Gin with a React + TypeScript frontend. Features live auction data from Bonhams and dealership inventory from Lookers, with multiple game modes, leaderboards, and a shareable challenge system.

## Features

- Multiple difficulty levels with real-time data from Bonhams auctions and Lookers dealerships
- Three game modes: Stay at Zero, Streak, and 10-car Challenge
- User authentication with persistent scoring and leaderboards
- Friend challenges with shareable invite codes
- Rate-limited public API with comprehensive security measures
- Swagger documentation in development mode

## Requirements

- Go 1.24+ (CGO enabled for SQLite)
- Node.js 18+ and npm
- Chrome/Chromium for headless scraping
- `swag` CLI for API documentation (optional)

## Quick Start

### Development
```bash
# Backend
make deps && make dev

# Frontend (separate terminal)
cd frontend && npm install && npm run dev
```

Visit `http://localhost:5173` for the development server.

### Production
```bash
# Build frontend
cd frontend && npm run build

# Run server
cd .. && go run cmd/server/main.go
```

Visit `http://localhost:8080` to play.

## Configuration

- `ADMIN_KEY`: Protects admin routes (auto-generated if not set)
- `PORT`: Override default port 8080
- Data stored in `data/` directory (auto-created)

## API Endpoints

```
GET  /api/random-enhanced-listing       # Get listing by difficulty
POST /api/check-guess                   # Submit price guess
GET  /api/leaderboard                   # View leaderboards
POST /api/challenge/start               # Start challenge session
POST /api/auth/register                 # User registration
POST /api/friends/challenges            # Create friend challenge
GET  /api/health                        # Health check
POST /api/admin/refresh-listings        # Manual cache refresh
```

Full API documentation available at `/swagger/index.html` in development mode.

## Project Structure

```
cmd/server/          # Application entry point
internal/game/       # Game logic and scraping
internal/handlers/   # Authentication and challenges
internal/database/   # SQLite schema and persistence
internal/scraper/    # Web scrapers for data sources
internal/cache/      # Cache management
frontend/            # React frontend
```

## Development Commands

```bash
make dev            # Run with debug logging
make prod           # Production mode
make build          # Compile binary
make test           # Run tests
make fmt            # Format code
```

## License

MIT
