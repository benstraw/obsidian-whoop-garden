#!/bin/zsh
# run_weekly.sh
# Shell wrapper for launchd — sources zsh profile to pick up correct PATH
# before running whoop-garden weekly and persona refresh.

source /Users/benstrawbridge/.zprofile 2>/dev/null || true
source /Users/benstrawbridge/.zshrc 2>/dev/null || true

cd /Volumes/wanderer/dev/solo/obsidian-whoop-garden
./whoop-garden weekly
./whoop-garden persona
