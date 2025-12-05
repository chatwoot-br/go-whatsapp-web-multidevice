#!/bin/bash

# GOWA Development Environment Quick Start
# This script helps you get started with the development environment

echo "🚀 GOWA Development Environment"
echo "================================"
echo ""

# Check if we're in a dev container
if [ -f /.dockerenv ]; then
    echo "✅ Running in dev container"
else
    echo "❌ Not running in dev container"
    echo "Please open this project in VS Code and use 'Reopen in Container'"
    exit 1
fi

# Check if supervisord is running
if sudo supervisorctl status >/dev/null 2>&1; then
    echo "✅ Supervisord is running"
else
    echo "⚠️  Starting supervisord..."
    sudo supervisord -c /etc/supervisor/supervisord.conf
    sleep 2
fi

# Check if GOWA binary exists
if [ -f /usr/local/bin/whatsapp ]; then
    echo "✅ GOWA binary is installed"
else
    echo "⚠️  Building GOWA binary..."
    cd /workspaces/go-whatsapp-web-multidevice/src
    go build -o /tmp/whatsapp .
    sudo mv /tmp/whatsapp /usr/local/bin/whatsapp
    sudo chmod +x /usr/local/bin/whatsapp
    echo "✅ GOWA binary built and installed"
fi

echo ""
echo "🎯 Quick Commands:"
echo "  • ./.devcontainer/dev.sh build         - Build the binary"
echo "  • ./.devcontainer/dev.sh start-admin   - Start Admin API (port 8088)"
echo "  • ./.devcontainer/dev.sh start-rest    - Start REST API (port 3000)"
echo "  • ./.devcontainer/dev.sh create 3001   - Create instance on port 3001"
echo "  • ./.devcontainer/dev.sh list          - List all instances"
echo "  • ./.devcontainer/dev.sh status        - Show service status"
echo ""
echo "🌐 URLs:"
echo "  • Admin API: http://localhost:8088"
echo "  • REST API: http://localhost:3000"
echo "  • Supervisor: http://localhost:9001 (admin/admin123)"
echo ""
echo "🔐 Default Admin Token: dev-token-123"
echo ""
echo "📚 For more help: ./.devcontainer/dev.sh help"
echo ""

# Test API connectivity
echo "🔍 Testing API connectivity..."
if curl -s http://localhost:8088/healthz >/dev/null 2>&1; then
    echo "✅ Admin API is responding"
else
    echo "⚠️  Admin API is not running (use: ./.devcontainer/dev.sh start-admin)"
fi

if curl -s http://localhost:3000/ >/dev/null 2>&1; then
    echo "✅ REST API is responding"
else
    echo "⚠️  REST API is not running (use: ./.devcontainer/dev.sh start-rest)"
fi

echo ""
echo "✨ Development environment is ready!"
