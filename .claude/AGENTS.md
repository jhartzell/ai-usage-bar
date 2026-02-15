# ai-usage-bar

Go binary for waybar that shows AI provider usage (Claude, Codex, OpenRouter) in a single combined module.

## Architecture

```
cmd/ai-usage-bar/main.go     — CLI entry, --detail flag for yad popup
internal/provider/            — Provider interface + parallel fetch
  provider.go                 — Result type, FetchAll(), shared helpers
  claude.go                   — OAuth usage + profile API
  codex.go                    — ChatGPT wham/usage API
  openrouter.go               — /api/v1/key endpoint
internal/waybar/waybar.go     — Waybar JSON output formatting
internal/detail/detail.go     — Yad HTML popup (Catppuccin Frappe themed)
Taskfile.yml                  — task build/install/run/dev
```

## How It Works

1. Binary runs, reads credentials from disk/env
2. Fetches all 3 APIs in parallel (goroutines, 5s timeout each)
3. Default mode: outputs waybar JSON to stdout (`{"text":"...", "tooltip":"...", "class":"...", "percentage":N}`)
4. `--detail` mode: opens a yad HTML popup with colored progress bars

## Credential Sources

| Provider | Source | API |
|----------|--------|-----|
| Claude | `~/.claude/.credentials.json` → `claudeAiOauth.accessToken` | `GET https://api.anthropic.com/api/oauth/usage` + `/api/oauth/profile` |
| Codex | `~/.codex/auth.json` → `tokens.access_token` | `GET https://chatgpt.com/backend-api/wham/usage` |
| OpenRouter | `$OPENROUTER_API_KEY` env var | `GET https://openrouter.ai/api/v1/key` |

## Waybar Integration

- Module: `custom/ai_usage` in `~/.config/waybar/config`
- CSS classes: `normal`, `warning` (>75%), `critical` (>90%) in `style.css`
- Sway float rules for `ai-usage-detail` app_id and `AI Usage` title
- Polls every 120s

## API Response Gotchas

- **Claude**: Uses `utilization` (not `percent_used`) and `resets_at` (not `reset_at`)
- **Codex**: Uses unix timestamps for `reset_at` (not RFC3339), nested under `rate_limit.primary_window`/`secondary_window`
- **OpenRouter**: No rate windows — purely spend-based. `label` field may contain the raw API key, skip if starts with `sk-`
- **Claude profile**: Separate endpoint at `/api/oauth/profile`, requires same OAuth token + `anthropic-beta` header

## Possible Future Features

- Token refresh (currently relies on CLI tools managing their own tokens)
- Local cost usage scanning (parsing session JSONL files for token accounting)
- Status page polling (Statuspage.io incidents for provider health)
- Configurable providers (enable/disable via config file)
- Cache API responses to reduce calls
- Nix package / flake for reproducible builds
