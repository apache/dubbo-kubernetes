#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RUN_DIR="$ROOT_DIR/.run"

if [ -d "$RUN_DIR" ]; then
    for pidfile in "$RUN_DIR"/*.pid; do
        if [ -f "$pidfile" ]; then
            PID=$(cat "$pidfile" 2>/dev/null || true)
            if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null; then
                echo "Stopping process $PID ($(basename "$pidfile" .pid))..."
                pkill -P "$PID" 2>/dev/null || true
                kill "$PID" 2>/dev/null || true
                for _ in {1..10}; do
                    if ! kill -0 "$PID" 2>/dev/null; then
                        break
                    fi
                    sleep 0.2
                done
                if kill -0 "$PID" 2>/dev/null; then
                    kill -9 "$PID" 2>/dev/null || true
                fi
            fi
            rm -f "$pidfile"
        fi
    done
fi

# Ensure all workers on project ports are terminated
PORTS=(8080 9001 9002 9003 9004 9005 9006 9007 9008 9009)
for p in "${PORTS[@]}"; do
    PIDS=$(lsof -ti ":$p" 2>/dev/null || true)
    if [ -n "$PIDS" ]; then
        for pid in $PIDS; do
            kill -9 "$pid" 2>/dev/null || true
        done
    fi
done

echo "All services stopped."
