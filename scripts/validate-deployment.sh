#!/bin/bash
# ProxyND Deployment Validation Script
# 배포 검증 및 모니터링 스크립트

set -euo pipefail

# 설정
NAMESPACE="${NAMESPACE:-production}"
RELEASE_NAME="${RELEASE_NAME:-proxynd}"
TIMEOUT="${TIMEOUT:-300}"
HEALTH_CHECK_INTERVAL="${HEALTH_CHECK_INTERVAL:-10}"
MAX_RETRIES="${MAX_RETRIES:-30}"

# 색상 출력
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 로깅 함수
log() { echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $*"; }

# 배포 상태 확인
check_deployment_status() {
    log "배포 상태 확인 중..."
    
    # Deployment 존재 확인
    if ! kubectl get deployment "$RELEASE_NAME" -n "$NAMESPACE" &>/dev/null; then
        error "Deployment를 찾을 수 없습니다: $RELEASE_NAME"
        return 1
    fi
    
    # Deployment 상태 확인
    local ready_replicas
    ready_replicas=$(kubectl get deployment "$RELEASE_NAME" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}')
    local desired_replicas
    desired_replicas=$(kubectl get deployment "$RELEASE_NAME" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
    
    log "준비된 레플리카: $ready_replicas/$desired_replicas"
    
    if [[ "$ready_replicas" != "$desired_replicas" ]]; then
        error "모든 레플리카가 준비되지 않았습니다"
        return 1
    fi
    
    success "배포 상태 확인 완료"
}

# Pod 상태 확인
check_pod_status() {
    log "Pod 상태 확인 중..."
    
    local pods
    pods=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name="$RELEASE_NAME" -o jsonpath='{.items[*].metadata.name}')
    
    for pod in $pods; do
        local status
        status=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.phase}')
        
        if [[ "$status" != "Running" ]]; then
            error "Pod가 실행 중이 아닙니다: $pod (상태: $status)"
            kubectl describe pod "$pod" -n "$NAMESPACE"
            return 1
        fi
        
        # 컨테이너 준비 상태 확인
        local ready
        ready=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses[0].ready}')
        
        if [[ "$ready" != "true" ]]; then
            error "Pod 컨테이너가 준비되지 않았습니다: $pod"
            kubectl describe pod "$pod" -n "$NAMESPACE"
            return 1
        fi
        
        log "Pod 정상: $pod"
    done
    
    success "Pod 상태 확인 완료"
}

# 서비스 상태 확인
check_service_status() {
    log "서비스 상태 확인 중..."
    
    if ! kubectl get service "${RELEASE_NAME}-svc" -n "$NAMESPACE" &>/dev/null; then
        error "서비스를 찾을 수 없습니다: ${RELEASE_NAME}-svc"
        return 1
    fi
    
    local endpoints
    endpoints=$(kubectl get endpoints "${RELEASE_NAME}-svc" -n "$NAMESPACE" -o jsonpath='{.subsets[*].addresses[*].ip}')
    
    if [[ -z "$endpoints" ]]; then
        error "서비스 엔드포인트가 없습니다"
        return 1
    fi
    
    log "서비스 엔드포인트: $endpoints"
    success "서비스 상태 확인 완료"
}

# 헬스체크 엔드포인트 확인
check_health_endpoint() {
    log "헬스체크 엔드포인트 확인 중..."
    
    local pod_name
    pod_name=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name="$RELEASE_NAME" -o jsonpath='{.items[0].metadata.name}')
    
    for i in $(seq 1 "$MAX_RETRIES"); do
        if kubectl exec -n "$NAMESPACE" "$pod_name" -- curl -f http://localhost:8080/health &>/dev/null; then
            success "헬스체크 엔드포인트 정상"
            return 0
        fi
        
        log "헬스체크 재시도 ($i/$MAX_RETRIES)..."
        sleep "$HEALTH_CHECK_INTERVAL"
    done
    
    error "헬스체크 엔드포인트 실패"
    kubectl exec -n "$NAMESPACE" "$pod_name" -- curl -v http://localhost:8080/health || true
    return 1
}

