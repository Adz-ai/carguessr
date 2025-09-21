package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"autotraderguesser/internal/models"
)

func TestMain(m *testing.M) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	root := filepath.Join(cwd, "..", "..")
	if err := os.Chdir(root); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

type restoreFn func()

func backupFile(t *testing.T, path string) restoreFn {
	t.Helper()
	_, err := os.Stat(path)
	if err == nil {
		backupPath := path + ".bak"
		if err := os.Rename(path, backupPath); err != nil {
			t.Fatalf("failed to backup %s: %v", path, err)
		}
		return func() {
			_ = os.Remove(path)
			_ = os.Rename(backupPath, path)
		}
	}
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("unexpected stat error for %s: %v", path, err)
	}
	return func() {
		_ = os.Remove(path)
	}
}

func writeCacheFile(t *testing.T, path string, data interface{}) {
	t.Helper()
	if err := os.MkdirAll("data", 0o755); err != nil {
		t.Fatalf("failed to ensure data directory: %v", err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create %s: %v", path, err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(data); err != nil {
		t.Fatalf("failed to encode cache: %v", err)
	}
}

func TestLoadBonhamsFromCacheScenarios(t *testing.T) {
	restore := backupFile(t, BonhamsCacheFileName)
	defer restore()

	// Missing file
	if listings, ok := LoadBonhamsFromCache(); ok || listings != nil {
		t.Fatalf("expected no cache data, got %v", listings)
	}

	// Corrupted file
	if err := os.WriteFile(BonhamsCacheFileName, []byte("not json"), 0o644); err != nil {
		t.Fatalf("failed to write corrupted cache: %v", err)
	}
	if listings, ok := LoadBonhamsFromCache(); ok || listings != nil {
		t.Fatalf("expected corrupted cache to be ignored")
	}

	// Expired cache
	expired := BonhamsCache{
		Data:      []*models.BonhamsCar{{ID: "1"}},
		Timestamp: time.Now().Add(-BonhamsCacheExpiry - time.Hour),
	}
	writeCacheFile(t, BonhamsCacheFileName, expired)
	if listings, ok := LoadBonhamsFromCache(); ok || listings != nil {
		t.Fatalf("expected expired cache to be ignored")
	}

	// Fresh cache
	fresh := BonhamsCache{
		Data:      []*models.BonhamsCar{{ID: "2"}},
		Timestamp: time.Now(),
	}
	writeCacheFile(t, BonhamsCacheFileName, fresh)
	listings, ok := LoadBonhamsFromCache()
	if !ok || len(listings) != 1 || listings[0].ID != "2" {
		t.Fatalf("expected fresh cache to load, got %v", listings)
	}
}

func TestLoadLookersFromCacheScenarios(t *testing.T) {
	restore := backupFile(t, LookersCacheFileName)
	defer restore()

	// Missing file
	if listings, ok := LoadLookersFromCache(); ok || listings != nil {
		t.Fatalf("expected no cache data, got %v", listings)
	}

	// Corrupted file
	if err := os.WriteFile(LookersCacheFileName, []byte("not json"), 0o644); err != nil {
		t.Fatalf("failed to write corrupted cache: %v", err)
	}
	if listings, ok := LoadLookersFromCache(); ok || listings != nil {
		t.Fatalf("expected corrupted cache to be ignored")
	}

	// Expired cache
	expired := LookersCache{
		Data:      []*models.LookersCar{{ID: "1"}},
		Timestamp: time.Now().Add(-LookersCacheExpiry - time.Hour),
	}
	writeCacheFile(t, LookersCacheFileName, expired)
	if listings, ok := LoadLookersFromCache(); ok || listings != nil {
		t.Fatalf("expected expired cache to be ignored")
	}

	// Fresh cache
	fresh := LookersCache{
		Data:      []*models.LookersCar{{ID: "2"}},
		Timestamp: time.Now(),
	}
	writeCacheFile(t, LookersCacheFileName, fresh)
	listings, ok := LoadLookersFromCache()
	if !ok || len(listings) != 1 || listings[0].ID != "2" {
		t.Fatalf("expected fresh cache to load, got %v", listings)
	}
}

func TestSaveAndUtilityFunctions(t *testing.T) {
	restoreBonhams := backupFile(t, BonhamsCacheFileName)
	defer restoreBonhams()

	restoreLookers := backupFile(t, LookersCacheFileName)
	defer restoreLookers()

	cars := []*models.BonhamsCar{{ID: "bonhams"}}
	if err := SaveBonhamsToCache(cars); err != nil {
		t.Fatalf("SaveBonhamsToCache failed: %v", err)
	}

	loaded, ok := LoadFromCache()
	if !ok || len(loaded) != 1 || loaded[0].ID != "bonhams" {
		t.Fatalf("expected wrapper load to return saved data")
	}

	if err := SaveToCache(cars); err != nil {
		t.Fatalf("SaveToCache wrapper failed: %v", err)
	}

	lookers := []*models.LookersCar{{ID: "lookers"}}
	if err := SaveLookersToCache(lookers); err != nil {
		t.Fatalf("SaveLookersToCache failed: %v", err)
	}

	if expired := IsCacheExpired(); expired {
		t.Fatalf("expected cache not expired immediately")
	}

	if expired := IsBonhamsCacheExpired(); expired {
		t.Fatalf("expected bonhams cache not expired immediately")
	}

	if expired := IsLookersCacheExpired(); expired {
		t.Fatalf("expected lookers cache not expired immediately")
	}

	age, err := GetCacheAge()
	if err != nil {
		t.Fatalf("GetCacheAge failed: %v", err)
	}
	if age <= 0 {
		t.Fatalf("expected positive age, got %v", age)
	}

	age, err = GetBonhamsCacheAge()
	if err != nil {
		t.Fatalf("GetBonhamsCacheAge failed: %v", err)
	}

	lookersAge, err := GetLookersCacheAge()
	if err != nil {
		t.Fatalf("GetLookersCacheAge failed: %v", err)
	}
	if lookersAge <= 0 {
		t.Fatalf("expected positive lookers age, got %v", lookersAge)
	}
}

func TestCacheExpiredWhenMissing(t *testing.T) {
	restore := backupFile(t, BonhamsCacheFileName)
	defer restore()
	if !IsBonhamsCacheExpired() {
		t.Fatalf("expected cache to be expired when file missing")
	}
}

func TestLookersCacheExpiredOnCorruption(t *testing.T) {
	restore := backupFile(t, LookersCacheFileName)
	defer restore()

	if err := os.WriteFile(LookersCacheFileName, []byte("bad"), 0o644); err != nil {
		t.Fatalf("failed to write corrupted cache: %v", err)
	}

	if !IsLookersCacheExpired() {
		t.Fatalf("expected corrupted cache to be treated as expired")
	}
}

func TestBonhamsCacheExpiredOnCorruption(t *testing.T) {
	restore := backupFile(t, BonhamsCacheFileName)
	defer restore()

	if err := os.WriteFile(BonhamsCacheFileName, []byte("bad"), 0o644); err != nil {
		t.Fatalf("failed to write corrupted cache: %v", err)
	}

	if !IsBonhamsCacheExpired() {
		t.Fatalf("expected corrupted cache to be treated as expired")
	}
}
