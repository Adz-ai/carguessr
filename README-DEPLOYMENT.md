# 🏠 Homelab Deployment Guide

This guide covers deploying Motors Price Guesser to your homelab server.

## 🐳 Docker Deployment (Recommended)

### Prerequisites
- Docker & Docker Compose installed
- 4GB+ RAM available
- Ports 80, 443, 8080 available

### Quick Start
```bash
# Clone and navigate to project
git clone <your-repo>
cd autotraderguesser

# Make deploy script executable
chmod +x deploy.sh

# Deploy (this handles everything)
./deploy.sh
```

### What Gets Deployed
- **Main App**: Motors Price Guesser on port 8080
- **Nginx**: Reverse proxy with SSL on ports 80/443
- **Chrome**: Headless browser for scraping (in container)

### Architecture
```
Internet → Nginx (80/443) → Motors App (8080) → Chrome Browser
```

## 🔧 Manual Docker Setup

### 1. Build and Run
```bash
# Build the image
docker-compose build

# Start services
docker-compose up -d

# Check status
docker-compose ps
docker-compose logs -f
```

### 2. SSL Configuration
```bash
# Generate self-signed cert (for testing)
mkdir ssl
openssl req -x509 -newkey rsa:4096 -keyout ssl/key.pem -out ssl/cert.pem -days 365 -nodes

# Or use your own certificates
cp your-cert.pem ssl/cert.pem
cp your-key.pem ssl/key.pem
```

### 3. Configuration
Edit `docker-compose.yml` for your needs:
- Change ports if needed
- Update domain in `nginx.conf`
- Adjust resource limits

## 🖥️ Direct Binary Deployment

### Prerequisites
- Go 1.24+
- Chrome/Chromium installed
- Systemd (for service management)

### Build and Install
```bash
# Build binary
go build -o motors-guesser cmd/server/main.go

# Copy to system location
sudo cp motors-guesser /usr/local/bin/
sudo chmod +x /usr/local/bin/motors-guesser

# Copy static files
sudo mkdir -p /opt/motors-guesser
sudo cp -r static /opt/motors-guesser/
```

### Create Systemd Service
```bash
# Copy service file
sudo cp motors-guesser.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable motors-guesser
sudo systemctl start motors-guesser
```

### Environment Variables
```bash
# Create environment file
sudo tee /etc/motors-guesser.env << EOF
GIN_MODE=release
PORT=8080
CHROME_PATH=/usr/bin/google-chrome
EOF
```

## 🌐 Reverse Proxy Options

### Nginx (Included)
The docker-compose includes Nginx with:
- SSL termination
- Rate limiting
- Security headers
- Static file caching

### Traefik Alternative
```yaml
# Add to existing Traefik setup
services:
  motors-guesser:
    build: .
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.motors.rule=Host(`motors.yourdomain.com`)"
      - "traefik.http.routers.motors.tls.certresolver=letsencrypt"
      - "traefik.http.services.motors.loadbalancer.server.port=8080"
```

### Caddy Alternative
```
motors.yourdomain.com {
    reverse_proxy motors-guesser:8080
    encode gzip
    header {
        X-Frame-Options "SAMEORIGIN"
        X-XSS-Protection "1; mode=block"
    }
}
```

## 📊 Monitoring & Maintenance

### Health Checks
```bash
# Check application health
curl http://localhost:8080/api/health

# Check container status
docker-compose ps
docker stats motors-price-guesser
```

### Logs
```bash
# View application logs
docker-compose logs -f motors-guesser

# View nginx logs
docker-compose logs -f nginx

# Follow all logs
docker-compose logs -f
```

### Updates
```bash
# Pull latest code
git pull

# Rebuild and restart
docker-compose build --no-cache
docker-compose up -d

# Check everything is working
curl http://localhost:8080/api/health
```

## 🔒 Security Considerations

### Firewall Rules
```bash
# Allow HTTP/HTTPS only
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Block direct access to app port (optional)
sudo ufw deny 8080/tcp
```

### Docker Security
- Containers run as non-root user
- Chrome runs with limited privileges
- Security headers enabled
- Rate limiting configured

### SSL/TLS
- Use proper certificates in production
- Enable HSTS headers
- Consider certificate pinning

## 🚨 Troubleshooting

### Common Issues

#### Chrome Issues in Docker
```bash
# Check Chrome process
docker exec motors-price-guesser ps aux | grep chrome

# Increase shared memory
# Add to docker-compose.yml:
shm_size: 2gb
```

#### Memory Issues
```bash
# Monitor memory usage
docker stats motors-price-guesser

# Increase memory limit in docker-compose.yml:
mem_limit: 4g
```

#### Scraping Failures
```bash
# Check scraper logs
docker-compose logs motors-guesser | grep -i "scraper"

# Test manually
curl http://localhost:8080/api/test-scraper
```

#### Permission Issues
```bash
# Fix file permissions
sudo chown -R 1000:1000 logs/
sudo chmod -R 755 static/
```

### Performance Tuning

#### For High Traffic
```yaml
# docker-compose.yml
services:
  motors-guesser:
    deploy:
      replicas: 3  # Run multiple instances
    environment:
      - GOMAXPROCS=4
```

#### For Low Memory
```yaml
# Reduce Chrome memory usage
environment:
  - CHROME_FLAGS=--memory-pressure-off,--max_old_space_size=512
```

## 📱 Production Checklist

- [ ] SSL certificates configured
- [ ] Domain DNS pointing to server
- [ ] Firewall rules applied
- [ ] Backup strategy in place
- [ ] Monitoring configured
- [ ] Log rotation enabled
- [ ] Auto-updates configured
- [ ] Health checks working

## 🆘 Support

If you encounter issues:

1. Check logs: `docker-compose logs -f`
2. Verify health: `curl localhost:8080/api/health`
3. Test scraper: `curl localhost:8080/api/test-scraper`
4. Monitor resources: `docker stats`

Common solutions are in the troubleshooting section above.