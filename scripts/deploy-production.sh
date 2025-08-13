#!/bin/bash
# ProxyND Production Deployment Script
# 프로덕션 배포용 스크립트 - Blue-Green 배포 지원

set -euo pipefail

# 설정
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
NAMESPACE="${NAMESPACE:-production}"
RELEASE_NAME="${RELEASE_NAME:-proxynd}"
REGISTRY="${REGISTRY:-ghcr.io/your-org}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
TIMEOUT="${TIMEOUT:-600s}"
HEALTH_CHECK_RETRIES="${HEALTH_CHECK_RETRIES:-30}"
DEPLOYMENT_STRATEGY="${DEPLOYMENT_STRATEGY:-blue-green}"

# 색상 출력
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 로깅 함수
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $*"
}

error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

# 사전 조건 확인
check_prerequisites() {
    log "사전 조건 확인 중..."
    
    # 필수 도구 확인
    for tool in kubectl helm docker jq; do
        if ! command -v "$tool" &> /dev/null; then
            error "$tool이 설치되어 있지 않습니다"
            exit 1
        fi
    done
    
    # Kubernetes 연결 확인
    if ! kubectl cluster-info &> /dev/null; then
        error "Kubernetes 클러스터에 연결할 수 없습니다"
        exit 1
    fi
    
    # Helm 차트 확인
    if [[ ! -f "$PROJECT_ROOT/helm/Chart.yaml" ]]; then
        error "Helm 차트를 찾을 수 없습니다: $PROJECT_ROOT/helm/Chart.yaml"
        exit 1
    fi
    
    success "사전 조건 확인 완료"
}

# 현재 배포 상태 확인
check_current_deployment() {
    log "현재 배포 상태 확인 중..."
    
    if kubectl get deployment "$RELEASE_NAME" -n "$NAMESPACE" &> /dev/null; then
        CURRENT_IMAGE=$(kubectl get deployment "$RELEASE_NAME" -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].image}')
        log "현재 이미지: $CURRENT_IMAGE"
        return 0
    else
        log "기존 배포가 존재하지 않습니다"
        return 1
    fi
}

# 백업 생성
create_backup() {
    log "현재 배포 백업 생성 중..."
    
    local backup_dir="$PROJECT_ROOT/backups/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$backup_dir"
    
    # 현재 배포 상태 백업
    kubectl get all -n "$NAMESPACE" -l app.kubernetes.io/name="$RELEASE_NAME" -o yaml > "$backup_dir/current-deployment.yaml"
    kubectl get configmap -n "$NAMESPACE" -l app.kubernetes.io/name="$RELEASE_NAME" -o yaml > "$backup_dir/configmaps.yaml"
    kubectl get secret -n "$NAMESPACE" -l app.kubernetes.io/name="$RELEASE_NAME" -o yaml > "$backup_dir/secrets.yaml"
    
    # Helm values 백업
    helm get values "$RELEASE_NAME" -n "$NAMESPACE" > "$backup_dir/helm-values.yaml"
    
    echo "$backup_dir" > "$PROJECT_ROOT/.last-backup"
    success "백업 완료: $backup_dir"
}

# 이미지 검증
validate_image() {
    log "이미지 검증 중: $REGISTRY/proxynd:$IMAGE_TAG"
    
    # 이미지 존재 확인
    if ! docker manifest inspect "$REGISTRY/proxynd:$IMAGE_TAG" &> /dev/null; then
        error "이미지를 찾을 수 없습니다: $REGISTRY/proxynd:$IMAGE_TAG"
        exit 1
    fi
    
    # 보안 스캔 결과 확인 (선택적)
    if [[ "${SKIP_SECURITY_SCAN:-false}" != "true" ]]; then
        log "보안 스캔 실행 중..."
        docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
            aquasec/trivy:latest image --exit-code 1 --severity HIGH,CRITICAL \
            "$REGISTRY/proxynd:$IMAGE_TAG" || {
            error "보안 스캔 실패. SKIP_SECURITY_SCAN=true로 건너뛸 수 있습니다"
            exit 1
        }
    fi
    
    success "이미지 검증 완료"
}

