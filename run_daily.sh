#!/bin/zsh
# run_daily.sh
# Shell wrapper for launchd — sources zsh profile to pick up correct PATH
# before running whoop-garden daily.

source "$HOME/.zprofile" 2>/dev/null || true
source "$HOME/.zshrc" 2>/dev/null || true

cd "$(dirname "$0")"
exec ./whoop-garden catch-up --days 30
