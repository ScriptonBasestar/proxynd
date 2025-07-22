#!/bin/bash

# ProxyND Deployment Script
# This script handles deployment to various environments

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
HELM_CHART_DIR="$PROJECT_ROOT/helm"

# Default values
ENVIRONMENT=""
VERSION=""
NAMESPACE=""
DRY_RUN=false
ROLLBACK=false
HEALTH_CHECK=true
TIMEOUT="600s"

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

# Help function
show_help() {
    cat << EOF
ProxyND Deployment Script

Usage: $0 [OPTIONS]

Options:
    -e, --environment ENVIRONMENT    Target environment (staging, production)
    -v, --version VERSION           Version to deploy (e.g., v1.0.0)
    -n, --namespace NAMESPACE       Kubernetes namespace (optional)
    -d, --dry-run                   Perform a dry run without actual deployment
    -r, --rollback                  Rollback to previous version
    -t, --timeout TIMEOUT          Deployment timeout (default: 600s)
    --no-health-check              Skip health checks after deployment
    -h, --help                     Show this help message

Examples:
    $0 -e staging -v v1.0.0
    $0 -e production -v v1.0.0 --dry-run
    $0 -e staging --rollback
    $0 -e production -v v1.2.3 -n custom-namespace

Environment Configurations:
    staging     - Deploys to staging environment with reduced resources
    production  - Deploys to production environment with full resources and autoscaling

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -e|--environment)
                ENVIRONMENT="$2"
                shift 2
                ;;
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -r|--rollback)
                ROLLBACK=true
                shift
                ;;
            -t|--timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            --no-health-check)
                HEALTH_CHECK=false
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Validate arguments
validate_args() {
    if [[ -z "$ENVIRONMENT" ]]; then
        log_error "Environment is required. Use -e or --environment"
        exit 1
    fi

    if [[ "$ENVIRONMENT" != "staging" && "$ENVIRONMENT" != "production" ]]; then
        log_error "Environment must be 'staging' or 'production'"
        exit 1
    fi

    if [[ "$ROLLBACK" == false && -z "$VERSION" ]]; then
        log_error "Version is required for deployment. Use -v or --version"
        exit 1
    fi

    # Set default namespace if not provided
    if [[ -z "$NAMESPACE" ]]; then
        if [[ "$ENVIRONMENT" == "staging" ]]; then
            NAMESPACE="proxynd-staging"
        else
            NAMESPACE="proxynd"
        fi
    fi
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi

    # Check if helm is installed
    if ! command -v helm &> /dev/null; then
        log_error "helm is not installed or not in PATH"
        exit 1
    fi

    # Check if kubectl can connect to cluster
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster. Check your kubeconfig"
        exit 1
    fi

    # Check if helm chart exists
    if [[ ! -d "$HELM_CHART_DIR" ]]; then
        log_error "Helm chart directory not found: $HELM_CHART_DIR"
        exit 1
    fi

    log_success "Prerequisites check passed"
}

# Get environment configuration
get_environment_config() {
    case "$ENVIRONMENT" in
        staging)
            cat << EOF
image:
  tag: "$VERSION"
environment: staging
replicaCount: 2
resources:
  requests:
    memory: "512Mi"
    cpu: "250m"
  limits:
    memory: "1Gi"
    cpu: "500m"
ingress:
  enabled: true
  hosts:
    - host: staging.proxynd.example.com
      paths:
        - path: /
          pathType: Prefix
autoscaling:
  enabled: false
monitoring:
  enabled: true
EOF
            ;;
        production)
            cat << EOF
image:
  tag: "$VERSION"
environment: production
replicaCount: 3
resources:
  requests:
    memory: "1Gi"
    cpu: "500m"
  limits:
    memory: "2Gi"
    cpu: "1000m"
ingress:
  enabled: true
  hosts:
    - host: proxynd.example.com
      paths:
        - path: /
          pathType: Prefix
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80
monitoring:
  enabled: true
performance:
  enabled: true
  cache_optimizer:
    enabled: true
  connection_pool:
    enabled: true
security:
  enabled: true
  networkPolicies:
    enabled: true
EOF
            ;;
    esac
}

