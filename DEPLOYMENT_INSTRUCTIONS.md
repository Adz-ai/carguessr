# CarGuessr Bug Fix - Deployment Instructions

## Issue Fixed
Fixed critical bug where the Friend Challenge Guide modal gets stuck open on page load, preventing access to the main application.

## Root Cause
The bug was caused by:
1. Persisted localStorage state (challengeSessionId, creatorSessionId) triggering automatic modal display
2. Insufficient state reset on page initialization
3. Home button using `location.reload()` which preserved stale state

## Changes Made

### 1. Frontend JavaScript Changes
- **File**: `static/js/auth.js`
- Added `goHome()` function with explicit modal closing and cache-busting navigation
- Modified `DOMContentLoaded` event handler to force correct initial state on every page load
- Explicitly closes challengeGuideModal and all other modals on page load
- Ensures modeSelection is visible and gameArea is hidden on fresh load

### 2. HTML Changes
- **File**: `static/index.html`
- Updated home button: `onclick="location.reload()"` → `onclick="goHome()"`
- Updated cache-busting versions to `v=1761238978` on all script and CSS tags

### 3. Minified Files
- **File**: `static/js/auth.min.js`
- Automatically regenerated via `./minify.sh`

### 4. New Diagnostic Tool
- **File**: `clear_state.html`
- Web-based tool to clear localStorage and sessionStorage
- Accessible at: `http://your-domain/clear-state`

### 5. Server Configuration
- **File**: `cmd/server/main.go`
- Added route for clear-state diagnostic tool: `r.StaticFile("/clear-state", "./clear_state.html")`

## Deployment Steps

### Step 1: Pull Latest Changes
```bash
cd /path/to/carguessr
git pull origin main
```

### Step 2: Rebuild Server
```bash
# The server will automatically use the updated static files
# No compilation needed unless you modified Go code
```

### Step 3: Restart Server
```bash
# Stop the current server (Ctrl+C or)
pkill -f "go run cmd/server/main.go"

# Start fresh server
go run cmd/server/main.go
# OR if you use a service manager:
# systemctl restart carguessr
```

### Step 4: Clear Browser State on Server
Option A - Use the diagnostic tool:
1. Visit `http://localhost:8080/clear-state` on your server's browser
2. Click "Go to Homepage"

Option B - Manual browser cache clear:
1. Open browser DevTools (F12)
2. Go to Application tab → Storage
3. Click "Clear site data"
4. Hard refresh: Ctrl+Shift+R (Windows/Linux) or Cmd+Shift+R (Mac)

### Step 5: Verify Fix
1. Visit `http://localhost:8080`
2. Verify modeSelection screen shows correctly
3. Test clicking the home button (🏠 icon in top-left)
4. Verify no modals are stuck open
5. Test creating a friend challenge to ensure it still works

## Testing Checklist

- [ ] Server starts without errors
- [ ] Homepage loads showing mode selection (not stuck modal)
- [ ] Home button (🏠) navigates correctly
- [ ] Page reload doesn't show stuck modal
- [ ] Friend Challenge feature still works
- [ ] Challenge Guide modal can be opened and closed normally
- [ ] Clear-state tool is accessible at `/clear-state`

## If Issue Persists

If the modal is still stuck after deployment:

1. **Check server logs** for any JavaScript errors
2. **Verify cache-busting is working**:
   - Open DevTools → Network tab
   - Check if `auth.js?v=1761238978` is being loaded (not cached version)
3. **Use clear-state tool**: Visit `/clear-state` to force localStorage clear
4. **Check localStorage state**:
   ```javascript
   // In browser console:
   console.log(Object.entries(localStorage));
   // Should show current storage state
   ```
5. **Add debug logging**:
   - Open DevTools console
   - Look for any errors during page load
   - Check if DOMContentLoaded event is firing

## Cloudflare Cache (if applicable)

If using Cloudflare Tunnel:
1. Log into Cloudflare Dashboard
2. Go to Caching → Configuration
3. Click "Purge Everything"
4. Wait 30 seconds for propagation

However, static files are now configured with no-cache headers in `main.go:141-148`, so Cloudflare should not cache them.

## Rollback Plan

If you need to rollback:
```bash
git log --oneline -5  # Find previous commit hash
git checkout <previous-commit-hash>
# Restart server
```

## Version Info
- Cache-busting version: `1761238978`
- Deployment date: 2025-10-23
- Fix applied: Forced state initialization + modal closure on DOMContentLoaded
