# ai-usage-bar Design

Go binary for waybar. Shows Claude, Codex, OpenRouter usage in one combined module.

## Output

- **text**: `C:42% X:67% OR:$1.23`
- **tooltip**: detailed per-provider breakdown with reset times
- **class**: normal/warning(>75%)/critical(>90%)

## APIs

| Provider | Credential | Endpoint | Auth |
|----------|-----------|----------|------|
| Claude | `~/.claude/.credentials.json` → `claudeAiOauth.accessToken` | `GET https://api.anthropic.com/api/oauth/usage` | Bearer + `anthropic-beta: oauth-2025-04-20` |
| Codex | `~/.codex/auth.json` → `tokens.access_token` | `GET https://chatgpt.com/backend-api/wham/usage` | Bearer |
| OpenRouter | `$OPENROUTER_API_KEY` | `GET https://openrouter.ai/api/v1/key` | Bearer |

## Structure

```
cmd/ai-usage-bar/main.go
internal/provider/provider.go   — interface + parallel fetch
internal/provider/claude.go
internal/provider/codex.go
internal/provider/openrouter.go
internal/waybar/waybar.go       — JSON output formatting
Taskfile.yml
```

## Error handling

- 5s timeout per provider
- Failed provider → `?` in text, skip in tooltip
- Missing creds → skip provider
- Expired token → `!` indicator
