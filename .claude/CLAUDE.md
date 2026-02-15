# ai-usage-bar

Waybar module for monitoring AI provider usage. Go binary, Catppuccin Frappe themed.

## Quick Reference

- `task build` — build binary
- `task install` — install to `~/.local/bin/ai-usage-bar`
- `task dev` — run with `go run`
- `ai-usage-bar` — waybar JSON mode
- `ai-usage-bar --detail` — yad popup mode

## Key Files

- Waybar config: `~/dotfiles/stow/waybar/.config/waybar/config` (custom/ai_usage module)
- Waybar CSS: `~/dotfiles/stow/waybar/.config/waybar/style.css` (ai_usage classes)
- Sway config: `~/dotfiles/stow/sway/.config/sway/config` (float rules for popup)

## Lessons Learned

- Always check raw API responses before writing parsers — field names vary (Claude uses `utilization` not `percent_used`, `resets_at` not `reset_at`)
- Codex reset times are unix timestamps, Claude reset times are RFC3339 strings
- OpenRouter `/api/v1/key` label field may just be the API key itself — don't display it
- yad `--close-on-unfocus` works with Sway's focus-follows-mouse for hover-to-dismiss
- yad `--class` sets app_id for Sway float rules, but also add title match as fallback
- yad `--html` supports full CSS/JS — use `window.close()` from keydown events for `q` to close
- User prefers single combined icon over separate per-provider modules
- User wants yad popup over terminal for detail view

## TODOs

- [ ] Handle expired/invalid tokens gracefully (currently shows `!` or `?`)
- [ ] Add token refresh for Claude OAuth (tokens expire)
- [ ] Config file for enabling/disabling providers
- [ ] Local session cost tracking from JSONL logs
- [ ] Nix flake for packaging
