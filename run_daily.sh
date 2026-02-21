#!/bin/zsh
# run_daily.sh
# Shell wrapper for launchd — sources zsh profile to pick up correct PATH
# before running whoop-garden daily.

source /Users/benstrawbridge/.zprofile 2>/dev/null || true
source /Users/benstrawbridge/.zshrc 2>/dev/null || true

cd /Volumes/wanderer/dev/solo/obsidian-whoop-garden
exec ./whoop-garden daily
