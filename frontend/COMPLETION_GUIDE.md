# CarGuessr React Migration - Completion Guide

## Current Status

✅ **Completed:**
- Project setup (Vite + React + TypeScript)
- Type definitions
- API client (full backend integration)
- State management stores (Zustand)
- Hooks and utilities
- CSS migration
- Main App structure
- TopNav and Footer components
- ModeSelection component (complete)
- Stub components for remaining features

## What's Left to Complete

### 1. Game Components (HIGH PRIORITY)

#### GameArea.tsx
Replace the stub with full implementation from `static/js/game.js`:
- Load car listing on mount (use `apiClient.getRandomCar()` or `apiClient.startChallenge()`)
- Display car using CarDisplay component
- Show PriceInput component
- Handle guess submission
- Show result modals

**Key Functions to Port:**
- `loadNextCar()` → useEffect hook
- `submitGuess()` → event handler
- `displayResult()` → state + modal display
- Challenge mode logic (`loadChallengeAuto`, `submitChallengeGuess`)

#### Create These Components:

**CarDisplay.tsx**
```typescript
// Display car images and details
// Port from `displayCar()` and `setupImageGallery()` in game.js
interface CarDisplayProps {
  car: CarListing;
}
```

**PriceInput.tsx**
```typescript
// Price input with slider
// Port slider logic from game.js (priceToSlider, sliderToPrice, sync functions)
interface PriceInputProps {
  onSubmit: (price: number) => void;
}
```

**ResultModal.tsx**
```typescript
// Show individual guess results
// Port from displayResult() in game.js
interface ResultModalProps {
  result: GuessResult;
  onNext: () => void;
  onEnd: () => void;
}
```

**GameOverModal.tsx**
```typescript
// Final game over screen
// Port from game.js gameOver modal logic
interface GameOverModalProps {
  score: number;
  mode: GameMode;
  onSubmitScore: () => void;
  onPlayAgain: () => void;
}
```

### 2. Authentication Components

#### Auth Modals to Create:

**LoginModal.tsx**
```typescript
// Port from static/js/auth.js handleLogin()
- Form with username/password
- Call apiClient.login()
- Use useAuthStore().login() on success
- Show toast on error
```

**RegisterModal.tsx**
```typescript
// Port from static/js/auth.js handleRegister()
- Form with all registration fields
- Call apiClient.register()
- Handle success/error
```

**PasswordResetModal.tsx**
```typescript
// Port from auth.js password reset functions
- Two-step process (verify user, then reset)
- Call apiClient.getSecurityQuestion() and apiClient.resetPassword()
```

**ProfileModal.tsx**
```typescript
// Port from auth.js showProfile()
- Display user info
- Show game statistics
- Call apiClient.getProfile() to refresh stats
```

**AuthModals.tsx** (container)
```typescript
// Listen for custom events and show appropriate modals
useEffect(() => {
  const handlers = {
    showLoginModal: () => setActiveModal('login'),
    showRegisterModal: () => setActiveModal('register'),
    // ... etc
  };

  Object.entries(handlers).forEach(([event, handler]) => {
    window.addEventListener(event, handler);
  });

  return () => {
    Object.entries(handlers).forEach(([event, handler]) => {
      window.removeEventListener(event, handler);
    });
  };
}, []);
```

### 3. Leaderboard Components

#### LeaderboardModal.tsx (Full Implementation)
```typescript
// Port from game.js leaderboard functions
- Tabs for game modes (challenge/streak)
- Difficulty filter (easy/hard)
- Fetch and display entries using apiClient.getLeaderboard()
- Highlight user's entry
```

#### NameInputModal.tsx
```typescript
// Port from showNameInputModal() in game.js
interface NameInputModalProps {
  onSubmit: (name: string) => void;
  onSkip: () => void;
}
// Auto-submit for logged-in users
// Show name input for guests
```

### 4. Challenge Components

#### CreateChallengeModal.tsx
```typescript
// Port from auth.js handleCreateChallenge()
- Form: title, difficulty, max participants
- Call apiClient.createChallenge()
- Show challenge code on success
```

#### JoinChallengeModal.tsx
```typescript
// Port from auth.js handleJoinChallenge()
- Input for challenge code
- Call apiClient.joinChallenge()
- Start challenge on success
```

#### MyChallengesModal.tsx
```typescript
// Port from auth.js loadMyChallenges()
- Two tabs: Created / Participating
- Fetch using apiClient.getMyChallenges()
- Display challenge cards with actions
```

#### ChallengeLeaderboardModal.tsx
```typescript
// Port from auth.js showChallengeLeaderboard()
- Fetch using apiClient.getChallengeLeaderboard()
- Display participants with scores
- Refresh button
```

#### ChallengeGuideModal.tsx
```typescript
// Port from HTML (#challengeGuideModal)
- Static content explaining how challenges work
- Just JSX, no complex logic
```

