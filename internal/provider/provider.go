package provider

import (
	"context"
	"sync"
	"time"
)

type RateWindow struct {
	Label    string
	UsedPct  float64
	ResetAt  time.Time
	HasReset bool
}

type SpendEntry struct {
	Label  string
	Amount float64
}

type Result struct {
	Name     string
	Identity string // email, key label, or account name
	Short    string // e.g. "42%" or "$1.23"
	Class    string // "normal", "warning", "critical"
	Windows  []RateWindow
	Spend    []SpendEntry
	Credits  *float64
	Plan     string
	Error    error
}

type Provider interface {
	Name() string
	Fetch(ctx context.Context) Result
}

func FetchAll(ctx context.Context, providers []Provider) []Result {
	results := make([]Result, len(providers))
	var wg sync.WaitGroup

	for i, p := range providers {
		wg.Add(1)
		go func(idx int, prov Provider) {
			defer wg.Done()
			fetchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			results[idx] = prov.Fetch(fetchCtx)
		}(i, p)
	}

	wg.Wait()
	return results
}

func classFromPct(pct float64) string {
	switch {
	case pct >= 90:
		return "critical"
	case pct >= 75:
		return "warning"
	default:
		return "normal"
	}
}

func formatResetDuration(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	switch {
	case days > 0:
		return pluralize(days, "d") + " " + pluralize(hours, "h")
	case hours > 0:
		return pluralize(hours, "h") + " " + pluralize(mins, "m")
	default:
		return pluralize(mins, "m")
	}
}

func pluralize(n int, unit string) string {
	return itoa(n) + unit
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
