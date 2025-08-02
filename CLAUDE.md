# 🤖 CLAUDE.md - Future Development Tasks

This file contains planned features and improvements for future development sessions.

## 🔄 **High Priority: Daily Listing Refresh System**

**Problem:** Listings only refresh when the application is restarted, leading to repetitive car data for users.

**Required Features:**
1. **Automatic Daily Refresh**
   - Cron job or internal scheduler to refresh listings every 24 hours
   - Configurable refresh interval (daily/12h/6h)
   - Graceful refresh without service downtime

2. **Variety Enforcement**
   - Algorithm to ensure different cars are selected each refresh
   - Track previously shown cars to avoid repetition
   - Rotate through different postcodes/locations for variety

3. **Implementation Options:**
   ```go
   // Option A: Internal Go scheduler
   func startListingRefreshScheduler() {
       ticker := time.NewTicker(24 * time.Hour)
       go func() {
           for range ticker.C {
               refreshListings()
           }
       }()
   }
   
   // Option B: System cron job
   // 0 2 * * * curl -X POST http://localhost:8080/api/refresh-listings
   ```

4. **API Endpoints Needed:**
   - `POST /api/refresh-listings` - Manual refresh trigger
   - `GET /api/refresh-status` - Check last refresh time
   - `GET /api/listings-stats` - Show variety metrics

5. **Database/Storage:**
   - Track listing refresh timestamps
   - Store previously shown car IDs to avoid duplicates
   - Maintain variety statistics (makes/models distribution)

## 🎯 **Medium Priority Features**

### **User Experience Improvements**
- [ ] Loading indicators during car fetching
- [ ] "New cars today!" notification after refresh
- [ ] Favorite/bookmark specific cars
- [ ] User statistics (games played, average accuracy)

### **Performance Optimizations**
- [ ] Cache scraped data in Redis/memory
- [ ] Lazy loading for car images
- [ ] CDN integration for faster image delivery
- [ ] Connection pooling for database queries

### **Admin Features**
- [ ] Admin dashboard for monitoring
- [ ] Manual blacklist for problematic listings
- [ ] Scraping health monitoring
- [ ] Error rate tracking and alerts

## 🔒 **Security Implementation**

### **Rate Limiting**
- **General API**: 60 requests per minute (1 req/sec with burst of 60)
- **Admin endpoints**: 2 requests per minute (strict limiting)
- **Per-IP tracking**: Automatic cleanup of old visitors
- **Rate limit headers**: Proper HTTP 429 responses

### **Admin Endpoint Protection**
- **Authentication**: Admin key required via header (`X-Admin-Key`) or query parameter
- **Refresh protection**: 30-minute cooldown between manual refreshes
- **Secured endpoints**: `/api/admin/refresh-listings`, `/api/admin/test-scraper`, etc.
- **Environment-based**: Admin key from `ADMIN_KEY` env var

### **Input Validation & Sanitization**
- **Price limits**: Maximum £10,000,000 to prevent unrealistic values
- **Listing ID format**: Alphanumeric, hyphens, underscores only (max 100 chars)
- **Session ID format**: Exactly 16 alphanumeric characters
- **JSON binding**: Gin's built-in validation with custom rules

### **Security Headers**
- **X-Frame-Options**: DENY (prevents clickjacking)
- **X-Content-Type-Options**: nosniff (prevents MIME sniffing)
- **X-XSS-Protection**: Enabled with blocking mode
- **Content-Security-Policy**: Restrictive policy for scripts/styles
- **Referrer-Policy**: strict-origin-when-cross-origin

### **API Structure**
- **Public endpoints**: `/api/*` with general rate limiting
- **Admin endpoints**: `/api/admin/*` with authentication + strict rate limiting
- **Protected operations**: Refresh, test scraper, admin listings access

## 🔧 **Technical Improvements**

### **Error Handling**
- [ ] Retry mechanism for failed scrapes
- [ ] Graceful degradation when Motors.co.uk is down
- [ ] Better error messages for users
- [ ] Logging improvements with structured logging

