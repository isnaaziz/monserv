# Deployment Guide - MonServ API

## Dynamic Host Configuration

MonServ API mendukung deployment di mana saja dengan automatic host detection.

## Environment Variables

### Required

```bash
# Server configuration
SERVER_PORT=18904                    # Port server (default: 8080)

# Monitoring targets
SERVERS=ssh://user:pass@host1:22,ssh://user:pass@host2:22
POLL_INTERVAL_SECONDS=4
MEM_THRESHOLD_PERCENT=80
DISK_THRESHOLD_PERCENT=80
PROC_RAM_THRESHOLD_PERCENT=90
```

### Optional - Swagger Host Override

```bash
# Set ini jika ingin force specific host di Swagger UI
# Jika tidak set, Swagger UI akan auto-detect dari browser
API_HOST=api.yourdomain.com:18904
```

## Deployment Options

### 1. Local Development

```bash
# Clone repository
git clone <repository-url>
cd monserv

# Copy environment file
cp .env.example .env

# Edit .env dengan konfigurasi Anda
nano .env

# Run server
go run cmd/server/main.go
```

**Access:**

- Web UI: `http://localhost:18904/`
- API: `http://localhost:18904/api/*`
- Swagger: `http://localhost:18904/swagger/index.html`
- WebSocket: `ws://localhost:18904/ws`

### 2. Production Server (VPS/Dedicated)

#### A. Build Binary

```bash
# Build for current OS
go build -o monserv cmd/server/main.go

# Build for Linux (if building on Mac/Windows)
GOOS=linux GOARCH=amd64 go build -o monserv cmd/server/main.go

# Build for ARM (Raspberry Pi, etc)
GOOS=linux GOARCH=arm64 go build -o monserv cmd/server/main.go
```

#### B. Deploy ke Server

```bash
# Upload binary
scp monserv user@your-server:/opt/monserv/
scp .env user@your-server:/opt/monserv/
scp -r web user@your-server:/opt/monserv/

# SSH ke server
ssh user@your-server

# Set permissions
cd /opt/monserv
chmod +x monserv

# Run with nohup
nohup ./monserv > monserv.log 2>&1 &
```

#### C. Systemd Service (Recommended)

Create `/etc/systemd/system/monserv.service`:

```ini
[Unit]
Description=MonServ Server Monitoring
After=network.target

[Service]
Type=simple
User=monserv
WorkingDirectory=/opt/monserv
ExecStart=/opt/monserv/monserv
Restart=always
RestartSec=10
StandardOutput=append:/var/log/monserv/monserv.log
StandardError=append:/var/log/monserv/error.log

# Environment variables
Environment="SERVER_PORT=18904"
Environment="GIN_MODE=release"

# Load from .env file
EnvironmentFile=/opt/monserv/.env

[Install]
WantedBy=multi-user.target
```

Setup dan start:

```bash
# Create user
sudo useradd -r -s /bin/false monserv

# Create directories
sudo mkdir -p /opt/monserv /var/log/monserv
sudo chown -R monserv:monserv /opt/monserv /var/log/monserv

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable monserv
sudo systemctl start monserv

# Check status
sudo systemctl status monserv

# View logs
sudo journalctl -u monserv -f
```

### 3. Docker Deployment

#### A. Build dan Run dengan Docker Compose (Recommended)

File `Dockerfile` dan `docker-compose.yml` sudah tersedia di repository.

**Dockerfile menggunakan multi-stage build:**

```dockerfile
# Build Stage
FROM golang:1.23-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o monserv ./cmd/server/main.go

# Final Stage
FROM golang:1.23-alpine AS final
WORKDIR /app

# Copy binary dan assets
COPY --from=builder /build/monserv .
COPY .env .env
COPY web/ ./web/

# Install godotenv
RUN go install github.com/joho/godotenv/cmd/godotenv@latest

EXPOSE 18904
CMD ["sh", "-c", "godotenv ./monserv"]
```

