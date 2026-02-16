# ai-usage-bar

A waybar custom module that monitors AI provider usage (Claude, Codex/ChatGPT, OpenRouter) and displays a combined status icon. Clicking the icon opens a styled yad HTML popup with detailed per-provider breakdowns. Written in Go with zero external dependencies.

## OS & Environment

- **Linux only** — depends on waybar, yad, and Sway/i3 window manager
- **Sway** compositor with **waybar** as the status bar
- **yad** (Yet Another Dialog) for the HTML detail popup
- Go 1.25+ for building
- go-task for the build system

## Requirements

| Dependency | Purpose | Install |
|---|---|---|
| Go 1.25+ | Compile the binary | System package manager / nix |
| go-task | Build automation (`Taskfile.yml`) | `go install github.com/go-task/task/v3/cmd/task@latest` |
| waybar | Status bar that runs the module | `pacman -S waybar` |
| yad | HTML popup for `--detail` mode | `pacman -S yad` |
| Sway | Wayland compositor (float rules for popup) | `pacman -S sway` |

## Credential / Auth Setup

Each provider requires pre-existing credentials:

| Provider | Credential Source | Notes |
|---|---|---|
| Claude | `~/.claude/.credentials.json` | OAuth token from Claude CLI login. Field: `claudeAiOauth.accessToken` |
| Codex | `~/.codex/auth.json` | Token from Codex CLI login. Field: `tokens.access_token` |
| OpenRouter | `$OPENROUTER_API_KEY` env var | API key from openrouter.ai dashboard |

All three are optional — missing credentials result in a `?` or `!` indicator, not a crash.

## Build & Run

```bash
task build     # compile to ./ai-usage-bar
task install   # compile to ~/.local/bin/ai-usage-bar
task dev       # run via go run
task fmt       # gofmt
task lint      # golangci-lint
```

## How It Works

### Two Modes

1. **Waybar mode** (default, no args): outputs a single JSON line for waybar's custom module protocol:
   ```json
   {"text":"󱚣 42%","tooltip":"","class":"normal","percentage":42}
   ```
2. **Detail mode** (`--detail`): opens a yad HTML popup with per-provider breakdowns, progress bars, spend info, and reset timers.

### Data Flow

```
main.go
  ├─ cache.Load()  →  check ~/.cache/ai-usage-bar/cache.json (1h TTL)
  │   ├─ cache hit  →  use cached results
  │   └─ cache miss →  provider.FetchAll() (concurrent, 5s timeout each)
  │                     ├─ Claude: GET https://api.anthropic.com/api/oauth/usage
  │                     │          GET https://api.anthropic.com/api/oauth/profile
  │                     ├─ Codex:  GET https://chatgpt.com/backend-api/wham/usage
  │                     └─ OpenRouter: GET https://openrouter.ai/api/v1/key
  │                     └─ cache.Save() → write results to cache file
  ├─ (default)     →  waybar.Format() → JSON to stdout
  └─ (--detail)    →  detail.ShowYad() → pipe HTML to yad --html
```

### Caching

- Cache file: `~/.cache/ai-usage-bar/cache.json`
- TTL: **1 hour** — API calls are only made when cache is missing, corrupt, or older than 1h
- Waybar calls the binary every 120s, so ~29 invocations serve cached data per fetch cycle
- Both waybar mode and `--detail` mode share the same cache
- Errors are cached too (as string representations)

### Provider Details

**Claude** (`internal/provider/claude.go`)
- Auth: OAuth bearer token from `~/.claude/.credentials.json`
- API header: `anthropic-beta: oauth-2025-04-20`
- Returns: 5-hour session window, 7-day weekly window (both as `utilization` 0-100 float)
- Reset times: RFC3339 strings
- Also fetches profile email via `/api/oauth/profile`
- Shows extra usage credits remaining if enabled on account

**Codex** (`internal/provider/codex.go`)
- Auth: Bearer token from `~/.codex/auth.json`
- Returns: primary window (5h), secondary window (7d) as `used_percent` 0-100
- Reset times: **unix timestamps** (not RFC3339)
- Shows `limit_reached` as critical status

