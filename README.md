# Logvance

Self-hosted, real-time VPS log analyzer — GoAccess se behtar

## Features

- **Real-time Dashboard**: WebSocket-powered live updates (<5s delay)
- **Multi-user Support**: Admin + Viewer + Client roles with JWT auth
- **Security Analysis**: SQLi, XSS, brute force detection with threat scoring
- **GeoIP Mapping**: World map visualization with MaxMind GeoLite2
- **Export Reports**: PDF, CSV, JSON exports
- **Production Ready**: Docker, systemd, nginx reverse proxy support

## Quick Start

### 1. Install Dependencies

```bash
# Go 1.22+
go version

# Node.js 18+
node --version
npm --version
```

### 2. Clone & Build

```bash
git clone https://github.com/yourusername/logvance.git
cd logvance

# Build backend
make build

# Install frontend deps
cd frontend && npm install && npm run build
cd ..
```

### 3. Configure

Edit `config.yaml`:

```yaml
server:
  host: 127.0.0.1
  port: 8080
database:
  path: ./data/logvance.db
logs:
  nginx_access: /var/log/nginx/access.log
  nginx_error: /var/log/nginx/error.log
```

### 4. Run

```bash
# Backend
./bin/logvance

# Frontend (in another terminal)
cd frontend && npm run dev
```

Open http://localhost:5174

### 5. Setup Admin User

POST to `/api/auth/setup`:

```json
{
  "username": "admin",
  "password": "yourpassword"
}
```

## Production Deployment

### Docker

```bash
docker-compose up -d
```

### Systemd

```bash
sudo useradd -r -s /bin/false logvance
sudo mkdir /opt/logvance
sudo cp bin/logvance /opt/logvance/
sudo cp config.yaml /opt/logvance/
sudo cp -r data /opt/logvance/
sudo chown -R logvance:logvance /opt/logvance

sudo cp deploy/logvance.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable logvance
sudo systemctl start logvance
```

### Nginx Reverse Proxy

```bash
sudo cp deploy/nginx-logvance.conf /etc/nginx/sites-available/logvance
sudo ln -s /etc/nginx/sites-available/logvance /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## API Endpoints

- `GET /health` - Health check
- `POST /api/auth/login` - Login
- `GET /api/v1/stats` - Statistics
- `GET /api/v1/top/ips` - Top IPs
- `GET /api/v1/top/paths` - Top paths
- `GET /api/v1/security/stats` - Security stats
- `GET /api/v1/export/pdf` - PDF export
- `GET /api/v1/ws` - WebSocket for real-time updates

## Security

- Bcrypt password hashing (cost=12)
- JWT tokens with 15min access, 7day refresh
- Rate limiting and account lockout
- Input validation and parameterized queries
- Security headers (CSP, X-Frame-Options, etc.)
- Geo-blocking and IP whitelisting

## Performance

- <10MB binary size
- 50K+ log lines/sec processing
- <5ms dashboard latency
- <100MB RAM usage (idle)
- 99.9% uptime target

## Architecture

- **Backend**: Go + Chi Router + DuckDB + WebSocket
- **Frontend**: React + Tailwind CSS + Recharts
- **Database**: DuckDB (embedded OLAP)
- **Real-time**: Gorilla WebSocket
- **Security**: Custom analyzer with regex patterns
- **Deployment**: Docker + Systemd + Nginx

Built for production VPS monitoring without SaaS dependencies.