package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func performRequest(r http.Handler, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestRateLimitMiddleware(t *testing.T) {
	limiter := NewRateLimiter(rate.Limit(1), 1)
	r := gin.New()
	r.Use(RateLimitMiddleware(limiter))
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	rec1 := performRequest(r, http.MethodGet, "/", map[string]string{"User-Agent": "test"})
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec1.Code)
	}

	rec2 := performRequest(r, http.MethodGet, "/", map[string]string{"User-Agent": "test"})
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on rapid second request, got %d", rec2.Code)
	}
}

func TestRefreshProtectionMiddleware(t *testing.T) {
	r := gin.New()
	r.Use(RefreshProtectionMiddleware())
	r.GET("/refresh", func(c *gin.Context) { c.String(http.StatusOK, "refreshed") })

	rec1 := performRequest(r, http.MethodGet, "/refresh", map[string]string{"User-Agent": "test"})
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected first refresh to succeed, got %d", rec1.Code)
	}

	rec2 := performRequest(r, http.MethodGet, "/refresh", map[string]string{"User-Agent": "test"})
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second refresh to be rate limited, got %d", rec2.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "headers") })

	rec := performRequest(r, http.MethodGet, "/", map[string]string{"User-Agent": "test"})
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	required := []string{"X-Frame-Options", "X-Content-Type-Options", "X-XSS-Protection", "Referrer-Policy", "Content-Security-Policy", "Strict-Transport-Security"}
	for _, header := range required {
		if rec.Header().Get(header) == "" {
			t.Fatalf("expected header %s to be set", header)
		}
	}
}

func TestSecurityScanDetection(t *testing.T) {
	r := gin.New()
	r.Use(SecurityScanDetection())
	r.GET("/.env", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	rec := performRequest(r, http.MethodGet, "/.env?query=select", map[string]string{"User-Agent": "test"})
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status for suspicious path: %d", rec.Code)
	}
}

func TestHTTPMethodFilter(t *testing.T) {
	r := gin.New()
	r.Use(HTTPMethodFilter([]string{http.MethodGet}))
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	rec := performRequest(r, http.MethodPost, "/", map[string]string{"User-Agent": "test"})
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for blocked method, got %d", rec.Code)
	}
}

func TestUserAgentFilter(t *testing.T) {
	r := gin.New()
	r.Use(UserAgentFilter())
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	recEmpty := performRequest(r, http.MethodGet, "/", map[string]string{})
	if recEmpty.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden for empty UA, got %d", recEmpty.Code)
	}

	recSuspicious := performRequest(r, http.MethodGet, "/", map[string]string{"User-Agent": "sqlmap"})
	if recSuspicious.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden for suspicious UA, got %d", recSuspicious.Code)
	}

	recOk := performRequest(r, http.MethodGet, "/", map[string]string{"User-Agent": "Mozilla"})
	if recOk.Code != http.StatusOK {
		t.Fatalf("expected success for benign UA, got %d", recOk.Code)
	}
}

func TestHoneypotEndpoints(t *testing.T) {
	r := gin.New()
	r.Use(HoneypotEndpoints())
	r.GET("/admin.php", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	start := time.Now()
	rec := performRequest(r, http.MethodGet, "/admin.php", map[string]string{"User-Agent": "Mozilla"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected honeypot to return 404, got %d", rec.Code)
	}
	if time.Since(start) < 5*time.Second {
		t.Fatalf("expected honeypot to delay response")
	}
}
