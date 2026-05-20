#!/usr/bin/env bash
# dlv-session.sh — Drive a single interactive dlv (Delve) debugging session
# through tmux, designed for an agent that calls one subcommand at a time.
#
# State persists between invocations in a small file under TMPDIR so the
# caller does not need to track tmux session names or output paths.
#
# Subcommands:
#   start <dlv args>     Start dlv. Examples below.
#   send <dlv command>   Send a dlv command, wait for the next (dlv) prompt,
#                        and print only the new output.
#   send-raw <text>      Send keys without waiting for a prompt (for
#                        confirmations such as "y" to dlv's "Would you like
#                        to kill the process?").
#   output [N]           Print last N lines (default 100) of raw output.
#   status               Show whether a session is active.
#   stop                 Quit dlv and clean up the tmux session.
#
# Examples:
#   dlv-session.sh start debug .
#   dlv-session.sh start debug ./cmd/foo -- --flag=value
#   dlv-session.sh start test ./pkg -- -test.run TestMyBug
#   dlv-session.sh start attach 12345
#   dlv-session.sh start attach 12345 ./bin/myserver
#   dlv-session.sh start connect 127.0.0.1:2345
#
#   dlv-session.sh send 'break main.findMax'
#   dlv-session.sh send 'continue'
#   dlv-session.sh send 'print numbers'
#   dlv-session.sh send 'goroutines'
#   dlv-session.sh stop
#
# Implementation notes:
#   - dlv is launched as the tmux pane's initial command (not via an
#     interactive shell), so zsh/bash autosuggestion plugins do not corrupt
#     the input/output stream.
#   - All pane output is captured to a file via `tmux pipe-pane`. This file
#     grows monotonically, so byte-offset slicing gives exact "new output
#     since last command".
#   - Prompt detection uses `tmux capture-pane` (visible pane only), so the
#     "(dlv) " prompt at the bottom of the pane is reliably the last line.

set -euo pipefail

STATE_DIR="${TMPDIR:-/tmp}/claude-dlv"
mkdir -p "$STATE_DIR"
STATE_FILE="$STATE_DIR/current"

# Strip ANSI escape codes and carriage returns from a stream.
strip_ansi() {
    sed $'s/\x1b\\[[0-9;?]*[a-zA-Z]//g; s/\r//g'
}

# Wait until the visible pane's last non-blank line contains a (dlv) prompt.
# Optionally also require the output file to have grown past min_size so a
# stale prompt from before the command was sent does not falsely match.
# Args: <session> [timeout_seconds] [min_size_bytes] [output_file]
wait_for_prompt() {
    local session="$1"
    local timeout="${2:-30}"
    local min_size="${3:-0}"
    local file="${4:-}"
    local start_time
    start_time=$(date +%s)

    while :; do
        if tmux has-session -t "$session" 2>/dev/null; then
            local size_ok=1
            if [[ -n "$file" ]] && (( min_size > 0 )); then
                local cur_size
                cur_size=$(wc -c < "$file" 2>/dev/null | tr -d ' ' || echo 0)
                if (( cur_size <= min_size )); then
                    size_ok=0
                fi
            fi
            if (( size_ok )); then
                local tail_lines
                tail_lines=$(tmux capture-pane -t "$session" -p 2>/dev/null \
                    | strip_ansi | awk 'NF' | tail -3)
                if printf '%s' "$tail_lines" | grep -qE '\(dlv\)[[:space:]]*$'; then
                    return 0
                fi
            fi
        else
            return 2
        fi
        local now
        now=$(date +%s)
        if (( now - start_time >= timeout )); then
            return 1
        fi
        sleep 0.2
    done
}

require_session() {
    if [[ ! -f "$STATE_FILE" ]]; then
        echo "Error: no active session. Run 'start <args>' first." >&2
        exit 1
    fi
    # shellcheck disable=SC1090
    source "$STATE_FILE"
    if ! tmux has-session -t "$session" 2>/dev/null; then
        echo "Error: session $session no longer running (clearing stale state)" >&2
        rm -f "$STATE_FILE"
        exit 1
    fi
}

