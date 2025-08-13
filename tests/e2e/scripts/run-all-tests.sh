#!/bin/bash
# 스크립트명: 통합 E2E 테스트 실행기
# 용도: 모든 E2E 테스트를 조율하고 실행
# 사용법: run-all-tests.sh [옵션]  
# 예시: run-all-tests.sh --mode parallel --update-fixtures

set -euo pipefail

# Step 4: 테스트 인프라 개선 - 통합 테스트 실행기

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESULTS_DIR="${TEST_RESULTS_DIR:-/results}"
VERBOSE="${VERBOSE:-false}"
CI="${CI:-false}"

# 색상 출력을 위한 함수들
red() { echo -e "\033[31m$1\033[0m"; }
green() { echo -e "\033[32m$1\033[0m"; }
yellow() { echo -e "\033[33m$1\033[0m"; }
blue() { echo -e "\033[34m$1\033[0m"; }

# 로그 함수들
log_info() { echo "ℹ️ $1"; }
log_success() { green "✅ $1"; }
log_warning() { yellow "⚠️ $1"; }
log_error() { red "❌ $1"; }
log_test() { blue "🚀 $1"; }

# 헬프 메시지
show_help() {
    echo "통합 E2E 테스트 실행기"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --mode MODE          Execution mode: parallel, sequential, matrix (default: parallel)"
    echo "  --update-fixtures    Update test fixtures before running tests"  
    echo "  --quick              Run only quick tests"
    echo "  --full               Run comprehensive tests"
    echo "  --type TYPE          Run tests for specific proxy type"
    echo "  --skip-health        Skip initial health checks"
    echo "  --generate-report    Generate HTML report after tests"
    echo "  --ci                 CI mode with stricter requirements"
    echo "  -v, --verbose        Verbose output"
    echo "  -h, --help           Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 --mode parallel --quick"
    echo "  $0 --update-fixtures --full --generate-report"
    echo "  $0 --type npm --verbose"
    echo "  $0 --ci --mode matrix"
}

# 기본값 설정
EXECUTION_MODE="parallel"
UPDATE_FIXTURES=false
TEST_SCOPE="all"
PROXY_TYPE=""
SKIP_HEALTH=false
GENERATE_REPORT=true

# 인수 파싱
while [[ $# -gt 0 ]]; do
    case $1 in
        --mode)
            EXECUTION_MODE="$2"
            shift 2
            ;;
        --update-fixtures)
            UPDATE_FIXTURES=true
            shift
            ;;
        --quick)
            TEST_SCOPE="quick"
            shift
            ;;
        --full)
            TEST_SCOPE="full"
            shift
            ;;
        --type)
            PROXY_TYPE="$2"
            shift 2
            ;;
        --skip-health)
            SKIP_HEALTH=true
            shift
            ;;
        --generate-report)
            GENERATE_REPORT=true
            shift
            ;;
        --ci)
            CI=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
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

# CI 모드 설정
if [[ "$CI" == "true" ]]; then
    log_info "Running in CI mode"
    GENERATE_REPORT=true
    # CI에서는 더 엄격한 검증
    set -x
fi

# 실행 권한 확인 및 부여
ensure_executable() {
    local scripts=(
        "$SCRIPT_DIR/parallel-test-runner.sh"
        "$SCRIPT_DIR/update-fixtures.sh" 
        "$SCRIPT_DIR/test-npm.sh"
        "$SCRIPT_DIR/test-maven.sh"
        "$SCRIPT_DIR/test-docker.sh"
        "$SCRIPT_DIR/test-apt.sh"
        "$SCRIPT_DIR/test-pip.sh"
        "$SCRIPT_DIR/test-yum.sh"
        "$SCRIPT_DIR/test-apk.sh"
    )
    
    for script in "${scripts[@]}"; do
        if [[ -f "$script" ]]; then
            chmod +x "$script"
        fi
    done
    
    log_success "Script permissions ensured"
}

# 환경 검증
verify_environment() {
    log_test "Verifying test environment..."
    
    # 필수 환경 변수 확인
    local required_vars=(
        "PROXYND_HOST"
        "PROXYND_PORT"
    )
    
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            log_error "Required environment variable not set: $var"
            return 1
        fi
    done
    
    # 필수 도구 확인
    local required_tools=(
        "curl"
        "jq"
        "timeout"
    )
    
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" >/dev/null 2>&1; then
            log_error "Required tool not available: $tool"
            return 1
        fi
    done
    
    # 결과 디렉토리 생성
    mkdir -p "$RESULTS_DIR"/{logs,reports,coverage,artifacts}
    
    log_success "Environment verification completed"
}

