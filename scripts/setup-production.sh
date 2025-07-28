#!/bin/bash

# Production Environment Setup Script
# This script sets up the production environment for the Cloud-Based Collaborative Development Platform

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ENV_FILE="$PROJECT_ROOT/.env.production"
COMPOSE_FILE="$PROJECT_ROOT/docker-compose.production.yml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -eq 0 ]]; then
        log_error "This script should not be run as root"
        exit 1
    fi
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed"
        exit 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi
    
    # Check available disk space (minimum 20GB)
    available_space=$(df / | awk 'NR==2 {print $4}')
    min_space=$((20 * 1024 * 1024)) # 20GB in KB
    
    if [[ $available_space -lt $min_space ]]; then
        log_error "Insufficient disk space. At least 20GB required."
        exit 1
    fi
    
    log_success "All prerequisites satisfied"
}

# Create necessary directories
create_directories() {
    log_info "Creating necessary directories..."
    
    directories=(
        "$PROJECT_ROOT/data/postgres"
        "$PROJECT_ROOT/data/redis"
        "$PROJECT_ROOT/data/storage"
        "$PROJECT_ROOT/logs"
        "$PROJECT_ROOT/backups"
        "$PROJECT_ROOT/ssl"
    )
    
    for dir in "${directories[@]}"; do
        mkdir -p "$dir"
        chmod 755 "$dir"
    done
    
    log_success "Directories created"
}

# Setup SSL certificates
setup_ssl() {
    log_info "Setting up SSL certificates..."
    
    SSL_DIR="$PROJECT_ROOT/ssl"
    
    if [[ ! -f "$SSL_DIR/server.crt" ]] || [[ ! -f "$SSL_DIR/server.key" ]]; then
        log_warning "SSL certificates not found. Generating self-signed certificates..."
        log_warning "For production, replace with proper SSL certificates from a CA"
        
        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout "$SSL_DIR/server.key" \
            -out "$SSL_DIR/server.crt" \
            -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"
        
        chmod 600 "$SSL_DIR/server.key"
        chmod 644 "$SSL_DIR/server.crt"
    fi
    
    log_success "SSL certificates configured"
}

# Generate secure environment file
setup_environment() {
    log_info "Setting up production environment file..."
    
    # Generate random secrets
    JWT_SECRET=$(openssl rand -base64 32)
    DATABASE_PASSWORD=$(openssl rand -base64 32)
    REDIS_PASSWORD=$(openssl rand -base64 32)
    ENCRYPTION_KEY=$(openssl rand -base64 32)
    
    cat > "$ENV_FILE" << EOF
# Production Environment Configuration
# WARNING: This file contains sensitive information. Keep it secure!

# Environment
NODE_ENV=production
GO_ENV=production

# Application
APP_NAME=Collaborative Development Platform
APP_VERSION=1.0.0

# Database Configuration
DB_HOST=postgres
DB_PORT=5432
DB_NAME=collaborative_platform
DB_USER=platform_user
DB_PASSWORD=$DATABASE_PASSWORD
DB_SSL_MODE=require

# Redis Configuration
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=$REDIS_PASSWORD
REDIS_DB=0

# Authentication
JWT_SECRET=$JWT_SECRET
ENCRYPTION_KEY=$ENCRYPTION_KEY
SESSION_SECRET=$(openssl rand -base64 32)

# Security
CORS_ORIGINS=https://yourdomain.com
TRUSTED_PROXIES=127.0.0.1,::1

# Storage (S3 Compatible)
STORAGE_TYPE=s3
S3_ENDPOINT=
S3_BUCKET=collaborative-platform-storage
S3_ACCESS_KEY=
S3_SECRET_KEY=
S3_REGION=us-east-1

# Monitoring
ENABLE_METRICS=true
METRICS_PORT=9090
JAEGER_ENDPOINT=http://jaeger:14268/api/traces

# Email Configuration (SMTP)
SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASSWORD=
SMTP_FROM=noreply@yourdomain.com

# Webhook Configuration
WEBHOOK_SECRET=$(openssl rand -base64 32)

# Rate Limiting
RATE_LIMIT_REQUESTS_PER_MINUTE=100
RATE_LIMIT_BURST=200

# File Upload Limits
MAX_FILE_SIZE=100MB
MAX_REQUEST_SIZE=10MB

# Backup Configuration
BACKUP_RETENTION_DAYS=30
BACKUP_S3_BUCKET=collaborative-platform-backups

# License
LICENSE_KEY=

EOF

    chmod 600 "$ENV_FILE"
    log_success "Environment file created at $ENV_FILE"
    log_warning "Please update the configuration values in $ENV_FILE before deployment"
}