# Blue-Green 배포
deploy_blue_green() {
    log "Blue-Green 배포 시작..."
    
    local green_release="${RELEASE_NAME}-green"
    
    # Green 슬롯에 새 버전 배포
    log "Green 슬롯에 새 버전 배포 중..."
    helm upgrade --install "$green_release" "$PROJECT_ROOT/helm" \
        --namespace "$NAMESPACE" \
        --create-namespace \
        --set image.repository="$REGISTRY/proxynd" \
        --set image.tag="$IMAGE_TAG" \
        --set nameOverride="$green_release" \
        --set service.name="${RELEASE_NAME}-green-svc" \
        --set ingress.enabled=false \
        --wait --timeout="$TIMEOUT"
    
    # Green 슬롯 헬스체크
    if ! health_check "$green_release"; then
        error "Green 슬롯 헬스체크 실패"
        rollback_deployment
        exit 1
    fi
    
    # 트래픽 전환
    log "트래픽을 Green 슬롯으로 전환 중..."
    kubectl patch service "${RELEASE_NAME}-svc" -n "$NAMESPACE" \
        -p '{"spec":{"selector":{"app.kubernetes.io/name":"'"$green_release"'"}}}'
    
    # 전환 후 헬스체크
    sleep 30
    if ! health_check_external; then
        error "트래픽 전환 후 헬스체크 실패"
        # 트래픽을 다시 Blue로 전환
        kubectl patch service "${RELEASE_NAME}-svc" -n "$NAMESPACE" \
            -p '{"spec":{"selector":{"app.kubernetes.io/name":"'"$RELEASE_NAME"'"}}}'
        rollback_deployment
        exit 1
    fi
    
    # Blue 슬롯 정리
    log "Blue 슬롯 정리 중..."
    helm uninstall "$RELEASE_NAME" -n "$NAMESPACE" || true
    
    # Green을 새로운 Blue로 이름 변경
    log "Green 슬롯을 기본 배포로 승격 중..."
    helm upgrade --install "$RELEASE_NAME" "$PROJECT_ROOT/helm" \
        --namespace "$NAMESPACE" \
        --set image.repository="$REGISTRY/proxynd" \
        --set image.tag="$IMAGE_TAG" \
        --wait --timeout="$TIMEOUT"
    
    # Green 임시 배포 정리
    helm uninstall "$green_release" -n "$NAMESPACE" || true
    
    success "Blue-Green 배포 완료"
}

# 롤링 배포
deploy_rolling() {
    log "롤링 배포 시작..."
    
    helm upgrade --install "$RELEASE_NAME" "$PROJECT_ROOT/helm" \
        --namespace "$NAMESPACE" \
        --create-namespace \
        --set image.repository="$REGISTRY/proxynd" \
        --set image.tag="$IMAGE_TAG" \
        --wait --timeout="$TIMEOUT"
    
    success "롤링 배포 완료"
}

# 헬스체크
health_check() {
    local deployment_name="${1:-$RELEASE_NAME}"
    log "헬스체크 실행 중: $deployment_name"
    
    # Pod 준비 상태 대기
    kubectl wait --for=condition=ready pod \
        -l app.kubernetes.io/name="$deployment_name" \
        -n "$NAMESPACE" \
        --timeout=300s
    
    # 애플리케이션 헬스체크
    local pod_name
    pod_name=$(kubectl get pods -n "$NAMESPACE" \
        -l app.kubernetes.io/name="$deployment_name" \
        -o jsonpath='{.items[0].metadata.name}')
    
    for i in $(seq 1 "$HEALTH_CHECK_RETRIES"); do
        if kubectl exec -n "$NAMESPACE" "$pod_name" -- \
            curl -f http://localhost:8080/health &> /dev/null; then
            success "헬스체크 성공: $deployment_name"
            return 0
        fi
        
        log "헬스체크 재시도 ($i/$HEALTH_CHECK_RETRIES)..."
        sleep 10
    done
    
    error "헬스체크 실패: $deployment_name"
    return 1
}

# 외부 헬스체크
health_check_external() {
    log "외부 헬스체크 실행 중..."
    
    local ingress_host
    ingress_host=$(kubectl get ingress -n "$NAMESPACE" \
        -l app.kubernetes.io/name="$RELEASE_NAME" \
        -o jsonpath='{.items[0].spec.rules[0].host}' 2>/dev/null || echo "localhost")
    
    for i in $(seq 1 "$HEALTH_CHECK_RETRIES"); do
        if curl -f "http://$ingress_host/health" &> /dev/null; then
            success "외부 헬스체크 성공"
            return 0
        fi
        
        log "외부 헬스체크 재시도 ($i/$HEALTH_CHECK_RETRIES)..."
        sleep 10
    done
    
    error "외부 헬스체크 실패"
    return 1
}