cmd_start() {
    if [[ -f "$STATE_FILE" ]]; then
        # shellcheck disable=SC1090
        source "$STATE_FILE"
        if tmux has-session -t "${session:-}" 2>/dev/null; then
            echo "Error: session $session already active. Run 'stop' first." >&2
            exit 1
        fi
        rm -f "$STATE_FILE"
    fi

    if ! command -v dlv >/dev/null 2>&1; then
        echo "Error: dlv not found in PATH." >&2
        echo "Install with: go install github.com/go-delve/delve/cmd/dlv@latest" >&2
        exit 1
    fi
    if ! command -v tmux >/dev/null 2>&1; then
        echo "Error: tmux not found in PATH." >&2
        exit 1
    fi

    if (( $# == 0 )); then
        echo "Error: 'start' requires dlv arguments (e.g., 'debug .', 'test ./pkg', 'attach <pid>')." >&2
        exit 1
    fi

    local stamp
    stamp=$(date +%s)
    local session="claude-dlv-${stamp}-$$"
    local output_file="$STATE_DIR/$session.out"
    : > "$output_file"

    # Quote each arg for the shell that tmux spawns to launch dlv.
    local dlv_cmd
    printf -v dlv_cmd 'exec dlv %q' "$1"; shift
    for a in "$@"; do
        printf -v dlv_cmd '%s %q' "$dlv_cmd" "$a"
    done

    tmux new-session -d -s "$session" -c "$(pwd)" -x 220 -y 60 \
        "sh -c '$dlv_cmd'"
    tmux set-option -t "$session" history-limit 100000 >/dev/null 2>&1 || true
    tmux pipe-pane -t "$session" -o "cat >> '$output_file'"

    local rc
    set +e
    wait_for_prompt "$session" 120
    rc=$?
    set -e
    if (( rc != 0 )); then
        echo "Error: dlv did not produce a (dlv) prompt within 120s (rc=$rc)" >&2
        echo "--- pane snapshot ---" >&2
        tmux capture-pane -t "$session" -p 2>/dev/null | strip_ansi | tail -50 >&2
        tmux kill-session -t "$session" 2>/dev/null || true
        rm -f "$output_file"
        exit 1
    fi

    cat > "$STATE_FILE" <<EOF
session=$session
output=$output_file
EOF

    echo "Session: $session"
    echo "Output:  $output_file"
    echo "---"
    # Show initial pane content (covers the case where pipe-pane attached
    # after dlv printed the first prompt).
    tmux capture-pane -t "$session" -p 2>/dev/null | strip_ansi | awk 'NF'
}

cmd_send() {
    require_session
    local cmd="$*"
    if [[ -z "$cmd" ]]; then
        echo "Error: empty command" >&2
        exit 1
    fi

    local before_size
    before_size=$(wc -c < "$output" 2>/dev/null | tr -d ' ' || echo 0)

    tmux send-keys -t "$session" -l "$cmd"
    tmux send-keys -t "$session" Enter

    local timeout="${DLV_SEND_TIMEOUT:-300}"
    local rc
    set +e
    wait_for_prompt "$session" "$timeout" "$before_size" "$output"
    rc=$?
    set -e
    if (( rc != 0 )); then
        echo "Error: command did not return to (dlv) prompt within ${timeout}s (rc=$rc)" >&2
        echo "--- partial output ---" >&2
        tail -c +"$((before_size + 1))" "$output" | strip_ansi >&2
        exit 1
    fi

    # Pipe-pane may have a small delay flushing to the file; allow one more
    # poll so the last line lands before we slice.
    local final_byte_size=0
    for _ in 1 2 3 4 5; do
        local cur
        cur=$(wc -c < "$output" 2>/dev/null | tr -d ' ' || echo 0)
        if (( cur > before_size )) && (( cur == final_byte_size )); then
            break
        fi
        final_byte_size=$cur
        sleep 0.1
    done

    tail -c +"$((before_size + 1))" "$output" | strip_ansi
}

cmd_send_raw() {
    require_session
    local cmd="$*"
    tmux send-keys -t "$session" -l "$cmd"
    tmux send-keys -t "$session" Enter
    sleep 0.3
}

cmd_output() {
    require_session
    local n="${1:-100}"
    tail -n "$n" "$output" | strip_ansi
}

cmd_status() {
    if [[ ! -f "$STATE_FILE" ]]; then
        echo "No active session"
        return 0
    fi
    # shellcheck disable=SC1090
    source "$STATE_FILE"
    if tmux has-session -t "$session" 2>/dev/null; then
        echo "Active session: $session"
        echo "Output file:    $output"
        local size
        size=$(wc -c < "$output" 2>/dev/null | tr -d ' ')
        echo "Output bytes:   ${size}"
    else
        echo "Session $session no longer running (stale state file)"
    fi
}

cmd_stop() {
    if [[ ! -f "$STATE_FILE" ]]; then
        echo "No active session"
        return 0
    fi
    # shellcheck disable=SC1090
    source "$STATE_FILE"

    if tmux has-session -t "$session" 2>/dev/null; then
        tmux send-keys -t "$session" -l "quit" 2>/dev/null || true
        tmux send-keys -t "$session" Enter 2>/dev/null || true
        sleep 0.4
        # dlv may prompt: "Would you like to kill the process? [Y/n]"
        tmux send-keys -t "$session" -l "y" 2>/dev/null || true
        tmux send-keys -t "$session" Enter 2>/dev/null || true
        sleep 0.3
        tmux kill-session -t "$session" 2>/dev/null || true
    fi

    rm -f "${output:-}" "$STATE_FILE"
    echo "Session stopped"
}

usage() {
    sed -n '2,38p' "$0"
}

case "${1:-}" in
    start)    shift; cmd_start "$@" ;;
    send)     shift; cmd_send "$@" ;;
    send-raw) shift; cmd_send_raw "$@" ;;
    output)   shift; cmd_output "$@" ;;
    status)   cmd_status ;;
    stop)     cmd_stop ;;
    -h|--help|help|"") usage ;;
    *)        usage; exit 1 ;;
esac
