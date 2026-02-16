package cache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/joshhartzell/ai-usage-bar/internal/provider"
)

const maxAge = 1 * time.Hour

type entry struct {
	FetchedAt time.Time      `json:"fetched_at"`
	Results   []cachedResult `json:"results"`
}

type cachedResult struct {
	Name     string                `json:"name"`
	Identity string                `json:"identity,omitempty"`
	Short    string                `json:"short,omitempty"`
	Class    string                `json:"class,omitempty"`
	Windows  []provider.RateWindow `json:"windows,omitempty"`
	Spend    []provider.SpendEntry `json:"spend,omitempty"`
	Credits  *float64              `json:"credits,omitempty"`
	Plan     string                `json:"plan,omitempty"`
	Error    string                `json:"error,omitempty"`
}

func cacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ai-usage-bar"), nil
}

func cachePath() (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cache.json"), nil
}

// Load returns cached results if the cache exists and is less than maxAge old.
// Returns nil if the cache is missing, stale, or corrupt.
func Load() []provider.Result {
	path, err := cachePath()
	if err != nil {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var e entry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil
	}

	if time.Since(e.FetchedAt) > maxAge {
		return nil
	}

	results := make([]provider.Result, len(e.Results))
	for i, cr := range e.Results {
		r := provider.Result{
			Name:     cr.Name,
			Identity: cr.Identity,
			Short:    cr.Short,
			Class:    cr.Class,
			Windows:  cr.Windows,
			Spend:    cr.Spend,
			Credits:  cr.Credits,
			Plan:     cr.Plan,
		}
		if cr.Error != "" {
			r.Error = errors.New(cr.Error)
		}
		results[i] = r
	}
	return results
}

// Save writes results to the cache file.
func Save(results []provider.Result) {
	path, err := cachePath()
	if err != nil {
		return
	}

	dir, _ := cacheDir()
	os.MkdirAll(dir, 0o700)

	cached := make([]cachedResult, len(results))
	for i, r := range results {
		cr := cachedResult{
			Name:     r.Name,
			Identity: r.Identity,
			Short:    r.Short,
			Class:    r.Class,
			Windows:  r.Windows,
			Spend:    r.Spend,
			Credits:  r.Credits,
			Plan:     r.Plan,
		}
		if r.Error != nil {
			cr.Error = r.Error.Error()
		}
		cached[i] = cr
	}

	e := entry{
		FetchedAt: time.Now(),
		Results:   cached,
	}

	data, err := json.Marshal(e)
	if err != nil {
		return
	}

	os.WriteFile(path, data, 0o600)
}
