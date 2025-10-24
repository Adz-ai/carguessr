# CarGuessr

CarGuessr is a Go/Gin powered car price guessing game with a modern React + TypeScript frontend. The backend curates 250 recent auction results from Bonhams for hard mode, dealership inventory from Lookers for easy mode, and exposes multiple game types, leaderboards, and a shareable challenge system.

## Highlights
- Live data from Bonhams auctions (hard mode) and Lookers dealerships (easy mode) cached for seven days.
- Gameplay modes: Stay at Zero, Streak, and 10-car Challenge sessions with persistent scoring.
- Registered players can save scores, join friend challenges via invite codes, and track favourite difficulty in SQLite.
- Hardened public API with rate limiting, security headers, honeypots, and admin-only refresh tooling.
- Swagger documentation (development mode), REST API consumed by the React frontend, and optional direct API usage.

## Requirements
- Go 1.24 (CGO enabled for `github.com/mattn/go-sqlite3`).
- Node.js 18+ and npm (for building the React frontend).
- Chrome or Chromium available for [rod](https://github.com/go-rod/rod) headless scraping (Linux hosts may need `--no-sandbox` prerequisites such as `libasound2`, `libnss3`, `libx11-xcb`).
- `swag` CLI (`go install github.com/swaggo/swag/cmd/swag@latest`) if you want to regenerate API docs via `make swagger` or `make setup`.

## Quick Start

### Development Mode
```bash
# Terminal 1: Start the Go backend
make deps          # Download Go modules
make swagger       # Optional: regenerate Swagger docs (requires swag)
make dev           # Run in development mode with Swagger at /swagger/

# Terminal 2: Start the React frontend
cd frontend
npm install
npm run dev        # Vite dev server on http://localhost:5173
```

Visit `http://localhost:5173` for development with hot module replacement.

### Production Mode
```bash
# Build the frontend
cd frontend
npm run build      # Generates optimized build to ../dist/

# Run the Go server (serves React app from dist/)
cd ..
go run cmd/server/main.go
```

Visit `http://localhost:8080` to play. The first request may take up to a minute while both scrapers populate caches in `data/` and seed the SQLite database at `data/carguessr.db`.

## Configuration & Data Storage
- `ADMIN_KEY` (optional) protects `/api/admin/*` routes. If unset, a temporary key is generated and printed at startup.
- `PORT` (optional) overrides the default `8080`.
- Cached listings are written to `data/bonhams_cache.json` and `data/lookers_cache.json` and refreshed automatically every seven days (or via the admin API).
- Game state, challenge templates, user accounts, and leaderboard entries live in `data/carguessr.db`.

## Gameplay Modes
- **Stay at Zero** – accumulate the smallest total difference between guesses and actual prices.
- **Streak** – keep guessing within ±10 % to extend your streak; one miss ends the run.
- **Challenge** – ten curated cars per session with GeoGuessr-style scoring. Registered users can create friend challenges that others join via a six-character code.
- Support for easy (`difficulty=easy`) and hard (`difficulty=hard`) listings across modes. Supply `X-Session-ID` to persist state from custom clients; the frontend handles this automatically.

## API Overview
Public endpoints are rate limited to ~60 req/min; admin endpoints require `X-Admin-Key`.

```
GET  /api/random-listing                # Hard mode, price hidden
GET  /api/random-enhanced-listing       # ?difficulty=easy|hard
POST /api/check-guess                   # Body: listingId, guessedPrice, gameMode, difficulty
GET  /api/leaderboard                   # Optional mode=streak|challenge, difficulty=easy|hard
POST /api/leaderboard/submit            # Submit score to leaderboard

POST /api/challenge/start               # Begin challenge session
GET  /api/challenge/:sessionId          # Retrieve current challenge state
POST /api/challenge/:sessionId/guess    # Submit guess within challenge

POST /api/auth/register | login | logout | reset-password
GET/PUT /api/auth/profile               # Requires session cookie

POST /api/friends/challenges            # Create invite-only challenge (auth required)
POST /api/friends/challenges/:code/join # Join friend challenge
GET  /api/friends/challenges/:code       # Challenge metadata & participants

GET  /api/data-source                   # Listing counts per source
GET  /api/health                        # Basic health check

POST /api/admin/refresh-listings        # Manual scrape (rate limited)
GET  /api/admin/cache-status            # Cache ages and counts
GET  /api/admin/listings                # Full listing payload (with prices)
GET  /api/admin/test-scraper            # Run Bonhams scrape diagnostic
```

Enable Swagger documentation by running in debug mode (`make dev`); browse at `/swagger/index.html`.

## Project Layout
```
cmd/server/main.go        # Gin setup, middleware wiring, route registration
internal/game/handler.go  # Game logic, challenges, scoring, scraping orchestration
internal/handlers/        # Auth, friend challenges, shared middleware contracts
internal/database/        # SQLite schema, migrations, leaderboard & user persistence
internal/scraper/         # Rod-based Bonhams & Lookers scrapers + browser helpers
internal/cache/           # JSON cache helpers for auction/dealer listings
frontend/                 # React + TypeScript frontend (Vite build system)
dist/                     # Production build output (generated by npm run build)
docs/                     # Swagger specs generated via swag
data/                     # SQLite database and JSON caches (created at runtime)
static-legacy-archived/   # Original vanilla JS implementation (archived)
```

## Development Workflow
- `make dev` – run with verbose logging and Swagger.
- `make prod` – run without Swagger (sets `GIN_MODE=release`).
- `make build` / `make build-prod` – compile binaries for local or Linux deployment.
- `make test` – execute the Go test suite.
- `make fmt` / `make lint` – format and lint Go code.
- `make kill` – stop any stray development server processes.

## Troubleshooting
- **Scraper failures** – ensure Chrome/Chromium is installed. On Linux, install headless dependencies and run with `GIN_MODE=release` for production-style behaviour. When the Bonhams scrape fails, the game falls back to a limited mock dataset.
- **Stuck caches** – use `POST /api/admin/refresh-listings` with the admin key or delete the JSON cache files before restarting.
- **Missing Swagger** – install the `swag` binary and rerun `make swagger`.

## License

MIT