# Create production Docker Compose file
setup_docker_compose() {
    log_info "Creating production Docker Compose configuration..."
    
    cat > "$COMPOSE_FILE" << 'EOF'
version: '3.8'

services:
  # Database
  postgres:
    image: postgres:15-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: ${DB_NAME}
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - ./data/postgres:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    networks:
      - backend
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER} -d ${DB_NAME}"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Redis Cache
  redis:
    image: redis:7-alpine
    restart: unless-stopped
    command: redis-server --requirepass ${REDIS_PASSWORD} --appendonly yes
    volumes:
      - ./data/redis:/data
    networks:
      - backend
    healthcheck:
      test: ["CMD", "redis-cli", "--no-auth-warning", "-a", "${REDIS_PASSWORD}", "ping"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Load Balancer
  nginx:
    image: nginx:alpine
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/ssl/certs:ro
      - ./logs/nginx:/var/log/nginx
    depends_on:
      - frontend
      - api-gateway
    networks:
      - frontend
      - backend
    healthcheck:
      test: ["CMD", "nginx", "-t"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Frontend Service
  frontend:
    build:
      context: .
      dockerfile: frontend/Dockerfile.production
    restart: unless-stopped
    environment:
      - NODE_ENV=production
    networks:
      - frontend
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  # API Gateway
  api-gateway:
    build:
      context: .
      dockerfile: cmd/api-gateway/Dockerfile
    restart: unless-stopped
    env_file: .env.production
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - frontend
      - backend
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Microservices
  iam-service:
    build:
      context: .
      dockerfile: cmd/iam-service/Dockerfile
    restart: unless-stopped
    env_file: .env.production
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - backend

  project-service:
    build:
      context: .
      dockerfile: cmd/project-service/Dockerfile
    restart: unless-stopped
    env_file: .env.production
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - backend

  file-service:
    build:
      context: .
      dockerfile: cmd/file-service/Dockerfile
    restart: unless-stopped
    env_file: .env.production
    volumes:
      - ./data/storage:/app/storage
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - backend

  git-gateway-service:
    build:
      context: .
      dockerfile: cmd/git-gateway-service/Dockerfile
    restart: unless-stopped
    env_file: .env.production
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - backend

  cicd-service:
    build:
      context: .
      dockerfile: cmd/cicd-service/Dockerfile
    restart: unless-stopped
    env_file: .env.production
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data/storage:/app/storage
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - backend

  notification-service:
    build:
      context: .
      dockerfile: cmd/notification-service/Dockerfile
    restart: unless-stopped
    env_file: .env.production
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - backend

  team-service:
    build:
      context: .
      dockerfile: cmd/team-service/Dockerfile
    restart: unless-stopped
    env_file: .env.production
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - backend

  tenant-service:
    build:
      context: .
      dockerfile: cmd/tenant-service/Dockerfile
    restart: unless-stopped
    env_file: .env.production
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - backend

  websocket-service:
    build:
      context: .
      dockerfile: cmd/websocket-service/Dockerfile
    restart: unless-stopped
    env_file: .env.production
    depends_on:
      redis:
        condition: service_healthy
    networks:
      - backend

  # Monitoring
  prometheus:
    image: prom/prometheus:latest
    restart: unless-stopped
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus:/etc/prometheus
      - ./data/prometheus:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--web.enable-lifecycle'
    networks:
      - monitoring

  grafana:
    image: grafana/grafana:latest
    restart: unless-stopped
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD:-admin}
    volumes:
      - ./data/grafana:/var/lib/grafana
      - ./monitoring/grafana:/etc/grafana/provisioning
    networks:
      - monitoring

  jaeger:
    image: jaegertracing/all-in-one:latest
    restart: unless-stopped
    ports:
      - "16686:16686"
      - "14268:14268"
    environment:
      - COLLECTOR_OTLP_ENABLED=true
    networks:
      - monitoring

networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
  monitoring:
    driver: bridge

volumes:
  postgres_data:
  redis_data:
  storage_data:
  prometheus_data:
  grafana_data:
EOF

    log_success "Docker Compose configuration created"
}

