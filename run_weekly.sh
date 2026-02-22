#!/bin/zsh
# run_weekly.sh
# Shell wrapper for launchd — sources zsh profile to pick up correct PATH
# before running whoop-garden weekly and persona refresh.

source "$HOME/.zprofile" 2>/dev/null || true
source "$HOME/.zshrc" 2>/dev/null || true

cd "$(dirname "$0")"
./whoop-garden weekly
./whoop-garden persona
