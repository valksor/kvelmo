#!/bin/bash
# Desktop development script with proper terminal handling

cleanup() {
    echo ""
    echo "Shutting down..."
    # Kill all child processes
    pkill -P $$ 2>/dev/null
    # Reset terminal to sane state
    stty sane 2>/dev/null
    reset 2>/dev/null
    exit 0
}

trap cleanup SIGINT SIGTERM

# Kill any existing processes on our ports
lsof -ti:6337 | xargs kill -9 2>/dev/null
lsof -ti:5173 | xargs kill -9 2>/dev/null

cd "$(dirname "$0")/../web" || exit 1

# Run with colors disabled and filter remaining escape codes
# Use stdbuf to disable buffering, perl to strip ANSI codes
NO_COLOR=1 CARGO_TERM_COLOR=never FORCE_COLOR=0 \
    stdbuf -oL bun tauri dev --no-watch 2>&1 | \
    stdbuf -oL perl -pe 's/\e\[[0-9;]*[a-zA-Z]//g; s/\e\][^\a]*\a//g; s/\e[PX^_].*?\e\\//g'
