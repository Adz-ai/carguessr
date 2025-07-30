package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"autotraderguesser/internal/models"
)

type ListingCache struct {
	Data      []*models.BonhamsCar `json:"data"`
	Timestamp time.Time            `json:"timestamp"`
}

const (
	CacheFileName = "listings_cache.json"
	CacheExpiry   = 24 * time.Hour
)

// LoadFromCache loads cached listings if they exist and are not expired
func LoadFromCache() ([]*models.BonhamsCar, bool) {
	file, err := os.Open(CacheFileName)
	if err != nil {
		fmt.Println("📁 No cache file found, will scrape fresh data")
		return nil, false
	}
	defer file.Close()

	var cache ListingCache
	if err := json.NewDecoder(file).Decode(&cache); err != nil {
		fmt.Printf("❌ Error reading cache file: %v\n", err)
		return nil, false
	}

	// Check if cache is expired
	if time.Since(cache.Timestamp) > CacheExpiry {
		fmt.Printf("⏰ Cache expired (%v old), will refresh\n", time.Since(cache.Timestamp).Round(time.Minute))
		return nil, false
	}

	fmt.Printf("✅ Loaded %d cars from cache (updated %v ago)\n",
		len(cache.Data), time.Since(cache.Timestamp).Round(time.Minute))
	return cache.Data, true
}

// SaveToCache saves listings to cache file
func SaveToCache(listings []*models.BonhamsCar) error {
	cache := ListingCache{
		Data:      listings,
		Timestamp: time.Now(),
	}

	file, err := os.Create(CacheFileName)
	if err != nil {
		return fmt.Errorf("failed to create cache file: %v", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(cache); err != nil {
		return fmt.Errorf("failed to encode cache: %v", err)
	}

	fmt.Printf("💾 Cached %d listings to %s\n", len(listings), CacheFileName)
	return nil
}

// IsCacheExpired checks if the cache is expired without loading it
func IsCacheExpired() bool {
	file, err := os.Open(CacheFileName)
	if err != nil {
		return true // No cache file means expired
	}
	defer file.Close()

	var cache ListingCache
	if err := json.NewDecoder(file).Decode(&cache); err != nil {
		return true // Corrupted cache means expired
	}

	return time.Since(cache.Timestamp) > CacheExpiry
}

// GetCacheAge returns the age of the cache
func GetCacheAge() (time.Duration, error) {
	file, err := os.Open(CacheFileName)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var cache ListingCache
	if err := json.NewDecoder(file).Decode(&cache); err != nil {
		return 0, err
	}

	return time.Since(cache.Timestamp), nil
}
