package middleware

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter stores rate limiters for each IP
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     r,
		burst:    b,
	}

	// Clean up old entries every minute
	go rl.cleanupVisitors()

	return rl
}

// GetLimiter returns the rate limiter for the given IP
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// cleanupVisitors removes old entries from the visitors map
func (rl *RateLimiter) cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		l := limiter.GetLimiter(ip)

		if !l.Allow() {
			log.Printf("Rate limit exceeded for %s", ip)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too many requests",
				"message": "Please slow down your requests",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RefreshProtectionMiddleware protects refresh endpoints
func RefreshProtectionMiddleware() gin.HandlerFunc {
	var (
		lastRefresh time.Time
		mu          sync.Mutex
	)

	return func(c *gin.Context) {
		mu.Lock()
		defer mu.Unlock()

		// Only allow refresh every 30 minutes
		if time.Since(lastRefresh) < 30*time.Minute {
			remaining := 30*time.Minute - time.Since(lastRefresh)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Refresh too frequent",
				"message": fmt.Sprintf("Please wait %d minutes before refreshing again", int(remaining.Minutes())),
			})
			c.Abort()
			return
		}

		lastRefresh = time.Now()
		c.Next()
	}
}

// SecurityHeaders adds security headers to responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy - stricter in production, relaxed in development
		csp := buildCSPPolicy()
		c.Header("Content-Security-Policy", csp)

		// Strict Transport Security (HTTPS only)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Hide server information
		c.Header("Server", "")

		// Prevent caching of sensitive responses
		if strings.Contains(c.Request.URL.Path, "/api/admin/") {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}

		c.Next()
	}
}

// AdminKeyMiddleware protects admin endpoints with a simple key
func AdminKeyMiddleware(adminKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-Admin-Key")
		if key == "" {
			key = c.Query("admin_key")
		}

		if key != adminKey {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Admin access required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SecurityScanDetection logs suspicious requests for fail2ban
func SecurityScanDetection() gin.HandlerFunc {
	suspiciousPaths := []string{
		".env", ".git", ".DS_Store", "wp-admin", "admin", "phpmyadmin",
		".htaccess", "config.php", "wp-config.php", ".ssh", "id_rsa",
		"backup", ".bak", ".sql", "database", "credentials",
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		ip := c.ClientIP()

		// Check for suspicious paths
		for _, suspicious := range suspiciousPaths {
			if strings.Contains(path, suspicious) {
				log.Printf("Security scan attempt from %s: %s %s", ip, c.Request.Method, path)
				break
			}
		}

		// Check for SQL injection attempts
		queryString := c.Request.URL.RawQuery
		if strings.Contains(strings.ToLower(queryString), "union") ||
			strings.Contains(strings.ToLower(queryString), "select") ||
			strings.Contains(strings.ToLower(queryString), "drop") ||
			strings.Contains(strings.ToLower(queryString), "insert") {
			log.Printf("SQL injection attempt from %s: %s", ip, queryString)
		}

		c.Next()
	}
}

// HTTPMethodFilter restricts allowed HTTP methods
func HTTPMethodFilter(allowedMethods []string) gin.HandlerFunc {
	allowed := make(map[string]bool)
	for _, method := range allowedMethods {
		allowed[method] = true
	}

	return func(c *gin.Context) {
		if !allowed[c.Request.Method] {
			log.Printf("Blocked HTTP method %s from %s", c.Request.Method, c.ClientIP())
			c.JSON(http.StatusMethodNotAllowed, gin.H{
				"error": "Method not allowed",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// UserAgentFilter blocks requests with suspicious or missing user agents
func UserAgentFilter() gin.HandlerFunc {
	suspiciousAgents := []string{
		"sqlmap", "nikto", "nmap", "masscan", "zap", "gobuster",
		"dirb", "dirbuster", "burp", "w3af", "havij", "libwww",
	}

	return func(c *gin.Context) {
		userAgent := strings.ToLower(c.GetHeader("User-Agent"))
		ip := c.ClientIP()

		// Block empty user agents (likely bots)
		if userAgent == "" {
			log.Printf("Blocked empty user agent from %s", ip)
			c.JSON(http.StatusForbidden, gin.H{"error": "User agent required"})
			c.Abort()
			return
		}

		// Block known attack tools
		for _, suspicious := range suspiciousAgents {
			if strings.Contains(userAgent, suspicious) {
				log.Printf("Blocked suspicious user agent from %s: %s", ip, userAgent)
				c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// HoneypotEndpoints creates fake endpoints to detect automated scanners
func HoneypotEndpoints() gin.HandlerFunc {
	honeypots := []string{
		"/admin.php", "/wp-login.php", "/login.php", "/admin/login",
		"/administrator", "/admin/admin", "/user/login", "/auth/login",
		"/xmlrpc.php", "/wp-admin/admin-ajax.php", "/api/v1/login",
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		for _, honeypot := range honeypots {
			if path == honeypot {
				ip := c.ClientIP()
				log.Printf("Honeypot triggered by %s: %s", ip, path)

				// Slow down the response to waste attacker's time
				time.Sleep(5 * time.Second)

				c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// buildCSPPolicy creates a Content Security Policy header
// In production: strict policy without unsafe-inline/unsafe-eval
// In development: relaxed policy for easier debugging
func buildCSPPolicy() string {
	isDevelopment := os.Getenv("GIN_MODE") != "release"

	if isDevelopment {
		// Development mode: Allow inline scripts/styles for easier debugging
		// Still includes Cloudflare Insights and webpack dev server support
		return "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://static.cloudflareinsights.com; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"connect-src 'self' webpack:;"
	}

	// Production mode: Strict CSP without unsafe-inline/unsafe-eval
	// For maximum security, frontend should use nonces or hashes for inline scripts
	return "default-src 'self'; " +
		"script-src 'self' https://static.cloudflareinsights.com; " +
		"style-src 'self'; " +
		"img-src 'self' data: https:; " +
		"connect-src 'self'; " +
		"font-src 'self'; " +
		"object-src 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'; " +
		"frame-ancestors 'none'; " +
		"upgrade-insecure-requests;"
}
