# 🚀 Motors Price Guesser - Production Deployment Guide

This guide shows how to deploy Motors Price Guesser to your Ubuntu server in production mode.

## 🎯 Quick Start

### Option 1: Run in Production Mode (Simple)
```bash
# On your local machine
make prod
```

### Option 2: Deploy to Ubuntu Server (Recommended)
```bash
# On your local machine
make build-prod
scp bin/motors-guesser-linux-amd64 your-server:/tmp/
scp -r static your-server:/tmp/

# On your server
sudo cp /tmp/motors-guesser-linux-amd64 /usr/local/bin/motors-price-guesser
sudo chmod +x /usr/local/bin/motors-price-guesser
sudo mkdir -p /opt/motors-price-guesser
sudo cp -r /tmp/static /opt/motors-price-guesser/
GIN_MODE=release /usr/local/bin/motors-price-guesser
```

### Option 3: Install as System Service (Best for Production)
```bash
# Copy project to your server first
scp -r . your-server:/tmp/motors-price-guesser/

# On your server
cd /tmp/motors-price-guesser
make install-service
make service-start
```

## 📋 All Available Make Commands

### Development
- `make dev` - Run in development mode (with Swagger docs)
- `make prod` - Run in production mode (no Swagger, secure)
- `make build` - Build development binary
- `make run` - Run the application normally

### Production
- `make build-prod` - Build optimized Linux production binary
- `make run-prod` - Build and run production binary locally

### Server Deployment (Ubuntu/Linux)
- `make install-service` - Install as systemd service
- `make service-start` - Start the service
- `make service-stop` - Stop the service
- `make service-restart` - Restart the service  
- `make service-status` - Check service status
- `make service-logs` - View service logs

## 🔧 Production Features

### ✅ What's Different in Production Mode?
- **Swagger Documentation Disabled** - `/swagger/` endpoints return 404 for security
- **Optimized Binary** - Smaller, faster binary with debug symbols stripped
- **Release Mode Logging** - Less verbose, production-appropriate logs
- **Better Performance** - Gin framework runs in release mode

### ✅ Automatic Features
- **Non-blocking Refresh** - Background car data updates don't interrupt gameplay
- **£700 Listing Filter** - Invalid price listings automatically removed
- **Auto-restart** - Service automatically restarts if it crashes
- **System Integration** - Starts automatically on server boot

## 🌐 Server Requirements

### Minimum Requirements
- **OS**: Ubuntu 18.04+ (or any Linux with systemd)
- **RAM**: 512MB minimum, 1GB recommended
- **CPU**: 1 vCPU minimum
- **Disk**: 100MB for application + cache
- **Network**: Outbound HTTPS for Bonhams scraping

### Dependencies
- No external dependencies required (statically linked binary)
- Go runtime NOT needed on production server

## 🔐 Security Considerations

### ✅ Production Security Features
- Swagger API documentation disabled
- Static file serving with proper headers
- CORS properly configured
- No debug information exposed

### 🛡️ Recommended Additional Security
```bash
# Firewall (only allow port 8080)
sudo ufw allow 8080
sudo ufw enable

# Reverse proxy with SSL (optional)
# nginx/apache config to proxy to localhost:8080
```

## 📊 Monitoring

### Check Service Status
```bash
make service-status
```

### View Live Logs
```bash
make service-logs
```

### Check Application Health
```bash
curl http://localhost:8080/api/health
curl http://localhost:8080/api/data-source
```

## 🔄 Updates and Maintenance

### Deploy New Version
```bash
# On local machine
make build-prod
scp bin/motors-guesser-linux-amd64 your-server:/tmp/

# On server
sudo systemctl stop motors-price-guesser
sudo cp /tmp/motors-guesser-linux-amd64 /usr/local/bin/motors-price-guesser
sudo systemctl start motors-price-guesser
```

### Or use the restart helper
```bash
# After copying new binary
make service-restart
```

### Manual Data Refresh
```bash
curl -X POST http://localhost:8080/api/refresh-listings
```

## 🆘 Troubleshooting

### Service Won't Start
```bash
sudo journalctl -u motors-price-guesser -n 50
```

### Port Already in Use
```bash
sudo netstat -tlnp | grep :8080
sudo kill [PID]
```

### Permission Issues
```bash
sudo chown -R www-data:www-data /opt/motors-price-guesser
sudo chmod +x /usr/local/bin/motors-price-guesser
```

### Clear Cache
```bash
sudo systemctl stop motors-price-guesser
sudo rm -f /opt/motors-price-guesser/cache/*.json
sudo systemctl start motors-price-guesser
```

## 📝 Example Production Deployment

```bash
# Complete deployment from scratch
git clone [your-repo]
cd motors-price-guesser

# Build production binary
make build-prod

# Copy to server
scp bin/motors-guesser-linux-amd64 server:/tmp/
scp -r static server:/tmp/
scp Makefile server:/tmp/

# On server - install as service
cd /tmp
sudo cp motors-guesser-linux-amd64 /usr/local/bin/motors-price-guesser
sudo chmod +x /usr/local/bin/motors-price-guesser
sudo mkdir -p /opt/motors-price-guesser
sudo cp -r static /opt/motors-price-guesser/
sudo chown -R www-data:www-data /opt/motors-price-guesser

# Create systemd service
make install-service
make service-start

# Verify it's working
curl http://localhost:8080/api/health
```

Your Motors Price Guesser is now running in production mode! 🎉