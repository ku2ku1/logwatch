#!/bin/bash
set -e

echo "=== LogWatch Installer ==="

# Check root
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root: sudo bash install.sh"
  exit 1
fi

# Create user
echo "[1/6] Creating logwatch user..."
id -u logwatch &>/dev/null || useradd -r -s /bin/false -d /opt/logwatch logwatch
usermod -aG adm logwatch  # log read access

# Create directories
echo "[2/6] Creating directories..."
mkdir -p /opt/logwatch/{bin,data,config}

# Copy files
echo "[3/6] Installing files..."
cp bin/logwatch /opt/logwatch/bin/
cp deploy/config.yaml /opt/logwatch/config/
chmod +x /opt/logwatch/bin/logwatch

# Generate JWT secret
JWT_SECRET=$(openssl rand -hex 32)
sed -i "s/CHANGE_THIS_TO_RANDOM_SECRET/$JWT_SECRET/" deploy/logwatch.service
echo "JWT_SECRET saved to /opt/logwatch/config/.env"
echo "JWT_SECRET=$JWT_SECRET" > /opt/logwatch/config/.env
chmod 600 /opt/logwatch/config/.env

# Permissions
echo "[4/6] Setting permissions..."
chown -R logwatch:logwatch /opt/logwatch
chmod 750 /opt/logwatch/data

# Systemd
echo "[5/6] Installing systemd service..."
cp deploy/logwatch.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable logwatch
systemctl start logwatch

# Nginx
echo "[6/6] Configuring nginx..."
cp deploy/nginx-logwatch.conf /etc/nginx/sites-available/logwatch
ln -sf /etc/nginx/sites-available/logwatch /etc/nginx/sites-enabled/logwatch
rm -f /etc/nginx/sites-enabled/default
nginx -t && systemctl reload nginx

echo ""
echo "=== Installation Complete! ==="
echo "Dashboard: http://$(curl -s ifconfig.me 2>/dev/null || echo 'your-vps-ip')"
echo "Service status: systemctl status logwatch"
echo ""
echo "IMPORTANT: Create admin user first:"
echo "curl -X POST http://localhost:8080/api/auth/setup \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d '{\"username\":\"admin\",\"password\":\"YOUR_STRONG_PASSWORD\"}'"