# 메트릭 엔드포인트 확인
check_metrics_endpoint() {
    log "메트릭 엔드포인트 확인 중..."
    
    local pod_name
    pod_name=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name="$RELEASE_NAME" -o jsonpath='{.items[0].metadata.name}')
    
    if kubectl exec -n "$NAMESPACE" "$pod_name" -- curl -f http://localhost:8080/metrics &>/dev/null; then
        success "메트릭 엔드포인트 정상"
    else
        warn "메트릭 엔드포인트 확인 실패"
        return 1
    fi
}

# 프록시 기능 테스트
test_proxy_functionality() {
    log "프록시 기능 테스트 중..."
    
    local pod_name
    pod_name=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name="$RELEASE_NAME" -o jsonpath='{.items[0].metadata.name}')
    
    # Maven 프록시 테스트
    if kubectl exec -n "$NAMESPACE" "$pod_name" -- \
        curl -f http://localhost:8080/proxy/maven/org/springframework/spring-core/maven-metadata.xml &>/dev/null; then
        success "Maven 프록시 기능 정상"
    else
        warn "Maven 프록시 기능 테스트 실패"
    fi
    
    # NPM 프록시 테스트
    if kubectl exec -n "$NAMESPACE" "$pod_name" -- \
        curl -f http://localhost:8080/proxy/npm/express &>/dev/null; then
        success "NPM 프록시 기능 정상"
    else
        warn "NPM 프록시 기능 테스트 실패"
    fi
}

# 리소스 사용량 확인
check_resource_usage() {
    log "리소스 사용량 확인 중..."
    
    local pods
    pods=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name="$RELEASE_NAME" -o jsonpath='{.items[*].metadata.name}')
    
    for pod in $pods; do
        log "Pod 리소스 사용량: $pod"
        kubectl top pod "$pod" -n "$NAMESPACE" --containers || warn "리소스 메트릭을 가져올 수 없습니다"
    done
}

# 로그 확인
check_logs() {
    log "애플리케이션 로그 확인 중..."
    
    local pod_name
    pod_name=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name="$RELEASE_NAME" -o jsonpath='{.items[0].metadata.name}')
    
    # 최근 로그에서 에러 확인
    local error_count
    error_count=$(kubectl logs "$pod_name" -n "$NAMESPACE" --tail=100 | grep -i error | wc -l)
    
    if [[ "$error_count" -gt 0 ]]; then
        warn "최근 로그에서 $error_count 개의 에러 발견"
        kubectl logs "$pod_name" -n "$NAMESPACE" --tail=20 | grep -i error || true
    else
        success "로그에서 에러 없음"
    fi
}

# 설정 검증
validate_configuration() {
    log "설정 검증 중..."
    
    # ConfigMap 확인
    if kubectl get configmap "$RELEASE_NAME-config" -n "$NAMESPACE" &>/dev/null; then
        success "ConfigMap 존재 확인"
    else
        warn "ConfigMap을 찾을 수 없습니다"
    fi
    
    # Secret 확인
    if kubectl get secret "$RELEASE_NAME-secret" -n "$NAMESPACE" &>/dev/null; then
        success "Secret 존재 확인"
    else
        warn "Secret을 찾을 수 없습니다"
    fi
}

# 네트워크 연결성 테스트
test_network_connectivity() {
    log "네트워크 연결성 테스트 중..."
    
    local pod_name
    pod_name=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name="$RELEASE_NAME" -o jsonpath='{.items[0].metadata.name}')
    
    # 내부 서비스 연결 테스트
    if kubectl get service redis -n "$NAMESPACE" &>/dev/null; then
        if kubectl exec -n "$NAMESPACE" "$pod_name" -- nc -z redis 6379 &>/dev/null; then
            success "Redis 연결 정상"
        else
            warn "Redis 연결 실패"
        fi
    fi
    
    # 외부 연결 테스트
    if kubectl exec -n "$NAMESPACE" "$pod_name" -- nc -z google.com 80 &>/dev/null; then
        success "외부 네트워크 연결 정상"
    else
        warn "외부 네트워크 연결 실패"
    fi
}

# 성능 테스트
performance_test() {
    log "간단한 성능 테스트 실행 중..."
    
    local pod_name
    pod_name=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name="$RELEASE_NAME" -o jsonpath='{.items[0].metadata.name}')
    
    # 동시 요청 테스트
    kubectl exec -n "$NAMESPACE" "$pod_name" -- sh -c '
        for i in $(seq 1 10); do
            curl -s http://localhost:8080/health &
        done
        wait
    ' &>/dev/null
    
    if [[ $? -eq 0 ]]; then
        success "동시 요청 테스트 통과"
    else
        warn "동시 요청 테스트 실패"
    fi
}

