#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log()  { echo -e "${GREEN}[✓]${NC} $1"; }
warn() { echo -e "${YELLOW}[!]${NC} $1"; }
err()  { echo -e "${RED}[✗]${NC} $1"; exit 1; }
info() { echo -e "${BLUE}[→]${NC} $1"; }

echo ""
echo "╔══════════════════════════════════════════╗"
echo "║     Logvance — VPS Log Analyzer          ║"
echo "║     Universal Installer v1.0             ║"
echo "╚══════════════════════════════════════════╝"
echo ""

# Root check
[ "$EUID" -ne 0 ] && err "Run as root: sudo bash install.sh"

# OS check
OS=$(lsb_release -si 2>/dev/null || echo "Unknown")
ARCH=$(uname -m)
log "OS: $OS | Arch: $ARCH | Kernel: $(uname -r)"

# ── STEP 1: Detect environment ──────────────────
info "Scanning VPS environment..."

NGINX_RUNNING=false
APACHE_RUNNING=false
FAIL2BAN_RUNNING=false
UFW_ACTIVE=false
DOCKER_RUNNING=false
HAS_SSL=false
DOMAIN=""

systemctl is-active --quiet nginx 2>/dev/null && NGINX_RUNNING=true
systemctl is-active --quiet apache2 2>/dev/null && APACHE_RUNNING=true
systemctl is-active --quiet fail2ban 2>/dev/null && FAIL2BAN_RUNNING=true
systemctl is-active --quiet docker 2>/dev/null && DOCKER_RUNNING=true
ufw status 2>/dev/null | grep -q "Status: active" && UFW_ACTIVE=true

# Domain detect
if [ -d "/etc/letsencrypt/live" ]; then
    DOMAIN=$(ls /etc/letsencrypt/live/ 2>/dev/null | grep -v README | head -1)
    [ -n "$DOMAIN" ] && HAS_SSL=true
fi

echo ""
echo "  Detected services:"
$NGINX_RUNNING    && echo "    ✓ Nginx"
$APACHE_RUNNING   && echo "    ✓ Apache"
$FAIL2BAN_RUNNING && echo "    ✓ Fail2ban"
$UFW_ACTIVE       && echo "    ✓ UFW Firewall"
$DOCKER_RUNNING   && echo "    ✓ Docker"
$HAS_SSL          && echo "    ✓ SSL ($DOMAIN)"
echo ""

# ── STEP 2: Install dependencies ────────────────
info "Installing dependencies..."
apt-get update -qq
apt-get install -y -qq curl wget git make gcc g++ nginx 2>/dev/null || true

# Go install
if ! command -v go &>/dev/null; then
    info "Installing Go..."
    GO_VERSION="1.22.5"
    case $ARCH in
        x86_64)  GO_ARCH="amd64" ;;
        aarch64) GO_ARCH="arm64" ;;
        armv7l)  GO_ARCH="armv6l" ;;
        *)       err "Unsupported arch: $ARCH" ;;
    esac
    wget -q "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" -O /tmp/go.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/go.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile.d/go.sh
    export PATH=$PATH:/usr/local/go/bin
fi
log "Go: $(go version)"

# Node.js
if ! command -v node &>/dev/null; then
    info "Installing Node.js..."
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash - 2>/dev/null
    apt-get install -y -qq nodejs
fi
log "Node: $(node --version)"

# ── STEP 3: Create user & directories ───────────
info "Creating logvance user..."
id -u logvance &>/dev/null || useradd -r -s /bin/false -d /opt/logvance logvance
usermod -aG adm logvance 2>/dev/null || true
usermod -aG systemd-journal logvance 2>/dev/null || true
$DOCKER_RUNNING && usermod -aG docker logvance 2>/dev/null || true

mkdir -p /opt/logvance/{bin,data,data/geoip,config,logs}
log "Directories created"

# ── STEP 4: Clone & build ───────────────────────
info "Downloading Logvance..."
if [ -d "/opt/logvance/src" ]; then
    cd /opt/logvance/src && git pull -q
else
    git clone -q https://github.com/ku2ku1/logvance /opt/logvance/src
fi

cd /opt/logvance/src

info "Building frontend..."
cd frontend
npm install --silent
npm run build --silent
cp -r dist ../internal/api/dist
cd ..

info "Building binary..."
CGO_ENABLED=1 go build -ldflags="-s -w" -o /opt/logvance/bin/logvance ./cmd/logvance
log "Binary built: $(ls -lh /opt/logvance/bin/logvance | awk '{print $5}')"

# ── STEP 5: GeoIP database ──────────────────────
info "Downloading GeoIP database..."
wget -q "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb" \
    -O /opt/logvance/data/geoip/GeoLite2-City.mmdb && log "GeoIP ready" || warn "GeoIP download failed (optional)"

