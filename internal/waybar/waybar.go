package waybar

import (
	"encoding/json"
	"fmt"

	"github.com/joshhartzell/ai-usage-bar/internal/provider"
)

type Output struct {
	Text       string `json:"text"`
	Tooltip    string `json:"tooltip"`
	Class      string `json:"class"`
	Percentage int    `json:"percentage"`
}

func Format(results []provider.Result) Output {
	worstClass := "normal"
	worstPct := 0.0

	for _, r := range results {
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
		Class:      worstClass,
		Percentage: int(worstPct),
	}
}

func FormatJSON(o Output) string {
	b, _ := json.Marshal(o)
	return string(b)
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
