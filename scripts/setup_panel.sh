#!/usr/bin/env bash
# setup_panel.sh — Full one-shot installation of the control panel
# Run as root on a fresh Ubuntu 22.04+ server.
# Usage: bash setup_panel.sh
set -euo pipefail

PANEL_DIR="/opt/panel"
PANEL_DB_USER="${PANEL_DB_USER:-paneluser}"
PANEL_DB_PASS="${PANEL_DB_PASS:-$(openssl rand -hex 16)}"
PANEL_DB_NAME="${PANEL_DB_NAME:-panel}"
JWT_SECRET="$(openssl rand -hex 64)"

echo "========================================"
echo "  Panel — Server Control Panel Setup"
echo "========================================"

# ── 0. Must run as root ────────────────────────────────────────────────────
if [ "$EUID" -ne 0 ]; then
  echo "[error] Please run as root" >&2
  exit 1
fi

# ── 1. System dependencies ─────────────────────────────────────────────────
echo "[1/9] Installing system dependencies..."
apt-get update -qq
apt-get install -y curl git openssl ufw build-essential

# ── 2. Node.js LTS ────────────────────────────────────────────────────────
echo "[2/9] Installing Node.js LTS..."
if ! command -v node &>/dev/null; then
  curl -fsSL https://deb.nodesource.com/setup_lts.x | bash -
  apt-get install -y nodejs
fi
echo "  Node: $(node -v)  NPM: $(npm -v)"

# ── 3. PM2 ────────────────────────────────────────────────────────────────
echo "[3/9] Installing PM2..."
npm install -g pm2 --quiet
# Generate and run the startup command so PM2 survives reboots
env PATH="$PATH:/usr/bin" pm2 startup systemd -u root --hp /root 2>/dev/null | \
  grep "sudo" | bash || true

# ── 4. NGINX ──────────────────────────────────────────────────────────────
echo "[4/9] Installing NGINX..."
bash "${PANEL_DIR}/scripts/install_nginx.sh"

# ── 5. PostgreSQL ─────────────────────────────────────────────────────────
echo "[5/9] Installing PostgreSQL..."
PANEL_DB_USER="${PANEL_DB_USER}" \
PANEL_DB_PASS="${PANEL_DB_PASS}" \
PANEL_DB_NAME="${PANEL_DB_NAME}" \
  bash "${PANEL_DIR}/scripts/install_postgres.sh"

# ── 6. Redis ──────────────────────────────────────────────────────────────
echo "[6/9] Installing Redis..."
bash "${PANEL_DIR}/scripts/install_redis.sh"