# ProxyND 헬스 체크
check_proxynd_health() {
    if [[ "$SKIP_HEALTH" == "true" ]]; then
        log_info "Skipping health check as requested"
        return 0
    fi
    
    log_test "Performing comprehensive ProxyND health check..."
    
    local health_url="http://${PROXYND_HOST}:${PROXYND_PORT}/healthz"
    local metrics_url="http://${PROXYND_HOST}:${PROXYND_PORT}/metrics" 
    local max_attempts=20
    local attempt=0
    
    # 기본 헬스 체크
    while [ $attempt -lt $max_attempts ]; do
        if curl -sf "$health_url" >/dev/null 2>&1; then
            log_success "ProxyND basic health check passed"
            break
        fi
        
        attempt=$((attempt + 1))
        log_info "Health check attempt $attempt/$max_attempts..."
        sleep 3
    done
    
    if [ $attempt -eq $max_attempts ]; then
        log_error "ProxyND health check failed after $max_attempts attempts"
        return 1
    fi
    
    # 메트릭스 엔드포인트 확인
    if curl -sf "$metrics_url" >/dev/null 2>&1; then
        log_success "ProxyND metrics endpoint accessible"
    else
        log_warning "ProxyND metrics endpoint not accessible"
    fi
    
    # 프록시 엔드포인트들 확인
    local proxy_types=("npm" "maven" "docker" "apt" "pip" "yum" "apk")
    local healthy_proxies=0
    
    for proxy_type in "${proxy_types[@]}"; do
        local proxy_url="http://${PROXYND_HOST}:${PROXYND_PORT}/proxy/$proxy_type"
        if curl -sf --max-time 5 "$proxy_url" >/dev/null 2>&1; then
            log_success "Proxy endpoint healthy: $proxy_type"
            ((healthy_proxies++))
        else
            log_warning "Proxy endpoint not responding: $proxy_type"
        fi
    done
    
    log_info "Healthy proxy endpoints: $healthy_proxies/${#proxy_types[@]}"
    
    if [ $healthy_proxies -eq 0 ]; then
        log_error "No proxy endpoints are healthy"
        return 1
    fi
    
    log_success "ProxyND comprehensive health check completed"
}

# 테스트 픽스처 업데이트
update_test_fixtures() {
    if [[ "$UPDATE_FIXTURES" == "false" ]]; then
        return 0
    fi
    
    log_test "Updating test fixtures..."
    
    local update_script="$SCRIPT_DIR/update-fixtures.sh"
    if [[ ! -f "$update_script" ]]; then
        log_warning "Fixture update script not found, skipping..."
        return 0
    fi
    
    local update_args="--all"
    if [[ "$VERBOSE" == "true" ]]; then
        update_args="$update_args --verbose"
    fi
    
    if bash "$update_script" $update_args; then
        log_success "Test fixtures updated successfully"
    else
        log_error "Test fixture update failed"
        return 1
    fi
}

# 순차 실행 모드
run_sequential_tests() {
    log_test "Running tests in sequential mode..."
    
    local test_scripts=(
        "test-npm.sh"
        "test-docker.sh"
        "test-maven.sh"
        "test-pip.sh"
        "test-apt.sh"
        "test-yum.sh"
        "test-apk.sh"
    )
    
    local passed=0
    local total=0
    
    for script in "${test_scripts[@]}"; do
        local test_path="$SCRIPT_DIR/$script"
        if [[ ! -f "$test_path" ]]; then
            continue
        fi
        
        local test_name=$(basename "$script" .sh)
        test_name=${test_name#test-}  # remove test- prefix
        
        # 특정 프록시 타입만 실행하는 경우 필터링
        if [[ -n "$PROXY_TYPE" && "$test_name" != "$PROXY_TYPE" ]]; then
            continue
        fi
        
        ((total++))
        log_info "Running test: $test_name"
        
        local log_file="$RESULTS_DIR/logs/${test_name}.log"
        local verbose_flag=""
        if [[ "$VERBOSE" == "true" ]]; then
            verbose_flag="--verbose"
        fi
        
        if timeout 600 bash "$test_path" $verbose_flag > "$log_file" 2>&1; then
            log_success "Test passed: $test_name"
            ((passed++))
        else
            log_error "Test failed: $test_name"
        fi
    done
    
    log_info "Sequential test results: $passed/$total passed"
    return $(( passed == total ? 0 : 1 ))
}

# 병렬 실행 모드
run_parallel_tests() {
    log_test "Running tests in parallel mode..."
    
    local runner_script="$SCRIPT_DIR/parallel-test-runner.sh"
    if [[ ! -f "$runner_script" ]]; then
        log_error "Parallel test runner not found: $runner_script"
        return 1
    fi
    
    local parallel_args=""
    case "$TEST_SCOPE" in
        quick)
            parallel_args="--quick"
            ;;
        full)
            parallel_args="--full"
            ;;
    esac
    
    if [[ -n "$PROXY_TYPE" ]]; then
        parallel_args="$parallel_args --type $PROXY_TYPE"
    fi
    
    if [[ "$VERBOSE" == "true" ]]; then
        parallel_args="$parallel_args --verbose"
    fi
    
    if bash "$runner_script" $parallel_args; then
        log_success "Parallel tests completed successfully"
        return 0
    else
        log_error "Parallel tests failed"
        return 1
    fi
}

