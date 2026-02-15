package waybar

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/joshhartzell/ai-usage-bar/internal/provider"
)

type Output struct {
	Text       string `json:"text"`
	Tooltip    string `json:"tooltip"`
	Class      string `json:"class"`
	Percentage int    `json:"percentage"`
}

func Format(results []provider.Result) Output {
	var tooltipParts []string
	worstClass := "normal"
	worstPct := 0.0

	for _, r := range results {
		tooltipParts = append(tooltipParts, formatTooltip(r))

		if r.Error == nil {
			for _, w := range r.Windows {
				if w.UsedPct > worstPct {
					worstPct = w.UsedPct
				}
			}
		}

		if classRank(r.Class) > classRank(worstClass) {
			worstClass = r.Class
		}
	}

	text := fmt.Sprintf("󱚣 %.0f%%", worstPct)

	return Output{
		Text:       text,
		Tooltip:    strings.Join(tooltipParts, "\n\n"),
		Class:      worstClass,
		Percentage: int(worstPct),
	}
}

func FormatJSON(o Output) string {
	b, _ := json.Marshal(o)
	return string(b)
}

func formatTooltip(r provider.Result) string {
	var sb strings.Builder

	header := r.Name
	if r.Plan != "" {
		header += " (" + r.Plan + ")"
	}
	sb.WriteString(header)
	if r.Identity != "" {
		sb.WriteString(fmt.Sprintf("\n  %s", r.Identity))
	}

	if r.Error != nil {
		sb.WriteString(fmt.Sprintf("\n  Error: %s", r.Error))
		return sb.String()
	}

	for _, w := range r.Windows {
		reset := ""
		if w.HasReset {
			d := time.Until(w.ResetAt)
			if d > 0 {
				reset = fmt.Sprintf(" (resets in %s)", formatResetDuration(d))
			} else {
				reset = " (resetting now)"
			}
		}
		sb.WriteString(fmt.Sprintf("\n  %s: %.0f%% used%s", w.Label, w.UsedPct, reset))
	}

	for _, s := range r.Spend {
		sb.WriteString(fmt.Sprintf("\n  %s: $%.2f", s.Label, s.Amount))
	}

	if r.Credits != nil {
		sb.WriteString(fmt.Sprintf("\n  Credits: $%.2f remaining", *r.Credits))
	}

	return sb.String()
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
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	default:
		return fmt.Sprintf("%dm", mins)
	}
}

func classRank(class string) int {
	switch class {
	case "critical":
		return 2
	case "warning":
		return 1
	default:
		return 0
	}
}
