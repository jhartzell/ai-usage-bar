package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joshhartzell/ai-usage-bar/internal/provider"
)

func TestLoadReturnsNilWhenMissing(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	if got := Load(); got != nil {
		t.Fatalf("expected nil on missing cache, got %#v", got)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	credits := 12.34
	results := []provider.Result{
		{
			Name:     "Claude",
			Identity: "user@example.com",
			Short:    "45%",
			Class:    "warning",
			Plan:     "max",
			Windows: []provider.RateWindow{
				{Label: "Session (5h)", UsedPct: 45, HasReset: true, ResetAt: time.Unix(1_700_000_000, 0)},
			},
			Spend:   []provider.SpendEntry{{Label: "This month", Amount: 3.21}},
			Credits: &credits,
		},
	}

	Save(results)
	loaded := Load()

	if loaded == nil {
		t.Fatal("expected cache load result, got nil")
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 result, got %d", len(loaded))
	}

	got := loaded[0]
	if got.Name != "Claude" || got.Identity != "user@example.com" || got.Short != "45%" || got.Class != "warning" {
		t.Fatalf("unexpected loaded metadata: %#v", got)
	}
	if got.Plan != "max" {
		t.Fatalf("unexpected plan: %q", got.Plan)
	}
	if got.Credits == nil || *got.Credits != credits {
		t.Fatalf("unexpected credits: %#v", got.Credits)
	}
	if len(got.Windows) != 1 || got.Windows[0].Label != "Session (5h)" || got.Windows[0].UsedPct != 45 {
		t.Fatalf("unexpected windows: %#v", got.Windows)
	}
	if len(got.Spend) != 1 || got.Spend[0].Amount != 3.21 {
		t.Fatalf("unexpected spend: %#v", got.Spend)
	}
}

func TestLoadReturnsNilForStaleCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	writeCacheEntry(t, entry{
		FetchedAt: time.Now().Add(-2 * time.Hour),
		Results:   []cachedResult{{Name: "Claude"}},
	})

	if got := Load(); got != nil {
		t.Fatalf("expected nil for stale cache, got %#v", got)
	}
}

func TestLoadReturnsNilWhenAnyProviderErrorCached(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	writeCacheEntry(t, entry{
		FetchedAt: time.Now(),
		Results: []cachedResult{
			{Name: "Claude", Error: "auth failed"},
		},
	})

	if got := Load(); got != nil {
		t.Fatalf("expected nil when cache contains provider error, got %#v", got)
	}
}

func TestLoadReturnsNilForCorruptJSON(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	path, err := cachePath()
	if err != nil {
		t.Fatalf("cachePath: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{not-json"), 0o600); err != nil {
		t.Fatalf("write corrupt cache: %v", err)
	}

	if got := Load(); got != nil {
		t.Fatalf("expected nil for corrupt cache, got %#v", got)
	}
}

func TestClearRemovesCacheFile(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	writeCacheEntry(t, entry{
		FetchedAt: time.Now(),
		Results:   []cachedResult{{Name: "Claude"}},
	})

	if err := Clear(); err != nil {
		t.Fatalf("clear cache: %v", err)
	}

	path, err := cachePath()
	if err != nil {
		t.Fatalf("cachePath: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected cache file removed, stat error: %v", err)
	}
}

func writeCacheEntry(t *testing.T, e entry) {
	t.Helper()

	path, err := cachePath()
	if err != nil {
		t.Fatalf("cachePath: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal entry: %v", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write cache: %v", err)
	}
}
