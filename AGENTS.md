# Agent Instructions

## Package Manager

- Go 1.25+, no external dependencies (stdlib only)
- `go mod tidy` after adding imports

## Build & Test

- `task build` — compile binary to `./ai-usage-bar`
- `task install` — compile and install to `~/.local/bin/ai-usage-bar`
- `task dev` — run via `go run`
- `task fmt` — format with gofmt
- `task lint` — run golangci-lint
- No test suite exists yet; verify changes by building and running

## Repo Layout

```
cmd/ai-usage-bar/main.go    Entry point, arg parsing, cache-then-fetch flow
internal/
  cache/cache.go             File-based JSON cache (~/.cache/ai-usage-bar/cache.json, 1h TTL)
  provider/provider.go       Provider interface, FetchAll (concurrent), Result types
  provider/claude.go         Claude OAuth usage + profile API
  provider/codex.go          Codex (ChatGPT) usage API
  provider/openrouter.go     OpenRouter key info API
  detail/detail.go           yad HTML popup for --detail mode
  waybar/waybar.go           Waybar JSON output formatting
```

## Key Conventions

- Providers implement `provider.Provider` interface (`Name()`, `Fetch(ctx)`)
- `FetchAll` runs all providers concurrently with 5s timeout each
- Cache is checked before any API calls; only fetches when cache is >1h old or missing
- Two modes: default outputs waybar JSON, `--detail` opens yad popup
- Catppuccin Frappe color scheme for all UI
- No tooltip on waybar icon; detail is click-only (yad popup)

## External Config (not in this repo)

- Waybar module: `~/dotfiles/stow/waybar/.config/waybar/config` (`custom/ai_usage`)
- Waybar CSS: `~/dotfiles/stow/waybar/.config/waybar/style.css`
- Sway float rules: `~/dotfiles/stow/sway/.config/sway/config`

## API Gotchas

- Claude OAuth creds: `~/.claude/.credentials.json` — fields are `utilization` and `resets_at`
- Codex auth: `~/.codex/auth.json` — reset times are **unix timestamps**, not RFC3339
- OpenRouter: `$OPENROUTER_API_KEY` env var — label field may be raw API key, skip display

## Safety & Scope

- Never commit credential files or API keys
- Keep the binary zero-dependency (stdlib only)
- Changes to waybar/sway config are in `~/dotfiles/stow/`, managed via GNU stow