# 롤백
rollback_deployment() {
    log "배포 롤백 시작..."
    
    if [[ -f "$PROJECT_ROOT/.last-backup" ]]; then
        local backup_dir
        backup_dir=$(cat "$PROJECT_ROOT/.last-backup")
        
        if [[ -f "$backup_dir/helm-values.yaml" ]]; then
            helm rollback "$RELEASE_NAME" -n "$NAMESPACE" || {
                # Helm 롤백 실패 시 백업에서 복원
                kubectl apply -f "$backup_dir/current-deployment.yaml"
            }
            success "롤백 완료"
        else
            error "백업 파일을 찾을 수 없습니다: $backup_dir"
        fi
    else
        error "백업 정보를 찾을 수 없습니다"
    fi
}

# 배포 후 검증
post_deployment_validation() {
    log "배포 후 검증 실행 중..."
    
    # 기본 헬스체크
    if ! health_check; then
        error "기본 헬스체크 실패"
        return 1
    fi
    
    # 메트릭 엔드포인트 확인
    local pod_name
    pod_name=$(kubectl get pods -n "$NAMESPACE" \
        -l app.kubernetes.io/name="$RELEASE_NAME" \
        -o jsonpath='{.items[0].metadata.name}')
    
    if ! kubectl exec -n "$NAMESPACE" "$pod_name" -- \
        curl -f http://localhost:8080/metrics &> /dev/null; then
        warn "메트릭 엔드포인트 확인 실패"
    fi
    
    # 기능 테스트 (간단한 프록시 테스트)
    if ! kubectl exec -n "$NAMESPACE" "$pod_name" -- \
        curl -f http://localhost:8080/proxy/maven/org/springframework/spring-core/maven-metadata.xml &> /dev/null; then
        warn "기능 테스트 실패"
    fi
    
    success "배포 후 검증 완료"
}

# 배포 정보 출력
print_deployment_info() {
    log "배포 정보:"
    echo "  네임스페이스: $NAMESPACE"
    echo "  릴리스명: $RELEASE_NAME"
    echo "  이미지: $REGISTRY/proxynd:$IMAGE_TAG"
    echo "  배포 전략: $DEPLOYMENT_STRATEGY"
    echo ""
    
    kubectl get all -n "$NAMESPACE" -l app.kubernetes.io/name="$RELEASE_NAME"
}

# 메인 함수
main() {
    log "ProxyND 프로덕션 배포 시작"
    
    # 사전 조건 확인
    check_prerequisites
    
    # 현재 배포 상태 확인
    if check_current_deployment; then
        create_backup
    fi
    
    # 이미지 검증
    validate_image
    
    # 배포 실행
    case "$DEPLOYMENT_STRATEGY" in
        "blue-green")
            deploy_blue_green
            ;;
        "rolling")
            deploy_rolling
            ;;
        *)
            error "지원하지 않는 배포 전략: $DEPLOYMENT_STRATEGY"
            exit 1
            ;;
    esac
    
    # 배포 후 검증
    post_deployment_validation
    
    # 배포 정보 출력
    print_deployment_info
    
    success "ProxyND 프로덕션 배포 완료!"
}

# 도움말
show_help() {
    cat << EOF
ProxyND 프로덕션 배포 스크립트

사용법:
    $0 [옵션]

환경변수:
    NAMESPACE              Kubernetes 네임스페이스 (기본값: production)
    RELEASE_NAME          Helm 릴리스명 (기본값: proxynd)
    REGISTRY              컨테이너 레지스트리 (기본값: ghcr.io/your-org)
    IMAGE_TAG             이미지 태그 (기본값: latest)
    DEPLOYMENT_STRATEGY   배포 전략 (blue-green|rolling, 기본값: blue-green)
    TIMEOUT               배포 타임아웃 (기본값: 600s)
    SKIP_SECURITY_SCAN    보안 스캔 건너뛰기 (기본값: false)

예시:
    # 기본 배포
    $0
    
    # 특정 태그로 배포
    IMAGE_TAG=v1.2.3 $0
    
    # 롤링 배포
    DEPLOYMENT_STRATEGY=rolling $0
    
    # 보안 스캔 건너뛰기
    SKIP_SECURITY_SCAN=true $0
EOF
}

# 스크립트 인자 처리
if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

# 메인 함수 실행
main "$@"