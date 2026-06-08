#!/bin/bash

# Play sound and show alert
# Usage: ./notify.sh <sound> <message>

sound="$1"
message="$2"

afplay "$sound" &
osascript -e "display alert \"Claude Code\" message \"$message\""