package provider

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubProvider struct {
	name  string
	fetch func(ctx context.Context) Result
}

func (s stubProvider) Name() string {
	return s.name
}

func (s stubProvider) Fetch(ctx context.Context) Result {
	return s.fetch(ctx)
}

func TestFetchAllPreservesProviderOrder(t *testing.T) {
	providers := []Provider{
		stubProvider{name: "first", fetch: func(ctx context.Context) Result {
			time.Sleep(15 * time.Millisecond)
			return Result{Name: "first", Short: "1"}
		}},
		stubProvider{name: "second", fetch: func(ctx context.Context) Result {
			return Result{Name: "second", Short: "2"}
		}},
		stubProvider{name: "third", fetch: func(ctx context.Context) Result {
			time.Sleep(5 * time.Millisecond)
			return Result{Name: "third", Short: "3"}
		}},
	}

	results := FetchAll(context.Background(), providers)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].Name != "first" || results[1].Name != "second" || results[2].Name != "third" {
		t.Fatalf("results out of provider order: %#v", results)
	}
}

func TestFetchAllUsesParentDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()

	providers := []Provider{
		stubProvider{name: "slow", fetch: func(ctx context.Context) Result {
			<-ctx.Done()
			return Result{Name: "slow", Error: ctx.Err(), Short: "timeout"}
		}},
		stubProvider{name: "fast", fetch: func(ctx context.Context) Result {
			return Result{Name: "fast", Short: "ok"}
		}},
	}

	start := time.Now()
	results := FetchAll(ctx, providers)
	elapsed := time.Since(start)

	if elapsed > 500*time.Millisecond {
		t.Fatalf("FetchAll should return quickly on parent deadline, took %s", elapsed)
	}

	if !errors.Is(results[0].Error, context.DeadlineExceeded) {
		t.Fatalf("expected slow provider deadline exceeded, got %v", results[0].Error)
	}

	if results[1].Short != "ok" {
		t.Fatalf("expected fast provider result, got %#v", results[1])
	}
}

func TestClassFromPct(t *testing.T) {
	tests := []struct {
		name string
		pct  float64
		want string
	}{
		{name: "normal low", pct: 0, want: "normal"},
		{name: "normal high", pct: 74.9, want: "normal"},
		{name: "warning boundary", pct: 75, want: "warning"},
		{name: "warning high", pct: 89.9, want: "warning"},
		{name: "critical boundary", pct: 90, want: "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classFromPct(tt.pct)
			if got != tt.want {
				t.Fatalf("classFromPct(%v): got %q, want %q", tt.pct, got, tt.want)
			}
		})
	}
}

func TestFormatResetDuration(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{name: "now", in: 0, want: "now"},
		{name: "minutes", in: 59 * time.Minute, want: "59m"},
		{name: "hours and mins", in: 2*time.Hour + 5*time.Minute, want: "2h 5m"},
		{name: "days and hours", in: 49 * time.Hour, want: "2d 1h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatResetDuration(tt.in)
			if got != tt.want {
				t.Fatalf("formatResetDuration(%s): got %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{in: 0, want: "0"},
		{in: 9, want: "9"},
		{in: 42, want: "42"},
		{in: -7, want: "-7"},
		{in: -1234, want: "-1234"},
	}

	for _, tt := range tests {
		got := itoa(tt.in)
		if got != tt.want {
			t.Fatalf("itoa(%d): got %q, want %q", tt.in, got, tt.want)
		}
	}
}
