# ai-usage-bar — Keep your AI limits visible from your status bar.

A lightweight waybar module for Linux that monitors your Claude, Codex (ChatGPT), and OpenRouter usage at a glance. Single status icon shows worst-case utilization across all providers; click for a detailed popup with per-provider breakdowns, progress bars, spend tracking, and reset timers.

Built in Go with zero external dependencies. Catppuccin Frappe themed.

<img src="screenshot.jpg" alt="ai-usage-bar detail popup" width="420" />

## Install

### Requirements

- Linux with [Sway](https://swaywm.org/) (or any wlroots compositor)
- [waybar](https://github.com/Alexays/Waybar) as your status bar
- [yad](https://github.com/v1cont/yad) for the detail popup
- Go 1.25+ to build from source

### Build from source

```bash
git clone git@github.com:jhartzell/ai-usage-bar.git
cd ai-usage-bar
go build -o ai-usage-bar ./cmd/ai-usage-bar
cp ai-usage-bar ~/.local/bin/
```

Or with [go-task](https://taskfile.dev/):

```bash
task install    # builds and copies to ~/.local/bin/
```

### First run

1. Sign in to the provider CLIs you use (`claude`, `codex`) so credential files exist.
2. Set `$OPENROUTER_API_KEY` in your environment if you use OpenRouter.
3. Add the waybar module config (see [Waybar Setup](#waybar-setup) below).
4. Reload waybar: `killall -SIGUSR2 waybar`

## Providers

| Provider | Auth Source | What It Shows |
|---|---|---|
| **Claude** | `~/.claude/.credentials.json` (OAuth) | 5-hour session + 7-day weekly utilization, extra usage credits |
| **Codex** | `~/.codex/auth.json` | 5-hour session + 7-day weekly utilization, rate limit status |
| **OpenRouter** | `$OPENROUTER_API_KEY` env var | Daily/weekly/monthly/all-time spend, budget remaining |

All providers are optional — missing credentials show a `?` indicator, not a crash. Auth failures show `!`.

## How It Works

**Two modes:**

- **Default** (no args) — outputs a single JSON line for waybar's custom module protocol. Shows a combined icon with the worst-case percentage across all providers.
- **`--detail`** — opens a yad HTML popup with full per-provider breakdowns.

**Caching:**

API calls are made at most **once per hour**. Results are cached to `~/.cache/ai-usage-bar/cache.json`. Since waybar polls the binary every 120 seconds, ~29 out of 30 invocations serve cached data — no unnecessary API traffic.

**Concurrent fetching:**

All providers are fetched in parallel with a 5-second timeout each. A slow or failing provider won't block the others.

## Waybar Setup

### Module config

Add to your waybar `config`:

```json
"custom/ai_usage": {
    "exec": "~/.local/bin/ai-usage-bar",
    "return-type": "json",
    "interval": 120,
    "tooltip": false,
    "on-click": "~/.local/bin/ai-usage-bar --detail"
}
```

Then add `"custom/ai_usage"` to your `modules-left`, `modules-center`, or `modules-right` array.

### Styling

Add to your waybar `style.css`:

```css
#custom-ai_usage {
    color: #81c8be;
}

#custom-ai_usage.warning {
    color: #e5c890;
}

#custom-ai_usage.critical {
    color: #e78284;
}
```

The module outputs a CSS class based on the worst utilization: `normal` (<75%), `warning` (75-89%), `critical` (90%+).

### Sway float rules

Add to your Sway config so the detail popup floats centered:

```
for_window [app_id="ai-usage-detail"] floating enable
for_window [title="AI Usage"] floating enable
```

## Detail Popup

The click-to-open popup shows:

- **Per-provider sections** with colored headers (Claude: purple, Codex: green, OpenRouter: teal)
- **Progress bars** color-coded green → yellow → red based on utilization
- **Reset countdowns** showing when rate limits refresh
- **Spend breakdowns** (daily, weekly, monthly, all-time)
- **Credits remaining** where applicable
- **Account identity** (email or key label)

Close with `q`, `Q`, or `Escape`.

## Color Scheme

Uses [Catppuccin Frappe](https://github.com/catppuccin/catppuccin) throughout:

| Role | Color |
|---|---|
| Background | `#303446` (Base) |
| Text | `#c6d0f5` (Text) |
| Normal | `#a6d189` (Green) |
| Warning | `#e5c890` (Yellow) |
| Critical | `#e78284` (Red) |
| Claude | `#ca9ee6` (Mauve) |
| Codex | `#a6d189` (Green) |
| OpenRouter | `#81c8be` (Teal) |

## Project Structure

```
cmd/ai-usage-bar/main.go       Entry point
internal/
  cache/cache.go                File-based JSON cache (1h TTL)
  provider/
    provider.go                 Provider interface, concurrent fetcher
    claude.go                   Claude OAuth API
    codex.go                    Codex (ChatGPT) API
    openrouter.go               OpenRouter API
  detail/detail.go              yad HTML popup
  waybar/waybar.go              Waybar JSON output
```

## Platform Support

| Platform | Status |
|---|---|
| Linux (Sway + waybar) | Supported |
| Linux (i3 + i3bar) | Untested, should work (outputs i3bar-compatible JSON) |
| Linux (other compositors + waybar) | Should work |
| macOS | Not supported (use [CodexBar](https://github.com/steipete/CodexBar)) |
| Windows | Not supported |

## Related

- [CodexBar](https://github.com/steipete/CodexBar) — macOS menu bar app for AI provider usage (the inspiration for this project)
- [ccusage](https://github.com/ryoppippi/ccusage) — CLI tool for Claude Code cost tracking

## License

MIT