# Create Nginx configuration
setup_nginx() {
    log_info "Creating Nginx configuration..."
    
    mkdir -p "$PROJECT_ROOT/nginx"
    
    cat > "$PROJECT_ROOT/nginx/nginx.conf" << 'EOF'
events {
    worker_connections 1024;
}

http {
    upstream frontend {
        server frontend:3000;
    }
    
    upstream api {
        server api-gateway:8080;
    }

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline';" always;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=login:10m rate=5r/m;

    # Redirect HTTP to HTTPS
    server {
        listen 80;
        server_name _;
        return 301 https://$host$request_uri;
    }

    # HTTPS server
    server {
        listen 443 ssl http2;
        server_name _;

        ssl_certificate /etc/ssl/certs/server.crt;
        ssl_certificate_key /etc/ssl/certs/server.key;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
        ssl_prefer_server_ciphers off;

        # Frontend
        location / {
            proxy_pass http://frontend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        # API endpoints
        location /api/ {
            limit_req zone=api burst=20 nodelay;
            proxy_pass http://api;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        # Login endpoint with stricter rate limiting
        location /api/auth/login {
            limit_req zone=login burst=5 nodelay;
            proxy_pass http://api;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        # WebSocket
        location /ws/ {
            proxy_pass http://websocket-service:8090;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}
EOF

    log_success "Nginx configuration created"
}

# Create backup script
setup_backup() {
    log_info "Creating backup script..."
    
    cat > "$PROJECT_ROOT/scripts/backup.sh" << 'EOF'
#!/bin/bash

# Backup script for production environment

set -euo pipefail

BACKUP_DIR="/tmp/backup-$(date +%Y%m%d-%H%M%S)"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Backup database
echo "Backing up database..."
docker-compose -f "$PROJECT_ROOT/docker-compose.production.yml" exec -T postgres \
    pg_dump -U platform_user collaborative_platform > "$BACKUP_DIR/database.sql"

# Backup storage data
echo "Backing up storage..."
cp -r "$PROJECT_ROOT/data/storage" "$BACKUP_DIR/"

# Backup configuration
echo "Backing up configuration..."
cp "$PROJECT_ROOT/.env.production" "$BACKUP_DIR/"
cp -r "$PROJECT_ROOT/ssl" "$BACKUP_DIR/"

# Create archive
echo "Creating backup archive..."
cd /tmp
tar -czf "backup-$(date +%Y%m%d-%H%M%S).tar.gz" "$(basename $BACKUP_DIR)"

# Upload to S3 (if configured)
if [[ -n "${BACKUP_S3_BUCKET:-}" ]]; then
    echo "Uploading to S3..."
    aws s3 cp "backup-$(date +%Y%m%d-%H%M%S).tar.gz" "s3://$BACKUP_S3_BUCKET/"
fi

# Cleanup old backups (keep last 7 days)
find /tmp -name "backup-*.tar.gz" -mtime +7 -delete

echo "Backup completed successfully"
EOF

    chmod +x "$PROJECT_ROOT/scripts/backup.sh"
    log_success "Backup script created"
}

# Create systemd service for auto-start
setup_systemd() {
    log_info "Creating systemd service..."
    
    SERVICE_FILE="/etc/systemd/system/collaborative-platform.service"
    
    sudo tee "$SERVICE_FILE" > /dev/null << EOF
[Unit]
Description=Collaborative Development Platform
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=$PROJECT_ROOT
ExecStart=/usr/bin/docker-compose -f docker-compose.production.yml up -d
ExecStop=/usr/bin/docker-compose -f docker-compose.production.yml down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

    sudo systemctl daemon-reload
    sudo systemctl enable collaborative-platform
    
    log_success "Systemd service created and enabled"
}

# Main setup function
main() {
    log_info "Starting production environment setup..."
    
    check_root
    check_prerequisites
    create_directories
    setup_ssl
    setup_environment
    setup_docker_compose
    setup_nginx
    setup_backup
    
    # Only setup systemd if running with sudo access
    if sudo -n true 2>/dev/null; then
        setup_systemd
    else
        log_warning "Skipping systemd setup (requires sudo)"
    fi
    
    log_success "Production environment setup completed!"
    echo
    log_info "Next steps:"
    echo "1. Update configuration in $ENV_FILE"
    echo "2. Replace SSL certificates in $PROJECT_ROOT/ssl/"
    echo "3. Run: docker-compose -f docker-compose.production.yml up -d"
    echo "4. Setup monitoring alerts and backup schedules"
    echo
    log_warning "Remember to secure your environment file and SSL certificates!"
}

# Run main function
main "$@"
EOF

chmod +x "$PROJECT_ROOT/scripts/setup-production.sh"