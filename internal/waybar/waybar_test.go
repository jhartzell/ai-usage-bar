package waybar

import (
	"encoding/json"
	"testing"

	"github.com/jhartzell/ai-usage-bar/internal/provider"
)

func TestFormatUsesWorstWindowPctIgnoringErrorProviders(t *testing.T) {
	results := []provider.Result{
		{
			Name: "Claude",
			Windows: []provider.RateWindow{
				{Label: "Session", UsedPct: 35},
				{Label: "Weekly", UsedPct: 78},
			},
			Class: "warning",
		},
		{
			Name:  "Codex",
			Error: assertErr("boom"),
			Windows: []provider.RateWindow{
				{Label: "Session", UsedPct: 99},
			},
			Class: "critical",
		},
	}

	out := Format(results)

	if out.Percentage != 78 {
		t.Fatalf("expected worst percentage 78, got %d", out.Percentage)
	}
	if len(out.Text) < len(" 78%") || out.Text[len(out.Text)-len(" 78%"):] != " 78%" {
		t.Fatalf("expected text to end with ' 78%%', got %q", out.Text)
	}
}

func TestFormatUsesHighestClassRank(t *testing.T) {
	results := []provider.Result{
		{Name: "A", Class: "normal", Windows: []provider.RateWindow{{UsedPct: 10}}},
		{Name: "B", Class: "warning", Windows: []provider.RateWindow{{UsedPct: 20}}},
		{Name: "C", Class: "critical", Windows: []provider.RateWindow{{UsedPct: 30}}},
	}

	out := Format(results)
	if out.Class != "critical" {
		t.Fatalf("expected critical class, got %q", out.Class)
	}
}

func TestFormatJSONProducesValidJSON(t *testing.T) {
	o := Output{Text: "x", Tooltip: "y", Class: "warning", Percentage: 75}
	encoded := FormatJSON(o)

	var decoded Output
	if err := json.Unmarshal([]byte(encoded), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if decoded != o {
		t.Fatalf("unexpected decoded output: got %#v want %#v", decoded, o)
	}
}

func TestClassRank(t *testing.T) {
	if classRank("critical") <= classRank("warning") {
		t.Fatal("critical should rank above warning")
	}
	if classRank("warning") <= classRank("normal") {
		t.Fatal("warning should rank above normal")
	}
	if classRank("anything-else") != 0 {
		t.Fatal("unknown classes should default to normal rank")
	}
}

type testErr string

func (e testErr) Error() string { return string(e) }

func assertErr(msg string) error {
	return testErr(msg)
}
