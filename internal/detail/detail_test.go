package detail

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/joshhartzell/ai-usage-bar/internal/provider"
)

func TestRenderHTMLEscapesUserFields(t *testing.T) {
	results := []provider.Result{
		{
			Name:     `<img src=x onerror=alert(1)>`,
			Identity: `user<script>alert(1)</script>@example.com`,
			Plan:     `<max>`,
		},
	}

	html := renderHTML(results)

	if strings.Contains(html, `<img src=x onerror=alert(1)>`) {
		t.Fatal("expected provider name to be escaped")
	}
	if strings.Contains(html, `<script>alert(1)</script>`) {
		t.Fatal("expected identity to be escaped")
	}
	if !strings.Contains(html, `&lt;max&gt;`) {
		t.Fatalf("expected escaped plan in HTML, got: %s", html)
	}
}

func TestRenderHTMLShowsRecoverButtonWhenAuthErrorPresent(t *testing.T) {
	html := renderHTML([]provider.Result{{Name: "Claude", Short: "!", Error: errors.New("auth expired")}})
	if !strings.Contains(html, "ai-usage-bar://recover-auth") {
		t.Fatalf("expected recover-auth link in HTML")
	}
}

func TestToProviderViewClaudeCreditsLabel(t *testing.T) {
	credits := 50.0
	v := toProviderView(provider.Result{
		Name:    "Claude",
		Credits: &credits,
	})

	if !v.ShowCredits {
		t.Fatal("expected credits to be shown")
	}
	if v.CreditsLabel != "Extra usage remaining" {
		t.Fatalf("unexpected credits label: %q", v.CreditsLabel)
	}
	if v.NoData {
		t.Fatal("provider with credits should not be marked as NoData")
	}
}

func TestToProviderViewClampsWindowPctAndResetText(t *testing.T) {
	v := toProviderView(provider.Result{
		Name: "Codex",
		Windows: []provider.RateWindow{
			{Label: "Past", UsedPct: -10, HasReset: true, ResetAt: time.Now().Add(-time.Minute)},
			{Label: "Future", UsedPct: 120, HasReset: true, ResetAt: time.Now().Add(2 * time.Hour)},
		},
	})

	if len(v.Windows) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(v.Windows))
	}
	if v.Windows[0].UsedPct != 0 {
		t.Fatalf("expected clamped low percent 0, got %v", v.Windows[0].UsedPct)
	}
	if v.Windows[1].UsedPct != 100 {
		t.Fatalf("expected clamped high percent 100, got %v", v.Windows[1].UsedPct)
	}
	if v.Windows[0].Reset != "" {
		t.Fatalf("expected past reset to be blank, got %q", v.Windows[0].Reset)
	}
	if !strings.Contains(v.Windows[1].Reset, "resets in") {
		t.Fatalf("expected future reset text, got %q", v.Windows[1].Reset)
	}
}

func TestToProviderViewErrorShortCircuitsDataRows(t *testing.T) {
	v := toProviderView(provider.Result{
		Name:  "Claude",
		Error: errors.New("boom"),
		Windows: []provider.RateWindow{
			{Label: "ignored", UsedPct: 10},
		},
	})

	if v.Error != "boom" {
		t.Fatalf("expected error text, got %q", v.Error)
	}
	if len(v.Windows) != 0 || len(v.Spend) != 0 || v.ShowCredits {
		t.Fatalf("expected no data rows when provider has error, got %#v", v)
	}
}

func TestPopupSizeBounds(t *testing.T) {
	width, height := popupSize(nil)
	if width != 560 {
		t.Fatalf("unexpected width: %d", width)
	}
	if height != 300 {
		t.Fatalf("expected minimum height 300, got %d", height)
	}

	var large []provider.Result
	for i := 0; i < 25; i++ {
		windows := make([]provider.RateWindow, 0, 8)
		for j := 0; j < 8; j++ {
			windows = append(windows, provider.RateWindow{Label: "W", UsedPct: 50})
		}
		large = append(large, provider.Result{Name: "P", Windows: windows})
	}

	_, capped := popupSize(large)
	if capped != 760 {
		t.Fatalf("expected capped height 760, got %d", capped)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{in: 0, want: "now"},
		{in: 35 * time.Minute, want: "35m"},
		{in: 2*time.Hour + 4*time.Minute, want: "2h 4m"},
		{in: 26 * time.Hour, want: "1d 2h"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.in)
		if got != tt.want {
			t.Fatalf("formatDuration(%s): got %q want %q", tt.in, got, tt.want)
		}
	}
}

func TestColorForPctThresholds(t *testing.T) {
	if got := colorForPct(95); got != "#e78284" {
		t.Fatalf("critical color mismatch: %q", got)
	}
	if got := colorForPct(80); got != "#e5c890" {
		t.Fatalf("warning color mismatch: %q", got)
	}
	if got := colorForPct(60); got != "#a6d189" {
		t.Fatalf("normal color mismatch: %q", got)
	}
}

func TestProviderClassAndClampPct(t *testing.T) {
	if got := providerClass("Open Router"); got != "openrouter" {
		t.Fatalf("unexpected provider class: %q", got)
	}
	if got := clampPct(-1); got != 0 {
		t.Fatalf("expected clamp low to 0, got %v", got)
	}
	if got := clampPct(110); got != 100 {
		t.Fatalf("expected clamp high to 100, got %v", got)
	}
}
