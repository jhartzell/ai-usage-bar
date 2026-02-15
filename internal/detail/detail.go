package detail

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/joshhartzell/ai-usage-bar/internal/provider"
)

func ShowYad(results []provider.Result) {
	html := renderHTML(results)

	cmd := exec.Command("yad",
		"--html",
		"--title=AI Usage",
		"--width=420",
		"--height=380",
		"--center",
		"--no-buttons",
		"--undecorated",
		"--skip-taskbar",
		"--class=ai-usage-detail",
	)
	cmd.Stdin = strings.NewReader(html)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func renderHTML(results []provider.Result) string {
	var sb strings.Builder

	sb.WriteString(`<html><head><style>
body {
  background: #303446;
  color: #c6d0f5;
  font-family: 'JetBrains Mono', 'JetBrainsMono Nerd Font', monospace;
  font-size: 13px;
  padding: 16px;
  margin: 0;
}
.provider { margin-bottom: 16px; }
.provider-name {
  font-weight: bold;
  font-size: 15px;
  margin-bottom: 6px;
}
.plan { color: #a5adce; font-weight: normal; font-size: 12px; }
.row {
  display: flex;
  align-items: center;
  margin: 4px 0;
  gap: 8px;
}
.label { width: 110px; color: #a5adce; font-size: 12px; }
.bar-bg {
  width: 120px;
  height: 10px;
  background: #414559;
  border-radius: 5px;
  overflow: hidden;
}
.bar-fill {
  height: 100%;
  border-radius: 5px;
}
.pct { width: 40px; text-align: right; font-size: 12px; }
.reset { color: #737994; font-size: 11px; }
.value { font-size: 12px; }
.spend { color: #c6d0f5; }
.credits { color: #a6d189; }
.error { color: #e78284; }
.claude .provider-name { color: #ca9ee6; }
.codex .provider-name { color: #a6d189; }
.openrouter .provider-name { color: #81c8be; }
</style></head><body>`)

	for _, r := range results {
		cssClass := strings.ToLower(strings.ReplaceAll(r.Name, " ", ""))
		sb.WriteString(fmt.Sprintf(`<div class="provider %s">`, cssClass))
		sb.WriteString(fmt.Sprintf(`<div class="provider-name">%s`, r.Name))
		if r.Plan != "" {
			sb.WriteString(fmt.Sprintf(` <span class="plan">(%s)</span>`, r.Plan))
		}
		sb.WriteString(`</div>`)
		if r.Identity != "" {
			sb.WriteString(fmt.Sprintf(`<div style="color:#a5adce;font-size:11px;margin-bottom:6px;">%s</div>`, r.Identity))
		}

		if r.Error != nil {
			sb.WriteString(fmt.Sprintf(`<div class="row"><span class="error">%s</span></div>`, r.Error))
		} else {
			for _, w := range r.Windows {
				color := colorForPct(w.UsedPct)
				resetStr := ""
				if w.HasReset {
					d := time.Until(w.ResetAt)
					if d > 0 {
						resetStr = fmt.Sprintf(`<span class="reset">resets in %s</span>`, formatDuration(d))
					}
				}
				sb.WriteString(fmt.Sprintf(`<div class="row">
					<span class="label">%s</span>
					<div class="bar-bg"><div class="bar-fill" style="width:%.0f%%;background:%s"></div></div>
					<span class="pct" style="color:%s">%.0f%%</span>
					%s
				</div>`, w.Label, w.UsedPct, color, color, w.UsedPct, resetStr))
			}

			for _, s := range r.Spend {
				sb.WriteString(fmt.Sprintf(`<div class="row">
					<span class="label">%s</span>
					<span class="value spend">$%.2f</span>
				</div>`, s.Label, s.Amount))
			}

			if r.Credits != nil {
				sb.WriteString(fmt.Sprintf(`<div class="row">
					<span class="label">Credits</span>
					<span class="value credits">$%.2f</span>
				</div>`, *r.Credits))
			}
		}

		sb.WriteString(`</div>`)
	}

	sb.WriteString(`<script>
document.addEventListener('keydown', function(e) {
  if (e.key === 'q' || e.key === 'Q' || e.key === 'Escape') {
    window.close();
  }
});
</script>
<div style="position:fixed;bottom:8px;right:12px;color:#737994;font-size:11px;">press q to close</div>
</body></html>`)
	return sb.String()
}

func colorForPct(pct float64) string {
	switch {
	case pct >= 90:
		return "#e78284"
	case pct >= 75:
		return "#e5c890"
	default:
		return "#a6d189"
	}
}

func formatDuration(d time.Duration) string {
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
