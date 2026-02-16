package detail

import (
	"fmt"
	"html"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/joshhartzell/ai-usage-bar/internal/provider"
)

func ShowYad(results []provider.Result) {
	htmlDoc := renderHTML(results)
	width, height := popupSize(results)

	cmd := exec.Command("yad",
		"--html",
		"--title=AI Usage",
		fmt.Sprintf("--width=%d", width),
		fmt.Sprintf("--height=%d", height),
		"--borders=0",
		"--css=window,dialog,box,frame,scrolledwindow,viewport,grid { border: 0; box-shadow: none; background: #303446; } scrollbar, scrollbar slider { min-width: 0; min-height: 0; opacity: 0; }",
		"--user-style=html,body{margin:0;padding:0;background:#303446;overflow:hidden;}::-webkit-scrollbar{width:0;height:0;}",
		"--hscroll-policy=never",
		"--vscroll-policy=never",
		"--center",
		"--no-buttons",
		"--undecorated",
		"--skip-taskbar",
		"--class=ai-usage-detail",
	)
	cmd.Stdin = strings.NewReader(htmlDoc)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func renderHTML(results []provider.Result) string {
	var sb strings.Builder

	sb.WriteString(`<html><head><style>
html, body {
  margin: 0;
  padding: 0;
  width: 100%;
  height: 100%;
}
body {
  background: radial-gradient(circle at 10% 10%, #414559 0%, #303446 42%, #292c3c 100%);
  color: #c6d0f5;
  font-family: 'JetBrains Mono', 'JetBrainsMono Nerd Font', monospace;
  font-size: 13px;
  overflow: hidden;
}
.panel {
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  position: relative;
  padding: 10px 12px 24px;
  background: linear-gradient(180deg, #303446 0%, #2f3446 100%);
}
.title-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 10px;
  padding-bottom: 8px;
  border-bottom: 1px solid #51576d;
}
.title {
  font-size: 14px;
  letter-spacing: 0.04em;
  color: #f2d5cf;
}
.subtitle {
  font-size: 11px;
  color: #838ba7;
}
.provider {
  --accent: #8caaee;
  margin-bottom: 10px;
  padding: 9px 10px 7px;
  border: 1px solid #51576d;
  border-radius: 10px;
  background: linear-gradient(180deg, #353a4f 0%, #32374a 100%);
}
.provider:last-child { margin-bottom: 0; }
.provider.claude {
  --accent: #ca9ee6;
  border-color: #7b5f92;
}
.provider.codex {
  --accent: #a6d189;
  border-color: #678456;
}
.provider.openrouter {
  --accent: #81c8be;
  border-color: #4f8178;
}
.provider-name {
  font-weight: 700;
  font-size: 14px;
  margin-bottom: 6px;
  color: var(--accent);
  letter-spacing: 0.02em;
}
.plan {
  color: #a5adce;
  font-weight: normal;
  font-size: 11px;
  margin-left: 2px;
}
.identity {
  color: #b5bfe2;
  font-size: 11px;
  margin-bottom: 6px;
}
.meter-row {
  display: grid;
  grid-template-columns: 92px 1fr 44px 112px;
  align-items: center;
  margin: 4px 0;
  gap: 8px;
}
.meter-label {
  color: #a5adce;
  font-size: 12px;
}
.bar-bg {
  width: 100%;
  height: 12px;
  background: #414559;
  border-radius: 999px;
  overflow: hidden;
  box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.25);
}
.bar-fill {
  height: 100%;
  border-radius: 999px;
  min-width: 8px;
}
.pct {
  text-align: right;
  font-size: 12px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}
.reset {
  color: #838ba7;
  font-size: 11px;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}
.kv-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin: 4px 0;
}
.kv-label {
  color: #a5adce;
}
.kv-value {
  font-size: 12px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.spend { color: #c6d0f5; }
.credits { color: #a6d189; }
.error {
  color: #e78284;
  font-size: 12px;
  margin-top: 4px;
}
.no-data {
  color: #838ba7;
  font-size: 11px;
  margin-top: 3px;
}
.hint {
  position: absolute;
  right: 12px;
  bottom: 6px;
  color: #737994;
  font-size: 11px;
}
</style></head><body>`)
	sb.WriteString(`<div class="panel">`)
	sb.WriteString(`<div class="title-row"><span class="title">AI Usage</span><span class="subtitle">live account snapshot</span></div>`)

	for _, r := range results {
		cssClass := strings.ToLower(strings.ReplaceAll(r.Name, " ", ""))
		sb.WriteString(fmt.Sprintf(`<div class="provider %s">`, cssClass))
		sb.WriteString(fmt.Sprintf(`<div class="provider-name">%s`, html.EscapeString(r.Name)))
		if r.Plan != "" {
			sb.WriteString(fmt.Sprintf(` <span class="plan">(%s)</span>`, html.EscapeString(r.Plan)))
		}
		sb.WriteString(`</div>`)
		if r.Identity != "" {
			sb.WriteString(fmt.Sprintf(`<div class="identity">%s</div>`, html.EscapeString(r.Identity)))
		}

		hasData := false
		if r.Error != nil {
			sb.WriteString(fmt.Sprintf(`<div class="error">%s</div>`, html.EscapeString(r.Error.Error())))
		} else {
			for _, w := range r.Windows {
				hasData = true
				color := colorForPct(w.UsedPct)
				resetStr := ""
				if w.HasReset {
					d := time.Until(w.ResetAt)
					if d > 0 {
						resetStr = fmt.Sprintf("resets in %s", formatDuration(d))
					}
				}
				sb.WriteString(fmt.Sprintf(`<div class="meter-row">
					<span class="meter-label">%s</span>
					<div class="bar-bg"><div class="bar-fill" style="width:%.0f%%;background:%s"></div></div>
					<span class="pct" style="color:%s">%.0f%%</span>
					<span class="reset">%s</span>
				</div>`, html.EscapeString(w.Label), w.UsedPct, color, color, w.UsedPct, html.EscapeString(resetStr)))
			}

			for _, s := range r.Spend {
				hasData = true
				sb.WriteString(fmt.Sprintf(`<div class="kv-row">
					<span class="kv-label">%s</span>
					<span class="kv-value spend">$%.2f</span>
				</div>`, html.EscapeString(s.Label), s.Amount))
			}

			if r.Credits != nil {
				hasData = true
				creditsLabel := "Credits"
				if r.Name == "Claude" {
					creditsLabel = "Extra usage remaining"
				}
				sb.WriteString(fmt.Sprintf(`<div class="kv-row">
					<span class="kv-label">%s</span>
					<span class="kv-value credits">$%.2f</span>
				</div>`, html.EscapeString(creditsLabel), *r.Credits))
			}

			if !hasData {
				sb.WriteString(`<div class="no-data">No usage metrics available.</div>`)
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
<div class="hint">press q to close</div>
</div></body></html>`)
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

func popupSize(results []provider.Result) (int, int) {
	const width = 560

	height := 92
	for _, r := range results {
		rows := 1
		if r.Error == nil {
			rows = len(r.Windows) + len(r.Spend)
			if r.Credits != nil {
				rows++
			}
			if rows == 0 {
				rows = 1
			}
		}

		height += 62 + rows*22
	}

	if height < 300 {
		height = 300
	}
	if height > 760 {
		height = 760
	}

	return width, height
}