### 5. Go Server Configuration

Update your Go server to serve the Vite build:

**Example Go server changes:**

```go
// In your main.go or server setup

// Serve static files from dist folder
fs := http.FileServer(http.Dir("./dist"))

// API routes (existing)
http.HandleFunc("/api/", yourAPIHandler)

// Serve React app for all other routes (SPA)
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    // If requesting a file that exists, serve it
    path := "./dist" + r.URL.Path
    if _, err := os.Stat(path); err == nil {
        fs.ServeHTTP(w, r)
        return
    }

    // Otherwise serve index.html (SPA routing)
    http.ServeFile(w, r, "./dist/index.html")
})
```

**Alternative using a router like gorilla/mux:**

```go
r := mux.NewRouter()

// API routes
api := r.PathPrefix("/api").Subrouter()
api.HandleFunc("/random-listing", getRandomListing)
// ... other API routes

// Serve static files
r.PathPrefix("/assets").Handler(http.FileServer(http.Dir("./dist")))
r.PathPrefix("/css").Handler(http.FileServer(http.Dir("./dist")))
r.PathPrefix("/favicon_io").Handler(http.FileServer(http.Dir("./dist")))
r.PathPrefix("/images").Handler(http.FileServer(http.Dir("./dist")))

// Catch-all: serve index.html for client-side routing
r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "./dist/index.html")
})
```

### 6. Update index.html

Update `frontend/index.html` to include proper meta tags (copy from `static/index.html`):

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/png" href="/favicon_io/favicon-32x32.png" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />

    <!-- Copy SEO meta tags from static/index.html -->
    <meta name="description" content="..." />
    <!-- ... etc -->

    <title>CarGuessr - Test Your Car Valuation Skills</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

## Testing Checklist

### Game Modes
- [ ] Challenge Mode (Easy)
- [ ] Challenge Mode (Hard)
- [ ] Streak Mode (Easy)
- [ ] Streak Mode (Hard)
- [ ] Stay at Zero Mode (Easy)
- [ ] Stay at Zero Mode (Hard)

### Authentication
- [ ] Register new account
- [ ] Login
- [ ] Logout
- [ ] Password reset
- [ ] View profile
- [ ] Auto-login on page refresh

### Leaderboards
- [ ] View leaderboard (all modes)
- [ ] Submit score (guest)
- [ ] Submit score (logged in - auto)
- [ ] Difficulty filtering
- [ ] Ranking display

### Friend Challenges
- [ ] Create challenge
- [ ] Join challenge via code
- [ ] View my challenges
- [ ] Resume challenge
- [ ] Challenge leaderboard
- [ ] Challenge completion

### UI/UX
- [ ] Mobile responsiveness
- [ ] Toast notifications
- [ ] Modal interactions
- [ ] Image loading (Easy mode warning)
- [ ] Slider/input sync
- [ ] Navigation (home button)

## Development Workflow

1. **Start backend**: `./server` (or your Go server command)
2. **Start frontend**: `cd frontend && npm run dev`
3. **Access app**: http://localhost:5173
4. **Build for production**: `npm run build`
5. **Test production**: Copy `dist/*` to Go server location, start Go server

## Quick Reference

### Accessing Stores in Components
```typescript
import { useGameStore } from '../stores/gameStore';
import { useAuthStore } from '../stores/authStore';

const difficulty = useGameStore(state => state.difficulty);
const setDifficulty = useGameStore(state => state.setDifficulty);
const user = useAuthStore(state => state.user);
```

### Making API Calls
```typescript
import { apiClient } from '../api/client';

// Example
const car = await apiClient.getRandomCar(difficulty, sessionId);
const result = await apiClient.submitGuess({...});
```

### Showing Toasts
```typescript
import { showToast } from '../utils/toast';

showToast('Success!', 'success');
showToast('Error occurred', 'error');
showToast('Info message', 'info');
```

### Custom Events (for modal triggers)
```typescript
// Dispatch
window.dispatchEvent(new CustomEvent('showLoginModal'));

// Listen
useEffect(() => {
  const handler = () => setShowModal(true);
  window.addEventListener('showLoginModal', handler);
  return () => window.removeEventListener('showLoginModal', handler);
}, []);
```

## Tips

1. **Port incrementally**: Complete one component at a time and test
2. **Reuse existing CSS**: The styles are already imported
3. **Match original behavior**: Use the original JS as reference for game logic
4. **Type safety**: Let TypeScript guide you - if types don't match, fix the types
5. **Test frequently**: Test each feature as you build it

## Need Help?

- Original game logic: `static/js/game.js`
- Original auth logic: `static/js/auth.js`
- Original HTML: `static/index.html`
- API client already has all endpoints implemented
- Type definitions are complete in `src/types/index.ts`

Good luck! 🚀
