#!/usr/bin/env bash
# update_panel.sh — Pull latest code, rebuild backend + frontend, restart
# Usage: update_panel.sh
# This script is called by the panel's update endpoint.
# It runs as a detached child, so it survives the Go process restart.
set -euo pipefail

PANEL_DIR="/opt/panel"
LOG_FILE="/var/log/panel/update.log"

# Redirect all output to log file AND stdout (so the API can capture it)
exec > >(tee -a "$LOG_FILE") 2>&1

echo ""
echo "======================================"
echo "[update] $(date '+%Y-%m-%d %H:%M:%S') — Panel update started"
echo "======================================"

cd "$PANEL_DIR"

# ── 1. Pull latest code ───────────────────────────────────────────────────
echo ""
echo "[1/5] Pulling latest changes from GitHub..."
git fetch origin main 2>&1
LOCAL=$(git rev-parse HEAD)
REMOTE=$(git rev-parse origin/main)

if [ "$LOCAL" = "$REMOTE" ]; then
  echo "[update] Already up to date ($LOCAL)"
  echo "[update] Done — no changes."
  exit 0
fi

echo "[update] Updating from ${LOCAL:0:8} to ${REMOTE:0:8}"
git reset --hard origin/main 2>&1
echo "[1/5] Done."

# ── 2. Rebuild Go backend ─────────────────────────────────────────────────
echo ""
echo "[2/5] Building Go backend..."
export PATH="/usr/local/go/bin:$PATH"
cd "$PANEL_DIR/backend"
go build -o panel-server ./main.go 2>&1
echo "[2/5] Done — binary built successfully."

# ── 3. Rebuild frontend ───────────────────────────────────────────────────
echo ""
echo "[3/5] Building frontend..."
cd "$PANEL_DIR/frontend"
npm run build 2>&1
echo "[3/5] Done — frontend built successfully."

# ── 4. Copy scripts (in case they were updated) ──────────────────────────
echo ""
echo "[4/5] Updating scripts..."
chmod +x "$PANEL_DIR/scripts/"*.sh
echo "[4/5] Done."

# ── 5. Restart backend ────────────────────────────────────────────────────
echo ""
echo "[5/5] Restarting panel backend..."
pm2 restart panel-backend --update-env 2>&1
echo "[5/5] Done — backend restarted."

echo ""
echo "======================================"
NEW_HEAD=$(git rev-parse --short HEAD)
echo "[update] Update complete! Now running: $NEW_HEAD"
echo "[update] $(date '+%Y-%m-%d %H:%M:%S')"
echo "======================================"
