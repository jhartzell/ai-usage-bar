# Changelog

All notable changes to this project are documented in this file.

## [0.1.1] - 2026-02-17

### Added
- One-line installer script (`install.sh`) with version pinning support.
- Comprehensive regression test suite for cache, providers, popup rendering, and Waybar formatting.

### Changed
- Detail popup renderer refactored to use an embedded HTML template (`internal/detail/popup.html.tmpl`) for maintainability.
- README install docs now include one-liner installer usage and HTTPS clone examples.

### Fixed
- Reduced risk of popup rendering regressions by moving dynamic content into structured view-model mapping with escaping.

## [0.1.0] - 2026-02-16

### Added
- Waybar module output with click-to-open detail popup.
- Concurrent provider fetches for Claude, Codex, and OpenRouter.
- File cache with 1 hour TTL to reduce API calls.
- MIT license and open-source-safe defaults in `.gitignore`.

### Changed
- Polished detail popup layout, sizing, and provider sections.
- README rewritten to be concise and quick to set up.

### Fixed
- Automatic OAuth recovery for stale Claude and Codex sessions.
- Clear auth recovery behavior (`claude login` / `codex login`) when refresh fails.
- Cache now ignores cached provider errors to avoid stale failure states.