# ── 7. Directories and permissions ────────────────────────────────────────
echo "[7/9] Setting up directories..."
mkdir -p /var/www/apps /var/log/panel
chmod 755 /var/www/apps
chmod +x "${PANEL_DIR}"/scripts/*.sh

# ── 8. Write .env files ───────────────────────────────────────────────────
echo "[8/9] Writing configuration..."

cat > "${PANEL_DIR}/backend/.env" <<EOF
PORT=4000
NODE_ENV=production
JWT_SECRET=${JWT_SECRET}
ADMIN_USERNAME=admin
ADMIN_PASSWORD=changeme
DATABASE_URL=postgresql://${PANEL_DB_USER}:${PANEL_DB_PASS}@localhost:5432/${PANEL_DB_NAME}
APPS_DIR=/var/www/apps
NGINX_AVAILABLE=/etc/nginx/sites-available
NGINX_ENABLED=/etc/nginx/sites-enabled
SCRIPTS_DIR=${PANEL_DIR}/scripts
APP_PORT_START=3001
APP_PORT_END=3999
PANEL_ORIGIN=http://localhost:3000
EOF

cat > "${PANEL_DIR}/frontend/.env.local" <<EOF
BACKEND_URL=http://127.0.0.1:4000
NEXT_PUBLIC_API_URL=http://127.0.0.1:4000
EOF

# ── 9. Build and start ────────────────────────────────────────────────────
echo "[9/9] Installing dependencies and building..."

# Backend
echo "  → Building backend..."
cd "${PANEL_DIR}/backend"
npm install
npm run build

# Verify build output exists
if [ ! -f "${PANEL_DIR}/backend/dist/index.js" ]; then
  echo "[error] Backend build failed — dist/index.js not found" >&2
  exit 1
fi

# Frontend
echo "  → Building frontend..."
cd "${PANEL_DIR}/frontend"
npm install
npm run build

# Verify .next exists
if [ ! -d "${PANEL_DIR}/frontend/.next" ]; then
  echo "[error] Frontend build failed — .next directory not found" >&2
  exit 1
fi

# ── Start with PM2 ────────────────────────────────────────────────────────
echo "[panel] Starting services with PM2..."

pm2 delete panel-backend  2>/dev/null || true
pm2 delete panel-frontend 2>/dev/null || true

# Backend — pass the .env file explicitly
pm2 start "${PANEL_DIR}/backend/dist/index.js" \
  --name panel-backend \
  --cwd "${PANEL_DIR}/backend" \
  --env production \
  --log /var/log/panel/backend.log \
  --merge-logs

# Frontend — use npm start so Next.js finds its .next dir correctly
pm2 start npm \
  --name panel-frontend \
  --cwd "${PANEL_DIR}/frontend" \
  --log /var/log/panel/frontend.log \
  --merge-logs \
  -- start

pm2 save

# Wait for services to come up before writing NGINX config
echo "[panel] Waiting for services to start..."
sleep 5

# Quick health check
if curl -sf http://127.0.0.1:4000/health >/dev/null; then
  echo "  ✓ Backend healthy"
else
  echo "  ✗ Backend not responding — check: pm2 logs panel-backend"
fi

if curl -sf http://127.0.0.1:3000 >/dev/null; then
  echo "  ✓ Frontend healthy"
else
  echo "  ✗ Frontend not responding — check: pm2 logs panel-frontend"
fi

# ── NGINX config (IP-based, no domain required) ────────────────────────────
echo "[panel] Writing NGINX config..."

SERVER_IP="$(curl -fsSL --connect-timeout 5 https://api.ipify.org 2>/dev/null || hostname -I | awk '{print $1}')"

cat > /etc/nginx/sites-available/panel <<NGINXCONF
# Panel control panel — accessible via server IP
# Generated by setup_panel.sh — do not edit manually

server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;

    client_max_body_size 100M;

    location / {
        proxy_pass         http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header   Upgrade \$http_upgrade;
        proxy_set_header   Connection 'upgrade';
        proxy_set_header   Host \$host;
        proxy_set_header   X-Real-IP \$remote_addr;
        proxy_set_header   X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;
        proxy_read_timeout 60s;
    }
}
NGINXCONF

ln -sf /etc/nginx/sites-available/panel /etc/nginx/sites-enabled/panel
rm -f /etc/nginx/sites-enabled/default

nginx -t && systemctl reload nginx

# ── Firewall ───────────────────────────────────────────────────────────────
echo "[panel] Configuring firewall..."
ufw allow OpenSSH    >/dev/null
ufw allow 'Nginx Full' >/dev/null
ufw --force enable   >/dev/null

echo ""
echo "========================================"
echo "  Panel installed successfully!"
echo ""
echo "  Open in browser: http://${SERVER_IP}"
echo ""
echo "  Login:"
echo "    Username: admin"
echo "    Password: changeme"
echo ""
echo "  Change your password after first login:"
echo "    nano ${PANEL_DIR}/backend/.env"
echo "    pm2 restart panel-backend"
echo ""
echo "  Useful commands:"
echo "    pm2 list                     — process status"
echo "    pm2 logs panel-backend       — backend logs"
echo "    pm2 logs panel-frontend      — frontend logs"
echo "    pm2 restart panel-backend    — restart backend"
echo "========================================"