# ── STEP 6: Auto-configure based on environment ─
info "Auto-configuring for your environment..."

NGINX_LOG="/var/log/nginx/access.log"
APACHE_LOG="/var/log/apache2/access.log"
AUTH_LOG="/var/log/auth.log"
PORT=8080

# Find available port if 8080 is taken
if ss -tlnp | grep -q ":8080 "; then
    PORT=8081
    warn "Port 8080 in use, using 8081"
fi

JWT_SECRET=$(openssl rand -hex 32)

cat > /opt/logvance/config/config.yaml << EOF
server:
  host: "127.0.0.1"
  port: $PORT

database:
  path: "/opt/logvance/data/logvance.db"

logs:
  nginx_access: "$NGINX_LOG"
  nginx_error: "/var/log/nginx/error.log"
  auth_log: "$AUTH_LOG"
  fail2ban_log: "/var/log/fail2ban.log"
  ufw_log: "/var/log/ufw.log"

geoip:
  path: "/opt/logvance/data/geoip/GeoLite2-City.mmdb"
EOF

cat > /opt/logvance/config/.env << EOF
JWT_SECRET=$JWT_SECRET
PORT=$PORT
EOF

chmod 600 /opt/logvance/config/.env
log "Config generated"

# ── STEP 7: Systemd service ──────────────────────
info "Installing systemd service..."
cat > /etc/systemd/system/logvance.service << EOF
[Unit]
Description=Logvance — Universal VPS Log Analyzer
After=network.target
StartLimitIntervalSec=0

[Service]
Type=simple
User=logvance
Group=logvance
WorkingDirectory=/opt/logvance
EnvironmentFile=/opt/logvance/config/.env
ExecStart=/opt/logvance/bin/logvance
Restart=always
RestartSec=5

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/opt/logvance/data /opt/logvance/logs
ReadOnlyPaths=/var/log /opt/logvance/config

StandardOutput=journal
StandardError=journal
SyslogIdentifier=logvance

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable logvance
systemctl start logvance
log "Service started"

# ── STEP 8: Nginx reverse proxy ─────────────────
info "Configuring Nginx..."

if $HAS_SSL && [ -n "$DOMAIN" ]; then
    # Check if domain already has nginx config
    if grep -r "logvance\|$PORT" /etc/nginx/sites-enabled/ &>/dev/null; then
        warn "Nginx already has entry for port $PORT, skipping"
    else
        cat > /etc/nginx/sites-available/logvance << EOF
# Logvance — added by installer
server {
    listen 8443 ssl http2;
    server_name $DOMAIN;

    ssl_certificate /etc/letsencrypt/live/$DOMAIN/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/$DOMAIN/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:$PORT;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-Proto https;
        proxy_read_timeout 86400;
    }
}
EOF
        ln -sf /etc/nginx/sites-available/logvance /etc/nginx/sites-enabled/
        nginx -t && systemctl reload nginx
        log "Nginx configured: https://$DOMAIN:8443"
    fi
else
    warn "No SSL found. Access via http://YOUR_IP:$PORT"
fi

# ── STEP 9: Firewall ─────────────────────────────
if $UFW_ACTIVE; then
    info "Configuring UFW..."
    ufw allow $PORT/tcp comment "Logvance" 2>/dev/null || true
    log "UFW: port $PORT allowed"
fi

# ── STEP 10: Permissions ─────────────────────────
chown -R logvance:logvance /opt/logvance
chmod +x /opt/logvance/bin/logvance

# ── STEP 11: Setup admin ─────────────────────────
echo ""
echo "╔══════════════════════════════════════════╗"
echo "║         Installation Complete!           ║"
echo "╚══════════════════════════════════════════╝"
echo ""

sleep 2

# Create admin account
# Admin credentials — environment variables ya defaults
ADMIN_USER="${LOGVANCE_ADMIN_USER:-admin}"
ADMIN_PASS="${LOGVANCE_ADMIN_PASS:-$(openssl rand -base64 12)}"
echo ""
echo "  Admin username: $ADMIN_USER"
echo "  Admin password: $ADMIN_PASS"
echo "  (Change after first login!)"

curl -s -X POST "http://127.0.0.1:$PORT/api/auth/setup" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\"}" | \
    grep -q "token" && log "Admin account created!" || warn "Setup failed — run manually"

echo ""
$HAS_SSL && echo -e "  ${GREEN}Dashboard:${NC} https://$DOMAIN:8443" || \
            echo -e "  ${GREEN}Dashboard:${NC} http://$(curl -s ifconfig.me 2>/dev/null || echo 'YOUR_IP'):$PORT"
echo -e "  ${GREEN}Status:${NC}    systemctl status logvance"
echo -e "  ${GREEN}Logs:${NC}      journalctl -u logvance -f"
echo ""
echo "  Run this to monitor: journalctl -u logvance -f"
echo ""