# 매트릭스 실행 모드 (CI용)
run_matrix_tests() {
    log_test "Running tests in matrix mode..."
    
    local proxy_types=("npm" "maven" "docker")  # 핵심 프록시만
    local test_modes=("quick" "basic")
    local results=()
    
    for proxy_type in "${proxy_types[@]}"; do
        for mode in "${test_modes[@]}"; do
            log_info "Matrix test: $proxy_type-$mode"
            
            local log_file="$RESULTS_DIR/logs/matrix_${proxy_type}_${mode}.log"
            local result="PASSED"
            
            # 각 조합별 테스트 실행
            if ! PROXY_TYPE="$proxy_type" TEST_SCOPE="$mode" run_parallel_tests > "$log_file" 2>&1; then
                result="FAILED"
            fi
            
            results+=("$proxy_type-$mode: $result")
            log_info "Matrix result: $proxy_type-$mode = $result"
        done
    done
    
    # 매트릭스 결과 요약
    log_info "Matrix test results:"
    for result in "${results[@]}"; do
        if [[ "$result" == *"FAILED"* ]]; then
            log_error "$result"
        else
            log_success "$result"
        fi
    done
    
    # 실패가 있는지 확인
    for result in "${results[@]}"; do
        if [[ "$result" == *"FAILED"* ]]; then
            return 1
        fi
    done
    
    return 0
}

