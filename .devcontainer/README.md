# GOWA Development Environment

This dev container provides a complete development environment for the Go WhatsApp Web Multi-Device project with Admin API support.

## 🚀 Quick Start

1. **Open in Dev Container**: VS Code will automatically detect the dev container configuration and prompt you to reopen in container.

2. **Wait for Setup**: The container will automatically build the GOWA binary and start supervisord.

3. **Start Development**: Use the provided development scripts to manage services.

## 📋 Available Services

- **GOWA REST API**: Port 3000 - Main WhatsApp API
- **Admin API**: Port 8088 - Multi-instance management
- **Supervisor Web UI**: Port 9001 - Process management interface

## 🛠️ Development Commands

Use the development helper script for common tasks:

```bash
# Build the GOWA binary
./.devcontainer/dev.sh build

# Start the main REST API
./.devcontainer/dev.sh start-rest

# Start the Admin API (includes supervisord)
./.devcontainer/dev.sh start-admin

# Check service status
./.devcontainer/dev.sh status

# Stop all services
./.devcontainer/dev.sh stop

# Create a new GOWA instance
./.devcontainer/dev.sh create 3001

# List all instances
./.devcontainer/dev.sh list

# Delete an instance
./.devcontainer/dev.sh delete 3001
```

## 🔧 Manual Commands

### Build and Install Binary
```bash
cd src
go build -o /usr/local/bin/whatsapp .
```

### Start Services Manually
```bash
# Start supervisord
sudo supervisord -c /etc/supervisor/supervisord.conf

# Start GOWA REST API
cd src && go run . rest

# Start Admin API
cd src && go run . admin --port 8088
```

### Supervisor Management
```bash
# Check supervisor status
sudo supervisorctl status

# Restart all programs
sudo supervisorctl restart all

# Stop supervisor
sudo supervisorctl shutdown
```

## 🌐 Access URLs

- **GOWA REST API**: http://localhost:3000
- **Admin API**: http://localhost:8088
- **Supervisor Web UI**: http://localhost:9001 (admin/admin123)

## 🔐 Default Credentials

- **Admin API Token**: `dev-token-123`
- **Supervisor**: `admin/admin123`
- **Instance Basic Auth**: `admin:admin`

## 📚 API Examples

### Admin API Usage

Create a new instance:
```bash
curl -X POST "http://localhost:8088/admin/instances" \
  -H "Authorization: Bearer dev-token-123" \
  -H "Content-Type: application/json" \
  -d '{"port": 3001}'
```

List instances:
```bash
curl -X GET "http://localhost:8088/admin/instances" \
  -H "Authorization: Bearer dev-token-123"
```

Delete instance:
```bash
curl -X DELETE "http://localhost:8088/admin/instances/3001" \
  -H "Authorization: Bearer dev-token-123"
```

### Health Checks

```bash
# Check Admin API health
curl http://localhost:8088/healthz

# Check Admin API readiness
curl http://localhost:8088/readyz
```

## 🔍 Debugging

### View Logs
```bash
# Supervisor logs
tail -f /var/log/supervisor/supervisord.log

# Instance logs
tail -f /var/log/supervisor/gowa-3001.log
```

### Check Processes
```bash
# List all GOWA processes
ps aux | grep whatsapp

# Check port usage
lsof -i :3000
lsof -i :8088
```

## 📁 Directory Structure

```
/app/instances/          # Instance data storage
/etc/supervisor/conf.d/  # Supervisor program configs
/var/log/supervisor/     # Supervisor and instance logs
/usr/local/bin/whatsapp  # GOWA binary
```

## 🔄 Environment Variables

The dev container includes pre-configured environment variables in `src/.env`:

```env
ADMIN_TOKEN=dev-token-123
SUPERVISOR_URL=http://127.0.0.1:9001/RPC2
SUPERVISOR_USER=admin
SUPERVISOR_PASS=admin123
GOWA_BIN=/usr/local/bin/whatsapp
# ... and more
```

## 🐛 Troubleshooting

### Supervisord not starting
```bash
# Check supervisor status
sudo supervisorctl status

# Restart supervisor
sudo supervisord -c /etc/supervisor/supervisord.conf
```

### Port conflicts
```bash
# Check what's using a port
lsof -i :3000

# Kill process on port
sudo fuser -k 3000/tcp
```

### Permission issues
```bash
# Fix ownership
sudo chown -R vscode:vscode /app/instances
sudo chown -R vscode:vscode /var/log/supervisor
```

## 📖 Documentation

- [Admin API Documentation](../docs/admin-api.md)
- [Main Project README](../readme.md)
- [API Specification](../docs/openapi.yaml)

## 🎯 Development Workflow

1. Make code changes in `src/`
2. Run tests: `go test ./...`
3. Build binary: `./.devcontainer/dev.sh build`
4. Test Admin API: `./.devcontainer/dev.sh start-admin`
5. Create test instances: `./.devcontainer/dev.sh create 3001`
6. Test functionality and clean up: `./.devcontainer/dev.sh delete 3001`
