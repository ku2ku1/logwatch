#!/bin/bash
set -e

echo "=== Logvance Installer ==="

# Check root
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root: sudo bash install.sh"
  exit 1
fi

# Create user
echo "[1/6] Creating logvance user..."
id -u logvance &>/dev/null || useradd -r -s /bin/false -d /opt/logvance logvance
usermod -aG adm logvance  # log read access

# Create directories
echo "[2/6] Creating directories..."
mkdir -p /opt/logvance/{bin,data,config}

# Copy files
echo "[3/6] Installing files..."
cp bin/logvance /opt/logvance/bin/
cp deploy/config.yaml /opt/logvance/config/
chmod +x /opt/logvance/bin/logvance

# Generate JWT secret
JWT_SECRET=$(openssl rand -hex 32)
sed -i "s/CHANGE_THIS_TO_RANDOM_SECRET/$JWT_SECRET/" deploy/logvance.service
echo "JWT_SECRET saved to /opt/logvance/config/.env"
echo "JWT_SECRET=$JWT_SECRET" > /opt/logvance/config/.env
chmod 600 /opt/logvance/config/.env

# Permissions
echo "[4/6] Setting permissions..."
chown -R logvance:logvance /opt/logvance
chmod 750 /opt/logvance/data

# Systemd
echo "[5/6] Installing systemd service..."
cp deploy/logvance.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable logvance
systemctl start logvance

# Nginx
echo "[6/6] Configuring nginx..."
cp deploy/nginx-logvance.conf /etc/nginx/sites-available/logvance
ln -sf /etc/nginx/sites-available/logvance /etc/nginx/sites-enabled/logvance
rm -f /etc/nginx/sites-enabled/default
nginx -t && systemctl reload nginx

echo ""
echo "=== Installation Complete! ==="
echo "Dashboard: http://$(curl -s ifconfig.me 2>/dev/null || echo 'your-vps-ip')"
echo "Service status: systemctl status logvance"
echo ""
echo "IMPORTANT: Create admin user first:"
echo "curl -X POST http://localhost:8080/api/auth/setup \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d '{\"username\":\"admin\",\"password\":\"YOUR_STRONG_PASSWORD\"}'"