### **Configuration**
- [ ] Environment-based configuration
- [ ] Hot-reload configuration changes
- [ ] Feature flags for experimental features
- [ ] A/B testing framework

### **Security**
- [ ] Rate limiting per IP
- [ ] CAPTCHA for excessive usage
- [ ] API key authentication for admin endpoints
- [ ] Input validation improvements

## 🚀 **Future Enhancements**

### **Multi-Source Support**
- [ ] AutoTrader UK integration
- [ ] Cazoo/Cinch integration
- [ ] Price comparison across platforms
- [ ] Historical price tracking

### **Game Modes**
- [ ] Multiplayer competitions
- [ ] Daily challenges
- [ ] Leaderboards with user accounts
- [ ] Achievement system

### **Mobile App**
- [ ] React Native mobile app
- [ ] Push notifications for daily challenges
- [ ] Offline mode with cached data
- [ ] Share results on social media

## 📝 **Implementation Notes**

### **Daily Refresh Priority Implementation:**

1. **Phase 1: Basic Scheduler**
   ```go
   // Add to main.go
   go startListingRefreshScheduler(gameHandler)
   
   // Add endpoint
   api.POST("/refresh-listings", gameHandler.RefreshListings)
   ```

2. **Phase 2: Variety Algorithm**
   ```go
   type ListingTracker struct {
       LastShown    map[string]time.Time
       ShowCount    map[string]int
       LastRefresh  time.Time
   }
   ```

3. **Phase 3: User-Facing Features**
   - UI indicator showing "Updated X hours ago"
   - Manual refresh button for admin users
   - Variety statistics in footer

### **Hosting Considerations**
- ✅ **Current**: Ubuntu VM at home (working well!)
- Consider systemd service with auto-restart
- Log rotation for long-running processes
- Backup strategy for user data

### **Development Environment**
- Local development continues on Mac (works perfectly)
- Production deployment on Ubuntu VM (bot detection solved)
- Docker available as backup option with Windows-like fingerprinting

## 🎉 **Completed Items**
- ✅ Bot detection issues resolved (Ubuntu VM solution)
- ✅ Slider functionality fixed for 0-1M range
- ✅ Detail page enhancement working
- ✅ Docker deployment options created
- ✅ Cloudflare Tunnel integration
- ✅ Cross-platform architecture support
- ✅ **Session 30/07/2025: Major Improvements**
  - Fixed Bonhams scraper to only include SOLD items (filtering out "Bid to" listings)
  - Added home button for easy navigation (top-left with modern glassmorphism design)
  - Resolved Cloudflare caching issues for challenge mode:
    - Added cache-control headers to prevent static file caching
    - Added version query parameters to force cache invalidation
    - Fixed CORS headers to include X-Session-ID
    - Updated validation to accept "challenge" as valid game mode
  - Added comprehensive logging for debugging production issues

## 📝 **Session Summary - 30/07/2025**

### **1. Bonhams Scraper Filtering Fix**
- **Issue**: Scraper was including unsold items (those with "Bid to" status)
- **Solution**: Added filtering at the listing card level to only include items marked as "Sold for" or "Hammer price"
- **Result**: Game now only shows cars that actually sold at auction

### **2. Home Button Addition**
- **Feature**: Added a fixed home button in top-left corner
- **Design**: Modern glassmorphism effect with backdrop blur
- **Function**: Simple page reload to return to game mode selection

### **3. Cloudflare Deployment Issues**
- **Problem**: Challenge mode failing with 400 errors when behind Cloudflare
- **Root Causes**:
  1. Missing "challenge" in game mode validation
  2. Cloudflare caching old JavaScript files
  3. Missing X-Session-ID in CORS allowed headers
- **Solutions**:
  1. Updated model validation to include "challenge" mode
  2. Added cache-control headers for static files
  3. Added version parameters to JS/CSS files
  4. Fixed CORS configuration
  5. Added detailed request logging for debugging

### **Technical Details Added**
- Request body logging for debugging JSON parsing issues
- Middleware to prevent Cloudflare from caching static assets
- Improved error messages with more context

---

**Next Session Priority:** Implement daily listing refresh system with variety enforcement to keep the game fresh and engaging!