package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"autotraderguesser/internal/models"
)

type BonhamsCache struct {
	Data      []*models.BonhamsCar `json:"data"`
	Timestamp time.Time            `json:"timestamp"`
}

type LookersCache struct {
	Data      []*models.LookersCar `json:"data"`
	Timestamp time.Time            `json:"timestamp"`
}

const (
	BonhamsCacheFileName = "data/bonhams_cache.json"
	LookersCacheFileName = "data/lookers_cache.json"
	BonhamsCacheExpiry   = 7 * 24 * time.Hour // 7 days - auction data changes less frequently
	LookersCacheExpiry   = 7 * 24 * time.Hour // 7 days - dealership inventory also doesn't need frequent updates
)

// LoadBonhamsFromCache loads cached Bonhams listings if they exist and are not expired
func LoadBonhamsFromCache() ([]*models.BonhamsCar, bool) {
	file, err := os.Open(BonhamsCacheFileName)
	if err != nil {
		fmt.Println("📁 No Bonhams cache file found, will scrape fresh data")
		return nil, false
	}
	defer file.Close()

	var cache BonhamsCache
	if err := json.NewDecoder(file).Decode(&cache); err != nil {
		fmt.Printf("❌ Error reading Bonhams cache file: %v\n", err)
		return nil, false
	}

	// Check if cache is expired
	if time.Since(cache.Timestamp) > BonhamsCacheExpiry {
		fmt.Printf("⏰ Bonhams cache expired (%.1f days old), will refresh\n", time.Since(cache.Timestamp).Hours()/24)
		return nil, false
	}

	daysRemaining := (BonhamsCacheExpiry - time.Since(cache.Timestamp)).Hours() / 24
	fmt.Printf("✅ Loaded %d Bonhams cars from cache (updated %.1f days ago, %.1f days until refresh)\n",
		len(cache.Data), time.Since(cache.Timestamp).Hours()/24, daysRemaining)
	return cache.Data, true
}

// LoadLookersFromCache loads cached Lookers listings if they exist and are not expired
func LoadLookersFromCache() ([]*models.LookersCar, bool) {
	file, err := os.Open(LookersCacheFileName)
	if err != nil {
		fmt.Println("📁 No Lookers cache file found, will scrape fresh data")
		return nil, false
	}
	defer file.Close()

	var cache LookersCache
	if err := json.NewDecoder(file).Decode(&cache); err != nil {
		fmt.Printf("❌ Error reading Lookers cache file: %v\n", err)
		return nil, false
	}

	// Check if cache is expired
	if time.Since(cache.Timestamp) > LookersCacheExpiry {
		fmt.Printf("⏰ Lookers cache expired (%.1f days old), will refresh\n", time.Since(cache.Timestamp).Hours()/24)
		return nil, false
	}

	daysRemaining := (LookersCacheExpiry - time.Since(cache.Timestamp)).Hours() / 24
	fmt.Printf("✅ Loaded %d Lookers cars from cache (updated %.1f days ago, %.1f days until refresh)\n",
		len(cache.Data), time.Since(cache.Timestamp).Hours()/24, daysRemaining)
	return cache.Data, true
}

// LoadFromCache loads cached Bonhams listings (backward compatibility)
func LoadFromCache() ([]*models.BonhamsCar, bool) {
	return LoadBonhamsFromCache()
}

