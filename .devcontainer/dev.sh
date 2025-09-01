#!/bin/bash

# GOWA Development Helper Script
# This script provides common development tasks for the GOWA project

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC_DIR="$SCRIPT_DIR/../src"
BINARY_PATH="/usr/local/bin/whatsapp"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if supervisord is running
check_supervisor() {
    if sudo supervisorctl status >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Start supervisord if not running
start_supervisor() {
    if check_supervisor; then
        log_info "Supervisord is already running"
    else
        log_info "Starting supervisord..."
        sudo supervisord -c /etc/supervisor/supervisord.conf
        sleep 2
        if check_supervisor; then
            log_success "Supervisord started successfully"
        else
            log_error "Failed to start supervisord"
            exit 1
        fi
    fi
}

# Build the GOWA binary
build() {
    log_info "Building GOWA binary..."
    cd "$SRC_DIR"
    
    # Build binary to a temporary location first
    go build -o /tmp/whatsapp .
    
    # Move it to the final location with sudo
    sudo mv /tmp/whatsapp "$BINARY_PATH"
    sudo chmod +x "$BINARY_PATH"
    
    log_success "Binary built and installed to $BINARY_PATH"
}

# Run tests
test() {
    log_info "Running tests..."
    cd "$SRC_DIR"
    go test ./...
    log_success "Tests completed"
}

# Start the main GOWA REST API
start_rest() {
    log_info "Starting GOWA REST API on port 3000..."
    cd "$SRC_DIR"
    go run . rest
}

# Start the Admin API
start_admin() {
    start_supervisor
    log_info "Starting GOWA Admin API on port 8088..."
    cd "$SRC_DIR"
    
    # Load environment variables from .env file
    if [ -f .env ]; then
        export $(cat .env | grep -v '^#' | grep -v '^$' | xargs)
    fi
    
    go run . admin --port 8088
}

# Show status of all services
status() {
    echo ""
    log_info "=== Service Status ==="
    
    # Check supervisord
    if check_supervisor; then
        log_success "Supervisord: Running"
        echo ""
        sudo supervisorctl status
    else
        log_warning "Supervisord: Not running"
    fi
    
    echo ""
    log_info "=== Port Status ==="
    
    # Check if ports are in use
    check_port() {
        local port=$1
        local service=$2
        if lsof -i :$port >/dev/null 2>&1; then
            log_success "$service (port $port): Running"
        else
            log_warning "$service (port $port): Not running"
        fi
    }
    
    check_port 3000 "GOWA REST API"
    check_port 8088 "Admin API"
    check_port 9001 "Supervisor Web UI"
}

# Stop all services
stop() {
    log_info "Stopping all services..."
    
    # Stop any GOWA processes
    pkill -f "go run.*rest" || true
    pkill -f "go run.*admin" || true
    pkill -f "whatsapp" || true
    
    # Stop supervisord
    if check_supervisor; then
        sudo supervisorctl shutdown || true
        log_success "Supervisord stopped"
    fi
    
    log_success "All services stopped"
}

# Create a new GOWA instance via Admin API
create_instance() {
    local port=${1:-3001}
    log_info "Creating new GOWA instance on port $port..."
    
    curl -X POST "http://localhost:8088/admin/instances" \
        -H "Authorization: Bearer dev-token-123" \
        -H "Content-Type: application/json" \
        -d "{\"port\": $port}" \
        || log_error "Failed to create instance. Make sure Admin API is running."
}

# List all instances
list_instances() {
    log_info "Listing all GOWA instances..."
    
    curl -X GET "http://localhost:8088/admin/instances" \
        -H "Authorization: Bearer dev-token-123" \
        || log_error "Failed to list instances. Make sure Admin API is running."
}

# Delete an instance
delete_instance() {
    local port=${1:-3001}
    log_info "Deleting GOWA instance on port $port..."
    
    curl -X DELETE "http://localhost:8088/admin/instances/$port" \
        -H "Authorization: Bearer dev-token-123" \
        || log_error "Failed to delete instance. Make sure Admin API is running."
}

# Update an instance configuration
update_instance() {
    local port=${1:-3001}
    local config=${2:-'{"debug": true}'}
    log_info "Updating GOWA instance on port $port with config: $config"
    
    curl -X PATCH "http://localhost:8088/admin/instances/$port" \
        -H "Authorization: Bearer dev-token-123" \
        -H "Content-Type: application/json" \
        -d "$config" \
        || log_error "Failed to update instance. Make sure Admin API is running."
}

# Show help
help() {
    echo ""
    echo "🚀 GOWA Development Helper"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  build              Build the GOWA binary"
    echo "  test               Run tests"
    echo "  start-rest         Start GOWA REST API (port 3000)"
    echo "  start-admin        Start Admin API (port 8088)"
    echo "  status             Show status of all services"
    echo "  stop               Stop all services"
    echo "  create [port]      Create new instance (default port: 3001)"
    echo "  list               List all instances"
    echo "  delete [port]      Delete instance (default port: 3001)"
    echo "  update [port] [config]  Update instance config (default port: 3001)"
    echo "  help               Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 build"
    echo "  $0 start-admin"
    echo "  $0 create 3002"
    echo "  $0 update 3002 '{\"debug\": false, \"webhook\": \"https://new-webhook.com\"}'"
    echo "  $0 delete 3002"
    echo ""
    echo "🔗 URLs:"
    echo "  • GOWA REST API: http://localhost:3000"
    echo "  • Admin API: http://localhost:8088"
    echo "  • Supervisor Web UI: http://localhost:9001"
    echo ""
}

# Main command handling
case "${1:-help}" in
    build)
        build
        ;;
    test)
        test
        ;;
    start-rest)
        start_rest
        ;;
    start-admin)
        start_admin
        ;;
    status)
        status
        ;;
    stop)
        stop
        ;;
    create)
        create_instance "$2"
        ;;
    list)
        list_instances
        ;;
    delete)
        delete_instance "$2"
        ;;
    update)
        update_instance "$2" "$3"
        ;;
    help|--help|-h)
        help
        ;;
    *)
        log_error "Unknown command: $1"
        help
        exit 1
        ;;
esac
