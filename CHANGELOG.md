# Changelog

All notable changes to this project are documented in this file.

## [0.2.0](https://github.com/jhartzell/ai-usage-bar/compare/v0.1.1...v0.2.0) (2026-02-17)


### Features

* add auth recovery and cache clear commands ([e36e317](https://github.com/jhartzell/ai-usage-bar/commit/e36e31720f64380ba9136d9021ecda1dd8d5f07f))
* **ui:** add popup button to recover auth ([4246934](https://github.com/jhartzell/ai-usage-bar/commit/424693421f6f804154b02cb6a20a455ec43cbee3))


### Bug Fixes

* **ui:** align popup close hint with Esc behavior ([4329ef8](https://github.com/jhartzell/ai-usage-bar/commit/4329ef853ebb2b73b48701255146e4e23046241d))

## [Unreleased]

### Added
- `--recover-auth` command to run provider login flows and clear cache in one step.
- `--clear-cache` command to remove cached usage data without manual file deletion.

### Changed
- Recovery guidance now prefers the single-command flow over manual steps.

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