**Docker Compose setup:**

```yaml
version: "3.8"

services:
  monserv:
    build:
      context: .
      dockerfile: Dockerfile
      target: final
    container_name: monserv
    ports:
      - "18904:18904"
    env_file:
      - .env
    restart: always
    networks:
      - pdk_service
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - SERVER_PORT=18904
      - API_HOST=${API_HOST:-localhost:18904}

networks:
  pdk_service:
    external: true
```

**Setup dan jalankan:**

```bash
# 1. Pastikan network sudah ada
docker network ls | grep pdk_service || docker network create pdk_service

# 2. Edit .env file dengan konfigurasi Anda
nano .env

# 3. Build dan start container
docker-compose up -d

# 4. View logs
docker-compose logs -f monserv

# 5. Rebuild jika ada perubahan code
docker-compose up -d --build

# 6. Stop service
docker-compose down
```

#### B. Simple Docker Run (Tanpa Docker Compose)

```bash
# Build image
docker build -t monserv:latest .

# Run container
docker run -d \
  --name monserv \
  --network pdk_service \
  -p 18904:18904 \
  --env-file .env \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --restart always \
  monserv:latest

# View logs
docker logs -f monserv

# Stop container
docker stop monserv
docker rm monserv
```

# Stop

docker-compose down

````

### 4. Kubernetes Deployment

#### A. ConfigMap

Create `k8s-configmap.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: monserv-config
data:
  SERVER_PORT: "18904"
  POLL_INTERVAL_SECONDS: "4"
  MEM_THRESHOLD_PERCENT: "80"
  DISK_THRESHOLD_PERCENT: "80"
  PROC_RAM_THRESHOLD_PERCENT: "90"
````

#### B. Secret (untuk credentials)

```bash
kubectl create secret generic monserv-secrets \
  --from-literal=SERVERS='ssh://user:pass@host1:22,ssh://user:pass@host2:22' \
  --from-literal=EMAIL_PASSWORD='your-email-password'
```

#### C. Deployment

Create `k8s-deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: monserv
spec:
  replicas: 2
  selector:
    matchLabels:
      app: monserv
  template:
    metadata:
      labels:
        app: monserv
    spec:
      containers:
        - name: monserv
          image: your-registry/monserv:latest
          ports:
            - containerPort: 18904
          envFrom:
            - configMapRef:
                name: monserv-config
            - secretRef:
                name: monserv-secrets
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "256Mi"
              cpu: "200m"
```

#### D. Service

Create `k8s-service.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: monserv-service
spec:
  type: LoadBalancer
  selector:
    app: monserv
  ports:
    - protocol: TCP
      port: 80
      targetPort: 18904
```

Deploy:

```bash
kubectl apply -f k8s-configmap.yaml
kubectl apply -f k8s-deployment.yaml
kubectl apply -f k8s-service.yaml

# Check status
kubectl get pods
kubectl get svc monserv-service
```

## Reverse Proxy Configuration

### Nginx

```nginx
upstream monserv_backend {
    server localhost:18904;
}