# 보고서 생성
generate_final_report() {
    if [[ "$GENERATE_REPORT" == "false" ]]; then
        return 0
    fi
    
    log_test "Generating final test report..."
    
    local final_report="$RESULTS_DIR/final-report.html"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    cat > "$final_report" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>ProxyND E2E Test Final Report</title>
    <meta charset="utf-8">
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; margin: 0; padding: 20px; background-color: #f8f9fa; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; margin: -30px -30px 30px -30px; border-radius: 8px 8px 0 0; }
        .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin: 30px 0; }
        .stat-card { background: #f8f9fa; padding: 20px; border-radius: 8px; text-align: center; border: 2px solid #e9ecef; }
        .stat-card.passed { border-color: #28a745; background: #d4edda; }
        .stat-card.failed { border-color: #dc3545; background: #f8d7da; }
        .stat-number { font-size: 36px; font-weight: bold; margin: 10px 0; }
        .test-results { margin: 30px 0; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #dee2e6; }
        th { background-color: #495057; color: white; font-weight: 600; }
        .status-passed { color: #28a745; font-weight: bold; }
        .status-failed { color: #dc3545; font-weight: bold; }
        .status-timeout { color: #ffc107; font-weight: bold; }
        .footer { margin-top: 40px; padding-top: 20px; border-top: 1px solid #dee2e6; font-size: 14px; color: #6c757d; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 ProxyND E2E Test Report</h1>
            <p>Comprehensive End-to-End Testing Results</p>
            <p>Generated: $timestamp</p>
            <p>Mode: $EXECUTION_MODE | Scope: $TEST_SCOPE | CI: $CI</p>
        </div>
EOF
    
    # 통계 수집
    local total_tests=0
    local passed_tests=0
    local failed_tests=0
    local timeout_tests=0
    
    if [[ -d "$RESULTS_DIR/reports" ]]; then
        for result_file in "$RESULTS_DIR/reports"/*.result; do
            if [[ -f "$result_file" ]]; then
                ((total_tests++))
                local result=$(cat "$result_file")
                case "$result" in
                    PASSED) ((passed_tests++));;
                    FAILED) ((failed_tests++));;
                    TIMEOUT) ((timeout_tests++));;
                esac
            fi
        done
    fi
    
    local success_rate=0
    if [[ $total_tests -gt 0 ]]; then
        success_rate=$(( passed_tests * 100 / total_tests ))
    fi
    
    # 통계 카드 추가
    cat >> "$final_report" << EOF
        <div class="stats">
            <div class="stat-card">
                <div>Total Tests</div>
                <div class="stat-number">$total_tests</div>
            </div>
            <div class="stat-card passed">
                <div>Passed</div>
                <div class="stat-number">$passed_tests</div>
            </div>
            <div class="stat-card failed">
                <div>Failed</div>
                <div class="stat-number">$failed_tests</div>
            </div>
            <div class="stat-card">
                <div>Success Rate</div>
                <div class="stat-number">${success_rate}%</div>
            </div>
        </div>
        
        <div class="test-results">
            <h2>Test Results Details</h2>
EOF
    
    # 개별 테스트 결과가 있으면 표시
    if [[ $total_tests -gt 0 ]]; then
        cat >> "$final_report" << EOF
            <table>
                <thead>
                    <tr>
                        <th>Test Name</th>
                        <th>Status</th>
                        <th>Duration</th>
                        <th>Log File</th>
                    </tr>
                </thead>
                <tbody>
EOF
        
        for result_file in "$RESULTS_DIR/reports"/*.result; do
            if [[ -f "$result_file" ]]; then
                local test_name=$(basename "$result_file" .result)
                local result=$(cat "$result_file")
                local duration="N/A"
                local timing_file="$RESULTS_DIR/reports/${test_name}.timing"
                
                if [[ -f "$timing_file" ]]; then
                    duration=$(cat "$timing_file")s
                fi
                
                local status_class=""
                case "$result" in
                    PASSED) status_class="status-passed";;
                    FAILED) status_class="status-failed";;
                    TIMEOUT) status_class="status-timeout";;
                esac
                
                cat >> "$final_report" << EOF
                    <tr>
                        <td>$test_name</td>
                        <td class="$status_class">$result</td>
                        <td>$duration</td>
                        <td><a href="logs/${test_name}.log">View Log</a></td>
                    </tr>
EOF
            fi
        done
        
        echo "                </tbody>" >> "$final_report"
        echo "            </table>" >> "$final_report"
    else
        echo "            <p>No detailed test results available.</p>" >> "$final_report"
    fi
    
    cat >> "$final_report" << EOF
        </div>
        
        <div class="footer">
            <p>Report generated by ProxyND E2E Test Suite</p>
            <p>For more information, check individual test logs and reports.</p>
        </div>
    </div>
</body>
</html>
EOF
    
    log_success "Final report generated: $final_report"
}

# 정리 작업
cleanup() {
    log_info "Performing cleanup..."
    
    # 임시 파일 정리
    find "$RESULTS_DIR" -name "*.tmp" -delete 2>/dev/null || true
    
    # 빈 로그 파일 정리
    find "$RESULTS_DIR/logs" -size 0 -delete 2>/dev/null || true
    
    log_success "Cleanup completed"
}

# 메인 실행 함수
main() {
    local start_time=$(date +%s)
    local exit_code=0
    
    log_info "=== ProxyND E2E Test Suite Starting ==="
    log_info "Mode: $EXECUTION_MODE | Scope: $TEST_SCOPE | CI: $CI"
    
    # 환경 설정
    ensure_executable
    verify_environment
    
    # ProxyND 헬스 체크
    if ! check_proxynd_health; then
        log_error "Environment health check failed"
        exit 1
    fi
    
    # 테스트 픽스처 업데이트
    if ! update_test_fixtures; then
        log_warning "Test fixture update failed, continuing anyway..."
    fi
    
    # 테스트 실행
    case "$EXECUTION_MODE" in
        sequential)
            if ! run_sequential_tests; then
                exit_code=1
            fi
            ;;
        parallel)
            if ! run_parallel_tests; then
                exit_code=1
            fi
            ;;
        matrix)
            if ! run_matrix_tests; then
                exit_code=1
            fi
            ;;
        *)
            log_error "Unknown execution mode: $EXECUTION_MODE"
            exit 1
            ;;
    esac
    
    # 보고서 생성
    generate_final_report
    
    # 정리 작업
    cleanup
    
    local end_time=$(date +%s)
    local total_duration=$((end_time - start_time))
    
    log_info "=== E2E Test Suite Completed ==="
    log_info "Total execution time: ${total_duration}s"
    log_info "Results available in: $RESULTS_DIR"
    
    if [[ $exit_code -eq 0 ]]; then
        log_success "🎉 All tests completed successfully!"
    else
        log_error "❌ Some tests failed"
    fi
    
    exit $exit_code
}

# 스크립트 실행
main "$@"