# Perform rollback
perform_rollback() {
    log_info "Rolling back deployment in namespace: $NAMESPACE"

    local release_name="proxynd"
    if [[ "$ENVIRONMENT" == "staging" ]]; then
        release_name="proxynd-staging"
    fi

    if [[ "$DRY_RUN" == true ]]; then
        log_info "DRY RUN: Would rollback $release_name in namespace $NAMESPACE"
        return 0
    fi

    # Get revision history
    log_info "Getting rollback history..."
    helm history "$release_name" --namespace "$NAMESPACE"

    # Rollback to previous revision
    log_info "Rolling back to previous revision..."
    helm rollback "$release_name" --namespace "$NAMESPACE" --wait --timeout="$TIMEOUT"

    log_success "Rollback completed successfully"
}

# Deploy to environment
deploy() {
    log_info "Deploying ProxyND $VERSION to $ENVIRONMENT environment"
    log_info "Namespace: $NAMESPACE"
    log_info "Timeout: $TIMEOUT"

    local release_name="proxynd"
    if [[ "$ENVIRONMENT" == "staging" ]]; then
        release_name="proxynd-staging"
    fi

    # Create values file
    local values_file="/tmp/proxynd-${ENVIRONMENT}-values.yaml"
    get_environment_config > "$values_file"

    log_info "Generated values file: $values_file"

    # Helm deployment command
    local helm_cmd=(
        helm upgrade --install "$release_name" "$HELM_CHART_DIR"
        --namespace "$NAMESPACE"
        --create-namespace
        --values "$values_file"
        --wait
        --timeout="$TIMEOUT"
    )

    if [[ "$DRY_RUN" == true ]]; then
        helm_cmd+=(--dry-run)
        log_info "DRY RUN: Performing dry run deployment"
    fi

    # Execute helm command
    log_info "Executing Helm deployment..."
    "${helm_cmd[@]}"

    if [[ "$DRY_RUN" == false ]]; then
        log_success "Deployment completed successfully"
    else
        log_success "Dry run completed successfully"
    fi

    # Cleanup values file
    rm -f "$values_file"
}

# Run health checks
run_health_checks() {
    if [[ "$HEALTH_CHECK" == false || "$DRY_RUN" == true ]]; then
        return 0
    fi

    log_info "Running health checks..."

    local release_name="proxynd"
    if [[ "$ENVIRONMENT" == "staging" ]]; then
        release_name="proxynd-staging"
    fi

    # Wait for pods to be ready
    log_info "Waiting for pods to be ready..."
    kubectl wait --for=condition=ready pods \
        --selector="app.kubernetes.io/instance=$release_name" \
        --namespace="$NAMESPACE" \
        --timeout=300s

    # Check deployment status
    log_info "Checking deployment status..."
    kubectl get deployment --namespace="$NAMESPACE"

    # Check pod status
    log_info "Checking pod status..."
    kubectl get pods --namespace="$NAMESPACE" --selector="app.kubernetes.io/instance=$release_name"

    # Health check endpoint (if available)
    local health_url=""
    if [[ "$ENVIRONMENT" == "staging" ]]; then
        health_url="https://staging.proxynd.example.com/health"
    else
        health_url="https://proxynd.example.com/health"
    fi

    # Port forward for health check if ingress is not available
    log_info "Testing health endpoint via port-forward..."
    kubectl port-forward --namespace="$NAMESPACE" service/"$release_name" 8080:8080 &
    local port_forward_pid=$!

    sleep 5

    if curl -f http://localhost:8080/health &> /dev/null; then
        log_success "Health check passed"
    else
        log_warning "Health check endpoint not responding"
    fi

    # Cleanup port forward
    kill $port_forward_pid 2>/dev/null || true

    log_success "Health checks completed"
}

# Get deployment status
get_deployment_status() {
    log_info "Getting deployment status for $ENVIRONMENT environment"

    local release_name="proxynd"
    if [[ "$ENVIRONMENT" == "staging" ]]; then
        release_name="proxynd-staging"
    fi

    # Helm status
    log_info "Helm release status:"
    helm status "$release_name" --namespace "$NAMESPACE"

    # Deployment details
    log_info "Deployment details:"
    kubectl describe deployment --namespace="$NAMESPACE" --selector="app.kubernetes.io/instance=$release_name"

    # Pod status
    log_info "Pod status:"
    kubectl get pods --namespace="$NAMESPACE" --selector="app.kubernetes.io/instance=$release_name" -o wide
}

# Main execution
main() {
    log_info "ProxyND Deployment Script"
    log_info "========================="

    parse_args "$@"
    validate_args
    check_prerequisites

    if [[ "$ROLLBACK" == true ]]; then
        perform_rollback
    else
        deploy
        run_health_checks
    fi

    get_deployment_status

    log_success "Deployment script completed successfully!"
}

# Execute main function with all arguments
main "$@"
