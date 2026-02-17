#!/usr/bin/env bash
set -euo pipefail

REPO_PACKAGE="github.com/jhartzell/ai-usage-bar/cmd/ai-usage-bar"
VERSION="latest"
INSTALL_DIR="${AI_USAGE_BAR_INSTALL_DIR:-$HOME/.local/bin}"

usage() {
  cat <<'EOF'
Install ai-usage-bar using go install.

Usage:
  install.sh [--version <tag>] [--dir <path>]

Options:
  --version <tag>  Version tag/commit (default: latest)
  --dir <path>     Install directory (default: ~/.local/bin)
  -h, --help       Show this help

Examples:
  install.sh
  install.sh --version v0.1.0
  install.sh --dir "$HOME/bin"
EOF
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command not found: $1" >&2
    exit 1
  fi
}

check_go_version() {
  local goversion cleaned major minor

  goversion="$(go env GOVERSION 2>/dev/null || true)"
  if [[ -z "$goversion" ]]; then
    goversion="$(go version | awk '{print $3}')"
  fi

  cleaned="${goversion#go}"
  cleaned="${cleaned%%[^0-9.]*}"
  major="${cleaned%%.*}"
  minor="${cleaned#*.}"
  minor="${minor%%.*}"

  if [[ -z "$major" || -z "$minor" ]]; then
    echo "error: unable to detect Go version from: $goversion" >&2
    exit 1
  fi

  if (( major < 1 || (major == 1 && minor < 25) )); then
    echo "error: Go 1.25+ is required (found $goversion)" >&2
    exit 1
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    --dir)
      INSTALL_DIR="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if [[ -z "$VERSION" ]]; then
  echo "error: --version requires a value" >&2
  exit 1
fi

if [[ -z "$INSTALL_DIR" ]]; then
  echo "error: --dir requires a value" >&2
  exit 1
fi

if [[ "$VERSION" != "latest" && "$VERSION" != v* ]]; then
  VERSION="v$VERSION"
fi

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "warning: ai-usage-bar is primarily intended for Linux + Waybar." >&2
fi

require_cmd go
check_go_version

mkdir -p "$INSTALL_DIR"

echo "Installing ai-usage-bar@$VERSION to $INSTALL_DIR ..."
GOBIN="$INSTALL_DIR" go install "${REPO_PACKAGE}@${VERSION}"

if [[ ! -x "$INSTALL_DIR/ai-usage-bar" ]]; then
  echo "error: install completed but binary not found at $INSTALL_DIR/ai-usage-bar" >&2
  exit 1
fi

echo "Installed: $INSTALL_DIR/ai-usage-bar"

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo ""
    echo "Add this to your shell profile to use ai-usage-bar directly:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    ;;
esac

if command -v pgrep >/dev/null 2>&1 && pgrep -x waybar >/dev/null 2>&1; then
  if command -v pkill >/dev/null 2>&1; then
    pkill -SIGUSR2 waybar || true
    echo "Reloaded Waybar."
  fi
fi

echo "Done. Run: ai-usage-bar"