server {
    listen 80;
    server_name api.yourdomain.com;

    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.yourdomain.com;

    # SSL certificates
    ssl_certificate /etc/letsencrypt/live/api.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.yourdomain.com/privkey.pem;

    # REST API
    location /api/ {
        proxy_pass http://monserv_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Swagger UI
    location /swagger/ {
        proxy_pass http://monserv_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # WebSocket
    location /ws {
        proxy_pass http://monserv_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 86400;
    }

    # Web UI
    location / {
        proxy_pass http://monserv_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Traefik (Docker)

```yaml
version: "3.8"

services:
  traefik:
    image: traefik:v2.10
    command:
      - "--api.insecure=true"
      - "--providers.docker=true"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.myresolver.acme.tlschallenge=true"
      - "--certificatesresolvers.myresolver.acme.email=your@email.com"
      - "--certificatesresolvers.myresolver.acme.storage=/letsencrypt/acme.json"
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./letsencrypt:/letsencrypt

  monserv:
    build: .
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.monserv.rule=Host(`api.yourdomain.com`)"
      - "traefik.http.routers.monserv.entrypoints=websecure"
      - "traefik.http.routers.monserv.tls.certresolver=myresolver"
      - "traefik.http.services.monserv.loadbalancer.server.port=18904"
    env_file:
      - .env
```

## SSL/TLS Configuration

### Using Let's Encrypt

```bash
# Install certbot
sudo apt-get install certbot

# Get certificate
sudo certbot certonly --standalone -d api.yourdomain.com

# Auto-renewal
sudo certbot renew --dry-run
```

## Monitoring & Logging

### Application Logs

```bash
# Systemd
sudo journalctl -u monserv -f

# Docker
docker logs -f monserv

# File
tail -f /var/log/monserv/monserv.log
```

### Health Check

```bash
# REST API health
curl https://api.yourdomain.com/api/v1/health

# WebSocket test
wscat -c wss://api.yourdomain.com/ws
```

## Security Best Practices

1. **Firewall Configuration**

   ```bash
   # Allow only necessary ports
   sudo ufw allow 22/tcp    # SSH
   sudo ufw allow 80/tcp    # HTTP
   sudo ufw allow 443/tcp   # HTTPS
   sudo ufw enable
   ```

2. **Environment Variables**

   - Never commit `.env` to git
   - Use secrets management (Vault, AWS Secrets Manager)
   - Rotate SSH passwords regularly

3. **API Security**

   - Add rate limiting
   - Implement authentication (JWT)
   - Use HTTPS only in production
   - Set CORS headers properly

4. **WebSocket Security**
   - Use WSS (WebSocket Secure)
   - Validate origin
   - Add authentication tokens

## Performance Tuning

### Go Configuration

```bash
# Set number of CPU cores
export GOMAXPROCS=4

# Enable GC optimization
export GOGC=100
```

### Gin Mode

```bash
# Production mode
export GIN_MODE=release
```

## Backup & Recovery

### Backup Script

```bash
#!/bin/bash
# backup-monserv.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups/monserv"

# Create backup directory
mkdir -p $BACKUP_DIR

# Backup configuration
cp /opt/monserv/.env $BACKUP_DIR/env_$DATE

# Backup logs
tar -czf $BACKUP_DIR/logs_$DATE.tar.gz /var/log/monserv/

# Keep only last 7 days
find $BACKUP_DIR -mtime +7 -delete
```

## Troubleshooting

### Common Issues

1. **Port Already in Use**

   ```bash
   # Find process
   sudo lsof -i :18904

   # Kill process
   sudo kill -9 <PID>
   ```

2. **Permission Denied**

   ```bash
   # Fix ownership
   sudo chown -R monserv:monserv /opt/monserv
   ```

3. **WebSocket Connection Failed**
   - Check firewall rules
   - Verify nginx proxy settings
   - Check SSL certificate

## Swagger UI Access After Deployment

Setelah deploy, Swagger UI akan otomatis detect host:

- **Development**: `http://localhost:18904/swagger/index.html`
- **Production**: `https://api.yourdomain.com/swagger/index.html`

Swagger akan otomatis gunakan host dari browser, tidak perlu konfigurasi tambahan!

## Testing Deployment

```bash
# Health check
curl https://api.yourdomain.com/api/v1/health

# Get servers
curl https://api.yourdomain.com/api/v1/servers

# WebSocket
wscat -c wss://api.yourdomain.com/ws

# Load test
ab -n 1000 -c 10 https://api.yourdomain.com/api/v1/health
```

## Support

Untuk issues deployment:

1. Check logs terlebih dahulu
2. Verify firewall configuration
3. Test dengan curl/wscat
4. Check systemd service status