**OpenRouter** (`internal/provider/openrouter.go`)
- Auth: `$OPENROUTER_API_KEY` env var
- Returns: daily/weekly/monthly/all-time spend, optional budget limit + remaining credits
- Label field may be the raw API key itself — filtered out (skipped if starts with `sk-`)
- No rate windows unless a budget limit is configured

### Waybar Output

- Single icon: `󱚣` (nerd font) + worst-case percentage across all providers
- CSS class: `normal` / `warning` (>=75%) / `critical` (>=90%)
- No tooltip — detail is click-only via yad popup

### Detail Popup (yad)

- 420x380px, centered, undecorated, no buttons, skip-taskbar
- Catppuccin Frappe theme (bg: `#303446`, text: `#c6d0f5`)
- Per-provider sections with colored headers (Claude: purple, Codex: green, OpenRouter: teal)
- Horizontal progress bars with color-coded fill (green/yellow/red)
- Reset time countdowns
- Spend breakdowns (daily/weekly/monthly/all-time)
- Credits remaining
- Close with `q`, `Q`, or `Escape` key
- Sway float rules via `app_id="ai-usage-detail"` and `title="AI Usage"`

## Color Scheme (Catppuccin Frappe)

| Element | Color | Hex |
|---|---|---|
| Background | Base | `#303446` |
| Text | Text | `#c6d0f5` |
| Subtext | Subtext0 | `#a5adce` |
| Overlay | Surface0 | `#414559` |
| Muted | Overlay0 | `#737994` |
| Normal/Good | Green | `#a6d189` |
| Warning | Yellow | `#e5c890` |
| Critical/Error | Red | `#e78284` |
| Claude header | Mauve | `#ca9ee6` |
| Codex header | Green | `#a6d189` |
| OpenRouter header | Teal | `#81c8be` |
| Waybar icon (normal) | Teal | `#81c8be` |

## External Config Files (not in this repo)

These live in `~/dotfiles/stow/` managed via GNU stow:

**Waybar config** (`~/dotfiles/stow/waybar/.config/waybar/config`):
```json
"custom/ai_usage": {
  "exec": "~/.local/bin/ai-usage-bar",
  "return-type": "json",
  "interval": 120,
  "tooltip": false,
  "on-click": "~/.local/bin/ai-usage-bar --detail"
}
```

**Waybar CSS** (`~/dotfiles/stow/waybar/.config/waybar/style.css`):
```css
#custom-ai_usage { color: #81c8be; }
#custom-ai_usage.warning { color: #e5c890; }
#custom-ai_usage.critical { color: #e78284; }
```

**Sway config** (`~/dotfiles/stow/sway/.config/sway/config`):
```
for_window [app_id="ai-usage-detail"] floating enable
for_window [title="AI Usage"] floating enable
```

## Project Structure

```
cmd/ai-usage-bar/main.go         Entry point, cache-then-fetch, mode dispatch
internal/
  cache/cache.go                  File-based JSON cache (1h TTL)
  provider/
    provider.go                   Provider interface, FetchAll, Result/RateWindow/SpendEntry types
    claude.go                     Claude OAuth API
    codex.go                      Codex (ChatGPT) API
    openrouter.go                 OpenRouter API
  detail/detail.go                yad HTML popup renderer
  waybar/waybar.go                Waybar JSON formatter
Taskfile.yml                      go-task build system
AGENTS.md                         Agent instructions
```

## API Gotchas & Lessons Learned

- Claude uses `utilization` not `percent_used`, `resets_at` not `reset_at`
- Codex reset times are unix timestamps, Claude reset times are RFC3339 strings
- OpenRouter `/api/v1/key` label field may just be the API key itself — don't display it
- yad `--class` sets app_id for Sway float rules, but also add title match as fallback
- yad `--html` supports full CSS/JS — use `window.close()` from keydown events
- Always check raw API responses before writing parsers — field names vary between providers
- Tokens can expire; currently shows `!` for auth failures

## TODOs

- [ ] Handle expired/invalid tokens gracefully (currently shows `!` or `?`)
- [ ] Add token refresh for Claude OAuth (tokens expire)
- [ ] Config file for enabling/disabling providers
- [ ] Local session cost tracking from JSONL logs
- [ ] Nix flake for packaging