// SaveBonhamsToCache saves Bonhams listings to cache file
func SaveBonhamsToCache(listings []*models.BonhamsCar) error {
	cache := BonhamsCache{
		Data:      listings,
		Timestamp: time.Now(),
	}

	file, err := os.Create(BonhamsCacheFileName)
	if err != nil {
		return fmt.Errorf("failed to create Bonhams cache file: %v", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(cache); err != nil {
		return fmt.Errorf("failed to encode Bonhams cache: %v", err)
	}

	fmt.Printf("💾 Cached %d Bonhams listings to %s\n", len(listings), BonhamsCacheFileName)
	return nil
}

// SaveLookersToCache saves Lookers listings to cache file
func SaveLookersToCache(listings []*models.LookersCar) error {
	cache := LookersCache{
		Data:      listings,
		Timestamp: time.Now(),
	}

	file, err := os.Create(LookersCacheFileName)
	if err != nil {
		return fmt.Errorf("failed to create Lookers cache file: %v", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(cache); err != nil {
		return fmt.Errorf("failed to encode Lookers cache: %v", err)
	}

	fmt.Printf("💾 Cached %d Lookers listings to %s\n", len(listings), LookersCacheFileName)
	return nil
}

// SaveToCache saves Bonhams listings to cache (backward compatibility)
func SaveToCache(listings []*models.BonhamsCar) error {
	return SaveBonhamsToCache(listings)
}

// IsBonhamsCacheExpired checks if the Bonhams cache is expired without loading it
func IsBonhamsCacheExpired() bool {
	file, err := os.Open(BonhamsCacheFileName)
	if err != nil {
		return true // No cache file means expired
	}
	defer file.Close()

	var cache BonhamsCache
	if err := json.NewDecoder(file).Decode(&cache); err != nil {
		return true // Corrupted cache means expired
	}

	return time.Since(cache.Timestamp) > BonhamsCacheExpiry
}

// IsLookersCacheExpired checks if the Lookers cache is expired without loading it
func IsLookersCacheExpired() bool {
	file, err := os.Open(LookersCacheFileName)
	if err != nil {
		return true // No cache file means expired
	}
	defer file.Close()

	var cache LookersCache
	if err := json.NewDecoder(file).Decode(&cache); err != nil {
		return true // Corrupted cache means expired
	}

	return time.Since(cache.Timestamp) > LookersCacheExpiry
}

// IsCacheExpired checks if the Bonhams cache is expired (backward compatibility)
func IsCacheExpired() bool {
	return IsBonhamsCacheExpired()
}

// GetBonhamsCacheAge returns the age of the Bonhams cache
func GetBonhamsCacheAge() (time.Duration, error) {
	file, err := os.Open(BonhamsCacheFileName)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var cache BonhamsCache
	if err := json.NewDecoder(file).Decode(&cache); err != nil {
		return 0, err
	}

	return time.Since(cache.Timestamp), nil
}

// GetLookersCacheAge returns the age of the Lookers cache
func GetLookersCacheAge() (time.Duration, error) {
	file, err := os.Open(LookersCacheFileName)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var cache LookersCache
	if err := json.NewDecoder(file).Decode(&cache); err != nil {
		return 0, err
	}

	return time.Since(cache.Timestamp), nil
}

// GetCacheAge returns the age of the Bonhams cache (backward compatibility)
func GetCacheAge() (time.Duration, error) {
	return GetBonhamsCacheAge()
}

// LoadBonhamsFromCacheIgnoreExpiry loads cached Bonhams listings regardless of expiry (for fallback)
func LoadBonhamsFromCacheIgnoreExpiry() ([]*models.BonhamsCar, error) {
	file, err := os.Open(BonhamsCacheFileName)
	if err != nil {
		return nil, fmt.Errorf("no cache file found: %v", err)
	}
	defer file.Close()

	var cache BonhamsCache
	if err := json.NewDecoder(file).Decode(&cache); err != nil {
		return nil, fmt.Errorf("error reading cache file: %v", err)
	}

	age := time.Since(cache.Timestamp)
	fmt.Printf("📦 Loaded %d Bonhams cars from expired cache (%.1f days old)\n",
		len(cache.Data), age.Hours()/24)
	return cache.Data, nil
}

// LoadLookersFromCacheIgnoreExpiry loads cached Lookers listings regardless of expiry (for fallback)
func LoadLookersFromCacheIgnoreExpiry() ([]*models.LookersCar, error) {
	file, err := os.Open(LookersCacheFileName)
	if err != nil {
		return nil, fmt.Errorf("no cache file found: %v", err)
	}
	defer file.Close()

	var cache LookersCache
	if err := json.NewDecoder(file).Decode(&cache); err != nil {
		return nil, fmt.Errorf("error reading cache file: %v", err)
	}

	age := time.Since(cache.Timestamp)
	fmt.Printf("📦 Loaded %d Lookers cars from expired cache (%.1f days old)\n",
		len(cache.Data), age.Hours()/24)
	return cache.Data, nil
}

// BumpBonhamsExpiry updates the cache timestamp to extend its life by 7 days
func BumpBonhamsExpiry() error {
	// Load existing cache data (ignoring expiry)
	listings, err := LoadBonhamsFromCacheIgnoreExpiry()
	if err != nil {
		return fmt.Errorf("cannot bump expiry: %v", err)
	}

	// Resave with new timestamp
	if err := SaveBonhamsToCache(listings); err != nil {
		return fmt.Errorf("failed to bump Bonhams expiry: %v", err)
	}

	fmt.Printf("⏰ Bumped Bonhams cache expiry - valid for another 7 days\n")
	return nil
}

// BumpLookersExpiry updates the cache timestamp to extend its life by 7 days
func BumpLookersExpiry() error {
	// Load existing cache data (ignoring expiry)
	listings, err := LoadLookersFromCacheIgnoreExpiry()
	if err != nil {
		return fmt.Errorf("cannot bump expiry: %v", err)
	}

	// Resave with new timestamp
	if err := SaveLookersToCache(listings); err != nil {
		return fmt.Errorf("failed to bump Lookers expiry: %v", err)
	}

	fmt.Printf("⏰ Bumped Lookers cache expiry - valid for another 7 days\n")
	return nil
}