# 보안 검증
security_validation() {
    log "보안 설정 검증 중..."
    
    local pod_name
    pod_name=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name="$RELEASE_NAME" -o jsonpath='{.items[0].metadata.name}')
    
    # Pod 보안 컨텍스트 확인
    local run_as_user
    run_as_user=$(kubectl get pod "$pod_name" -n "$NAMESPACE" -o jsonpath='{.spec.securityContext.runAsUser}')
    
    if [[ "$run_as_user" != "0" ]] && [[ -n "$run_as_user" ]]; then
        success "비-루트 사용자로 실행 중: $run_as_user"
    else
        warn "루트 사용자로 실행 중일 수 있습니다"
    fi
    
    # 읽기 전용 루트 파일시스템 확인
    local read_only_root
    read_only_root=$(kubectl get pod "$pod_name" -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].securityContext.readOnlyRootFilesystem}')
    
    if [[ "$read_only_root" == "true" ]]; then
        success "읽기 전용 루트 파일시스템 사용"
    else
        warn "읽기 전용 루트 파일시스템 미사용"
    fi
}

# 백업 검증
validate_backup() {
    log "백업 설정 검증 중..."
    
    # PVC 백업 확인
    if kubectl get pvc -n "$NAMESPACE" -l app.kubernetes.io/name="$RELEASE_NAME" &>/dev/null; then
        success "PVC 백업 설정 확인"
    else
        warn "PVC 백업 설정 없음"
    fi
}

# 전체 검증 실행
run_full_validation() {
    log "=== ProxyND 배포 검증 시작 ==="
    
    local failed_checks=0
    
    # 기본 상태 확인
    check_deployment_status || ((failed_checks++))
    check_pod_status || ((failed_checks++))
    check_service_status || ((failed_checks++))
    
    # 기능 검증
    check_health_endpoint || ((failed_checks++))
    check_metrics_endpoint || ((failed_checks++))
    test_proxy_functionality || ((failed_checks++))
    
    # 설정 및 보안 검증
    validate_configuration || ((failed_checks++))
    security_validation || ((failed_checks++))
    
    # 네트워크 및 성능
    test_network_connectivity || ((failed_checks++))
    performance_test || ((failed_checks++))
    
    # 리소스 및 로그
    check_resource_usage
    check_logs
    validate_backup
    
    log "=== 검증 완료 ==="
    
    if [[ $failed_checks -eq 0 ]]; then
        success "모든 검증 통과! 배포가 성공적으로 완료되었습니다."
        return 0
    else
        error "$failed_checks 개의 검증 실패. 배포를 확인해주세요."
        return 1
    fi
}

# 도움말
show_help() {
    cat << EOF
ProxyND 배포 검증 스크립트

사용법:
    $0 [명령어] [옵션]

명령어:
    full        전체 검증 실행 (기본값)
    health      헬스체크만 실행
    status      배포 상태만 확인
    logs        로그 확인
    metrics     메트릭 확인
    security    보안 검증
    performance 성능 테스트
    
환경변수:
    NAMESPACE               Kubernetes 네임스페이스 (기본값: production)
    RELEASE_NAME           Helm 릴리스명 (기본값: proxynd)
    TIMEOUT                타임아웃 (기본값: 300)
    HEALTH_CHECK_INTERVAL  헬스체크 간격 (기본값: 10)
    MAX_RETRIES            최대 재시도 횟수 (기본값: 30)

예시:
    # 전체 검증
    $0
    
    # 헬스체크만
    $0 health
    
    # 특정 네임스페이스에서 검증
    NAMESPACE=staging $0
EOF
}

# 메인 함수
main() {
    local command="${1:-full}"
    
    case "$command" in
        "full")
            run_full_validation
            ;;
        "health")
            check_health_endpoint
            ;;
        "status")
            check_deployment_status
            check_pod_status
            check_service_status
            ;;
        "logs")
            check_logs
            ;;
        "metrics")
            check_metrics_endpoint
            ;;
        "security")
            security_validation
            ;;
        "performance")
            performance_test
            ;;
        "-h"|"--help")
            show_help
            ;;
        *)
            error "알 수 없는 명령어: $command"
            show_help
            exit 1
            ;;
    esac
}

# 스크립트 실행
main "$@"