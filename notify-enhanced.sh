#!/bin/bash

# Enhanced notification script with context
# Usage: ./notify-enhanced.sh <sound> <message>

sound="$1"
message="$2"

# Get current directory
dir=$(pwd)
dir_name=$(basename "$dir")

# Get git branch if in a git repo
git_branch=""
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    git_branch=$(git branch --show-current 2>/dev/null || echo "detached")
    git_info=" (git: $git_branch)"
else
    git_info=""
fi

# Construct enhanced message
enhanced_message="[$dir_name$git_info] $message"

# Play sound and show alert
afplay "$sound" &
osascript -e "display alert \"Claude Code\" message \"$enhanced_message\""