package provider

import (
	"context"
	"net/http"
	"testing"
)

func TestOpenRouterFetchRequiresAPIKey(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")

	r := OpenRouter{}.Fetch(context.Background())
	if r.Error == nil {
		t.Fatal("expected missing API key error")
	}
	if r.Short != "?" {
		t.Fatalf("expected short '?', got %q", r.Short)
	}
}

func TestOpenRouterFetchAuthFailure(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-key")

	withMockDefaultClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("missing auth header: %q", req.Header.Get("Authorization"))
		}
		return jsonResponse(http.StatusUnauthorized, `{}`), nil
	})

	r := OpenRouter{}.Fetch(context.Background())
	if r.Error == nil {
		t.Fatal("expected auth error")
	}
	if r.Short != "!" {
		t.Fatalf("expected short '!', got %q", r.Short)
	}
}

func TestOpenRouterFetchSuccessAndLabelFiltering(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-key")

	withMockDefaultClient(t, func(req *http.Request) (*http.Response, error) {
		body := `{
		  "data": {
		    "label": "sk-live-should-not-show",
		    "limit": 100,
		    "limit_remaining": 25,
		    "usage": 75,
		    "usage_daily": 1.25,
		    "usage_weekly": 5.5,
		    "usage_monthly": 20.75,
		    "is_free_tier": true
		  }
		}`
		return jsonResponse(http.StatusOK, body), nil
	})

	r := OpenRouter{}.Fetch(context.Background())

	if r.Error != nil {
		t.Fatalf("expected success, got error: %v", r.Error)
	}
	if r.Identity != "" {
		t.Fatalf("expected sk-* label to be hidden, got %q", r.Identity)
	}
	if r.Short != "$25.00" {
		t.Fatalf("unexpected short value: %q", r.Short)
	}
	if r.Class != "warning" {
		t.Fatalf("expected warning class from 75%% usage, got %q", r.Class)
	}
	if r.Plan != "free" {
		t.Fatalf("expected free plan, got %q", r.Plan)
	}
	if len(r.Windows) != 1 || r.Windows[0].Label != "Budget" {
		t.Fatalf("unexpected windows: %#v", r.Windows)
	}
	if len(r.Spend) != 4 {
		t.Fatalf("expected 4 spend rows, got %d", len(r.Spend))
	}
}
