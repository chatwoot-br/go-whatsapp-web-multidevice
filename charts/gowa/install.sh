#!/bin/bash

# Install script for Go WhatsApp Admin API Helm Chart
# This script demonstrates different installation scenarios

set -e

NAMESPACE="whatsapp-admin"
RELEASE_NAME="whatsapp-admin"
CHART_PATH="./charts"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Helm is installed
if ! command -v helm &> /dev/null; then
    print_error "Helm is not installed. Please install Helm first."
    exit 1
fi

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed. Please install kubectl first."
    exit 1
fi

# Function to create namespace if it doesn't exist
create_namespace() {
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        print_info "Creating namespace: $NAMESPACE"
        kubectl create namespace "$NAMESPACE"
    else
        print_info "Namespace $NAMESPACE already exists"
    fi
}

# Function to create admin token secret
create_admin_secret() {
    local token="$1"
    local secret_name="whatsapp-admin-secret"
    
    if kubectl get secret "$secret_name" -n "$NAMESPACE" &> /dev/null; then
        print_warning "Secret $secret_name already exists in namespace $NAMESPACE"
        read -p "Do you want to update it? (y/n): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            kubectl delete secret "$secret_name" -n "$NAMESPACE"
        else
            return 0
        fi
    fi
    
    print_info "Creating admin token secret: $secret_name"
    kubectl create secret generic "$secret_name" \
        --from-literal=token="$token" \
        -n "$NAMESPACE"
}

# Function for development installation
install_dev() {
    print_info "Installing for development environment..."
    
    create_namespace
    
    helm upgrade --install "$RELEASE_NAME" "$CHART_PATH" \
        --namespace "$NAMESPACE" \
        --set admin.token="dev-admin-token-$(date +%s)" \
        --set persistence.size="5Gi" \
        --set resources.requests.cpu="100m" \
        --set resources.requests.memory="256Mi" \
        --set resources.limits.cpu="500m" \
        --set resources.limits.memory="512Mi"
}

# Function for production installation
install_prod() {
    local admin_token="$1"
    local domain="$2"
    
    if [[ -z "$admin_token" ]]; then
        print_error "Admin token is required for production installation"
        exit 1
    fi
    
    if [[ -z "$domain" ]]; then
        print_error "Domain is required for production installation"
        exit 1
    fi
    
    print_info "Installing for production environment..."
    
    create_namespace
    create_admin_secret "$admin_token"
    
    helm upgrade --install "$RELEASE_NAME" "$CHART_PATH" \
        --namespace "$NAMESPACE" \
        --values "$CHART_PATH/values-production.yaml" \
        --set admin.existingSecret="whatsapp-admin-secret" \
        --set ingress.hosts[0].host="$domain" \
        --set ingress.tls[0].hosts[0]="$domain" \
        --set ingress.tls[0].secretName="whatsapp-admin-tls"
}

# Function to uninstall
uninstall() {
    print_info "Uninstalling $RELEASE_NAME..."
    helm uninstall "$RELEASE_NAME" -n "$NAMESPACE" || true
    
    read -p "Do you want to delete the namespace and all data? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        kubectl delete namespace "$NAMESPACE" || true
        print_info "Namespace $NAMESPACE deleted"
    fi
}

# Function to show status
status() {
    print_info "Checking status of $RELEASE_NAME..."
    helm status "$RELEASE_NAME" -n "$NAMESPACE"
    
    print_info "Pod status:"
    kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=go-whatsapp-admin-api
    
    print_info "Service status:"
    kubectl get svc -n "$NAMESPACE" -l app.kubernetes.io/name=go-whatsapp-admin-api
}

# Function to get admin URL
get_url() {
    local service_name="$RELEASE_NAME-go-whatsapp-admin-api"
    
    # Check if ingress is enabled
    if kubectl get ingress -n "$NAMESPACE" "$service_name" &> /dev/null; then
        local host=$(kubectl get ingress -n "$NAMESPACE" "$service_name" -o jsonpath='{.spec.rules[0].host}')
        local tls=$(kubectl get ingress -n "$NAMESPACE" "$service_name" -o jsonpath='{.spec.tls}')
        
        if [[ -n "$tls" ]]; then
            echo "Admin API: https://$host"
        else
            echo "Admin API: http://$host"
        fi
    else
        print_info "No ingress found. You can access the services using port-forward:"
        echo ""
        echo "# Admin API (port 8088)"
        echo "kubectl port-forward -n $NAMESPACE svc/$service_name 8088:8088"
        echo "Then access: http://localhost:8088"
        echo ""
        echo "# Supervisord RPC (port 9001)"
        echo "kubectl port-forward -n $NAMESPACE svc/$service_name 9001:9001"
        echo "Then access: http://localhost:9001/RPC2"
        echo ""
        echo "# WhatsApp instances (ports 3001-3010)"
        echo "kubectl port-forward -n $NAMESPACE svc/$service_name 3001:3001"
        echo "Then access: http://localhost:3001"
        echo ""
        echo "Available ports: 8088 (admin), 9001 (supervisor), 3001-3010 (instances)"
    fi
}

# Function to show logs
logs() {
    kubectl logs -n "$NAMESPACE" -l app.kubernetes.io/name=go-whatsapp-admin-api --tail=100 -f
}

# Main script logic
case "${1:-}" in
    "dev")
        install_dev
        ;;
    "prod")
        if [[ $# -lt 3 ]]; then
            print_error "Usage: $0 prod <admin-token> <domain>"
            print_error "Example: $0 prod 'super-secure-token' 'whatsapp-admin.example.com'"
            exit 1
        fi
        install_prod "$2" "$3"
        ;;
    "uninstall")
        uninstall
        ;;
    "status")
        status
        ;;
    "url")
        get_url
        ;;
    "logs")
        logs
        ;;
    *)
        echo "Usage: $0 {dev|prod|uninstall|status|url|logs}"
        echo ""
        echo "Commands:"
        echo "  dev                           - Install for development (with auto-generated token)"
        echo "  prod <admin-token> <domain>   - Install for production with custom token and domain"
        echo "  uninstall                     - Uninstall the chart and optionally delete namespace"
        echo "  status                        - Show deployment status"
        echo "  url                           - Get the admin API URL"
        echo "  logs                          - Show application logs"
        echo ""
        echo "Examples:"
        echo "  $0 dev"
        echo "  $0 prod 'my-super-secure-token' 'whatsapp-admin.example.com'"
        echo "  $0 status"
        exit 1
        ;;
esac
