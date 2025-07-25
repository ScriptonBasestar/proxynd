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
        docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \\\n            aquasec/trivy:latest image --exit-code 1 --severity HIGH,CRITICAL \\\n            "$REGISTRY/proxynd:$IMAGE_TAG" || {\n            error \"보안 스캔 실패. SKIP_SECURITY_SCAN=true로 건너뛸 수 있습니다\"\n            exit 1\n        }\n    fi\n    \n    success \"이미지 검증 완료\"\n}\n\n# Blue-Green 배포\ndeploy_blue_green() {\n    log \"Blue-Green 배포 시작...\"\n    \n    local green_release=\"${RELEASE_NAME}-green\"\n    \n    # Green 슬롯에 새 버전 배포\n    log \"Green 슬롯에 새 버전 배포 중...\"\n    helm upgrade --install \"$green_release\" \"$PROJECT_ROOT/helm\" \\\n        --namespace \"$NAMESPACE\" \\\n        --create-namespace \\\n        --set image.repository=\"$REGISTRY/proxynd\" \\\n        --set image.tag=\"$IMAGE_TAG\" \\\n        --set nameOverride=\"$green_release\" \\\n        --set service.name=\"${RELEASE_NAME}-green-svc\" \\\n        --set ingress.enabled=false \\\n        --wait --timeout=\"$TIMEOUT\"\n    \n    # Green 슬롯 헬스체크\n    if ! health_check \"$green_release\"; then\n        error \"Green 슬롯 헬스체크 실패\"\n        rollback_deployment\n        exit 1\n    fi\n    \n    # 트래픽 전환\n    log \"트래픽을 Green 슬롯으로 전환 중...\"\n    kubectl patch service \"${RELEASE_NAME}-svc\" -n \"$NAMESPACE\" \\\n        -p '{\"spec\":{\"selector\":{\"app.kubernetes.io/name\":\"'\"$green_release\"'\"}}}'\n    \n    # 전환 후 헬스체크\n    sleep 30\n    if ! health_check_external; then\n        error \"트래픽 전환 후 헬스체크 실패\"\n        # 트래픽을 다시 Blue로 전환\n        kubectl patch service \"${RELEASE_NAME}-svc\" -n \"$NAMESPACE\" \\\n            -p '{\"spec\":{\"selector\":{\"app.kubernetes.io/name\":\"'\"$RELEASE_NAME\"'\"}}}'\n        rollback_deployment\n        exit 1\n    fi\n    \n    # Blue 슬롯 정리\n    log \"Blue 슬롯 정리 중...\"\n    helm uninstall \"$RELEASE_NAME\" -n \"$NAMESPACE\" || true\n    \n    # Green을 새로운 Blue로 이름 변경\n    log \"Green 슬롯을 기본 배포로 승격 중...\"\n    helm upgrade --install \"$RELEASE_NAME\" \"$PROJECT_ROOT/helm\" \\\n        --namespace \"$NAMESPACE\" \\\n        --set image.repository=\"$REGISTRY/proxynd\" \\\n        --set image.tag=\"$IMAGE_TAG\" \\\n        --wait --timeout=\"$TIMEOUT\"\n    \n    # Green 임시 배포 정리\n    helm uninstall \"$green_release\" -n \"$NAMESPACE\" || true\n    \n    success \"Blue-Green 배포 완료\"\n}\n\n# 롤링 배포\ndeploy_rolling() {\n    log \"롤링 배포 시작...\"\n    \n    helm upgrade --install \"$RELEASE_NAME\" \"$PROJECT_ROOT/helm\" \\\n        --namespace \"$NAMESPACE\" \\\n        --create-namespace \\\n        --set image.repository=\"$REGISTRY/proxynd\" \\\n        --set image.tag=\"$IMAGE_TAG\" \\\n        --wait --timeout=\"$TIMEOUT\"\n    \n    success \"롤링 배포 완료\"\n}\n\n# 헬스체크\nhealth_check() {\n    local deployment_name=\"${1:-$RELEASE_NAME}\"\n    log \"헬스체크 실행 중: $deployment_name\"\n    \n    # Pod 준비 상태 대기\n    kubectl wait --for=condition=ready pod \\\n        -l app.kubernetes.io/name=\"$deployment_name\" \\\n        -n \"$NAMESPACE\" \\\n        --timeout=300s\n    \n    # 애플리케이션 헬스체크\n    local pod_name\n    pod_name=$(kubectl get pods -n \"$NAMESPACE\" \\\n        -l app.kubernetes.io/name=\"$deployment_name\" \\\n        -o jsonpath='{.items[0].metadata.name}')\n    \n    for i in $(seq 1 \"$HEALTH_CHECK_RETRIES\"); do\n        if kubectl exec -n \"$NAMESPACE\" \"$pod_name\" -- \\\n            curl -f http://localhost:8080/health &> /dev/null; then\n            success \"헬스체크 성공: $deployment_name\"\n            return 0\n        fi\n        \n        log \"헬스체크 재시도 ($i/$HEALTH_CHECK_RETRIES)...\"\n        sleep 10\n    done\n    \n    error \"헬스체크 실패: $deployment_name\"\n    return 1\n}\n\n# 외부 헬스체크\nhealth_check_external() {\n    log \"외부 헬스체크 실행 중...\"\n    \n    local ingress_host\n    ingress_host=$(kubectl get ingress -n \"$NAMESPACE\" \\\n        -l app.kubernetes.io/name=\"$RELEASE_NAME\" \\\n        -o jsonpath='{.items[0].spec.rules[0].host}' 2>/dev/null || echo \"localhost\")\n    \n    for i in $(seq 1 \"$HEALTH_CHECK_RETRIES\"); do\n        if curl -f \"http://$ingress_host/health\" &> /dev/null; then\n            success \"외부 헬스체크 성공\"\n            return 0\n        fi\n        \n        log \"외부 헬스체크 재시도 ($i/$HEALTH_CHECK_RETRIES)...\"\n        sleep 10\n    done\n    \n    error \"외부 헬스체크 실패\"\n    return 1\n}\n\n# 롤백\nrollback_deployment() {\n    log \"배포 롤백 시작...\"\n    \n    if [[ -f \"$PROJECT_ROOT/.last-backup\" ]]; then\n        local backup_dir\n        backup_dir=$(cat \"$PROJECT_ROOT/.last-backup\")\n        \n        if [[ -f \"$backup_dir/helm-values.yaml\" ]]; then\n            helm rollback \"$RELEASE_NAME\" -n \"$NAMESPACE\" || {\n                # Helm 롤백 실패 시 백업에서 복원\n                kubectl apply -f \"$backup_dir/current-deployment.yaml\"\n            }\n            success \"롤백 완료\"\n        else\n            error \"백업 파일을 찾을 수 없습니다: $backup_dir\"\n        fi\n    else\n        error \"백업 정보를 찾을 수 없습니다\"\n    fi\n}\n\n# 배포 후 검증\npost_deployment_validation() {\n    log \"배포 후 검증 실행 중...\"\n    \n    # 기본 헬스체크\n    if ! health_check; then\n        error \"기본 헬스체크 실패\"\n        return 1\n    fi\n    \n    # 메트릭 엔드포인트 확인\n    local pod_name\n    pod_name=$(kubectl get pods -n \"$NAMESPACE\" \\\n        -l app.kubernetes.io/name=\"$RELEASE_NAME\" \\\n        -o jsonpath='{.items[0].metadata.name}')\n    \n    if ! kubectl exec -n \"$NAMESPACE\" \"$pod_name\" -- \\\n        curl -f http://localhost:8080/metrics &> /dev/null; then\n        warn \"메트릭 엔드포인트 확인 실패\"\n    fi\n    \n    # 기능 테스트 (간단한 프록시 테스트)\n    if ! kubectl exec -n \"$NAMESPACE\" \"$pod_name\" -- \\\n        curl -f http://localhost:8080/proxy/maven/org/springframework/spring-core/maven-metadata.xml &> /dev/null; then\n        warn \"기능 테스트 실패\"\n    fi\n    \n    success \"배포 후 검증 완료\"\n}\n\n# 배포 정보 출력\nprint_deployment_info() {\n    log \"배포 정보:\"\n    echo \"  네임스페이스: $NAMESPACE\"\n    echo \"  릴리스명: $RELEASE_NAME\"\n    echo \"  이미지: $REGISTRY/proxynd:$IMAGE_TAG\"\n    echo \"  배포 전략: $DEPLOYMENT_STRATEGY\"\n    echo \"\"\n    \n    kubectl get all -n \"$NAMESPACE\" -l app.kubernetes.io/name=\"$RELEASE_NAME\"\n}\n\n# 메인 함수\nmain() {\n    log \"ProxyND 프로덕션 배포 시작\"\n    \n    # 사전 조건 확인\n    check_prerequisites\n    \n    # 현재 배포 상태 확인\n    if check_current_deployment; then\n        create_backup\n    fi\n    \n    # 이미지 검증\n    validate_image\n    \n    # 배포 실행\n    case \"$DEPLOYMENT_STRATEGY\" in\n        \"blue-green\")\n            deploy_blue_green\n            ;;\n        \"rolling\")\n            deploy_rolling\n            ;;\n        *)\n            error \"지원하지 않는 배포 전략: $DEPLOYMENT_STRATEGY\"\n            exit 1\n            ;;\n    esac\n    \n    # 배포 후 검증\n    post_deployment_validation\n    \n    # 배포 정보 출력\n    print_deployment_info\n    \n    success \"ProxyND 프로덕션 배포 완료!\"\n}\n\n# 도움말\nshow_help() {\n    cat << EOF\nProxyND 프로덕션 배포 스크립트\n\n사용법:\n    $0 [옵션]\n\n환경변수:\n    NAMESPACE              Kubernetes 네임스페이스 (기본값: production)\n    RELEASE_NAME          Helm 릴리스명 (기본값: proxynd)\n    REGISTRY              컨테이너 레지스트리 (기본값: ghcr.io/your-org)\n    IMAGE_TAG             이미지 태그 (기본값: latest)\n    DEPLOYMENT_STRATEGY   배포 전략 (blue-green|rolling, 기본값: blue-green)\n    TIMEOUT               배포 타임아웃 (기본값: 600s)\n    SKIP_SECURITY_SCAN    보안 스캔 건너뛰기 (기본값: false)\n\n예시:\n    # 기본 배포\n    $0\n    \n    # 특정 태그로 배포\n    IMAGE_TAG=v1.2.3 $0\n    \n    # 롤링 배포\n    DEPLOYMENT_STRATEGY=rolling $0\n    \n    # 보안 스캔 건너뛰기\n    SKIP_SECURITY_SCAN=true $0\nEOF\n}\n\n# 스크립트 인자 처리\nif [[ \"${1:-}\" == \"-h\" || \"${1:-}\" == \"--help\" ]]; then\n    show_help\n    exit 0\nfi\n\n# 메인 함수 실행\nmain \"$@\""