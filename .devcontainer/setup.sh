#!/bin/bash

# Setup script for GOWA development environment
set -e

echo "🚀 Setting up GOWA development environment..."

# Change to the src directory
cd /workspaces/go-whatsapp-web-multidevice/src

# Build the whatsapp binary
echo "📦 Building whatsapp binary..."
go build -o /tmp/whatsapp .

# Move it to the final location with sudo
sudo mv /tmp/whatsapp /usr/local/bin/whatsapp
sudo chmod +x /usr/local/bin/whatsapp

echo "✅ whatsapp binary built and installed to /usr/local/bin/whatsapp"

# Start supervisord in the background
echo "🔧 Starting supervisord..."
sudo supervisord -c /etc/supervisor/supervisord.conf

# Wait a moment for supervisord to start
sleep 2

# Check if supervisord is running
if sudo supervisorctl status >/dev/null 2>&1; then
    echo "✅ Supervisord is running and accessible"
else
    echo "❌ Failed to start supervisord"
    exit 1
fi

echo ""
echo "🎉 Setup complete! Your development environment is ready."
echo ""
echo "📋 Quick reference:"
echo "  • Build binary: cd src && go build -o /usr/local/bin/whatsapp ."
echo "  • Start admin server: cd src && go run . admin --port 8088"
echo "  • Check supervisord: sudo supervisorctl status"
echo "  • Admin API will be available at: http://localhost:8088"
echo ""
echo "🔐 Default credentials:"
echo "  • Supervisor: admin/admin123"
echo "  • Admin token: dev-token-123"
echo ""
echo "📖 For more information, see docs/admin-api.md"
