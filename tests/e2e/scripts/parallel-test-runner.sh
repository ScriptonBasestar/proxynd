#!/bin/bash
# 스크립트명: 병렬 테스트 러너
# 용도: E2E 테스트들을 병렬로 실행하여 성능 최적화
# 사용법: parallel-test-runner.sh [옵션]
# 예시: parallel-test-runner.sh --parallel 4 --timeout 300

set -euo pipefail

# Step 4-1: 병렬 테스트 실행 최적화

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESULTS_DIR="${TEST_RESULTS_DIR:-/results}"
PARALLEL_JOBS="${TEST_PARALLEL:-4}"
TIMEOUT="${TEST_TIMEOUT:-600}"  # 10분
VERBOSE="${VERBOSE:-false}"

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
log_test() { blue "🧪 $1"; }

# 헬프 메시지
show_help() {
    echo "병렬 테스트 러너"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --parallel JOBS   Number of parallel jobs (default: 4)"
    echo "  --timeout SECS    Test timeout in seconds (default: 600)"
    echo "  --quick          Run only quick tests"
    echo "  --full           Run all tests including slow ones"
    echo "  --type TYPE      Run tests for specific proxy type"
    echo "  -v, --verbose    Verbose output"
    echo "  -h, --help       Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 --parallel 8 --quick"
    echo "  $0 --type npm --verbose"
    echo "  $0 --full --timeout 1200"
}

# 인수 파싱
TEST_MODE="all"
PROXY_TYPE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --parallel)
            PARALLEL_JOBS="$2"
            shift 2
            ;;
        --timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        --quick)
            TEST_MODE="quick"
            shift
            ;;
        --full)
            TEST_MODE="full"
            shift
            ;;
        --type)
            PROXY_TYPE="$2"
            shift 2
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

# 결과 디렉토리 생성
mkdir -p "$RESULTS_DIR"/{logs,reports,coverage}

log_info "Starting parallel test execution..."
log_info "Parallel jobs: $PARALLEL_JOBS"
log_info "Timeout: ${TIMEOUT}s"
log_info "Test mode: $TEST_MODE"

# 테스트 정의 배열
declare -A TESTS
TESTS["npm"]="$SCRIPT_DIR/test-npm.sh"
TESTS["maven"]="$SCRIPT_DIR/test-maven.sh"
TESTS["docker"]="$SCRIPT_DIR/test-docker.sh"
TESTS["apt"]="$SCRIPT_DIR/test-apt.sh"
TESTS["pip"]="$SCRIPT_DIR/test-pip.sh"
TESTS["yum"]="$SCRIPT_DIR/test-yum.sh"
TESTS["apk"]="$SCRIPT_DIR/test-apk.sh"

# 테스트 우선순위 정의 (빠른 테스트부터)
declare -A TEST_PRIORITY
TEST_PRIORITY["npm"]=1
TEST_PRIORITY["docker"]=2
TEST_PRIORITY["maven"]=3
TEST_PRIORITY["pip"]=4
TEST_PRIORITY["apt"]=5
TEST_PRIORITY["yum"]=6
TEST_PRIORITY["apk"]=7

# 테스트 실행 시간 예상 (초)
declare -A TEST_DURATION
TEST_DURATION["npm"]=60
TEST_DURATION["docker"]=90
TEST_DURATION["maven"]=120
TEST_DURATION["pip"]=45
TEST_DURATION["apt"]=150
TEST_DURATION["yum"]=180
TEST_DURATION["apk"]=120

# 단일 테스트 실행 함수
run_single_test() {
    local test_name="$1"
    local test_script="$2"
    local start_time=$(date +%s)

    log_test "Starting test: $test_name"

    # 결과 파일들
    local log_file="$RESULTS_DIR/logs/${test_name}.log"
    local result_file="$RESULTS_DIR/reports/${test_name}.result"
    local timing_file="$RESULTS_DIR/reports/${test_name}.timing"

    # 테스트 실행 (타임아웃 적용)
    local exit_code=0
    if timeout "$TIMEOUT" bash "$test_script" ${VERBOSE:+--verbose} > "$log_file" 2>&1; then
        echo "PASSED" > "$result_file"
        log_success "Test completed: $test_name"
    else
        exit_code=$?
        if [ $exit_code -eq 124 ]; then
            echo "TIMEOUT" > "$result_file"
            log_error "Test timed out: $test_name"
        else
            echo "FAILED" > "$result_file"
            log_error "Test failed: $test_name"
        fi
    fi

    # 실행 시간 기록
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    echo "$duration" > "$timing_file"

    # Verbose 모드에서 로그 출력
    if [[ "$VERBOSE" == "true" && -f "$log_file" ]]; then
        echo "--- $test_name LOG ---"
        cat "$log_file"
        echo "--- END $test_name LOG ---"
    fi

    return $exit_code
}

# 병렬 테스트 실행 함수
run_parallel_tests() {
    local tests_to_run=()

    # 실행할 테스트 목록 결정
    if [[ -n "$PROXY_TYPE" ]]; then
        if [[ -n "${TESTS[$PROXY_TYPE]:-}" ]]; then
            tests_to_run=("$PROXY_TYPE")
        else
            log_error "Unknown proxy type: $PROXY_TYPE"
            return 1
        fi
    else
        # 우선순위에 따라 테스트 정렬
        for test_name in $(printf '%s\n' "${!TESTS[@]}" | sort -k1,1 -t' ' -n <(for t in "${!TESTS[@]}"; do echo "${TEST_PRIORITY[$t]} $t"; done | sort -n | cut -d' ' -f2)); do
            case "$TEST_MODE" in
                quick)
                    # 빠른 테스트만 (60초 이하)
                    if [[ ${TEST_DURATION[$test_name]} -le 60 ]]; then
                        tests_to_run+=("$test_name")
                    fi
                    ;;
                full|all)
                    tests_to_run+=("$test_name")
                    ;;
            esac
        done
    fi

    if [[ ${#tests_to_run[@]} -eq 0 ]]; then
        log_error "No tests selected for execution"
        return 1
    fi

    log_info "Tests to run: ${tests_to_run[*]}"

    # GNU parallel 사용 (설치되어 있는 경우)
    if command -v parallel >/dev/null 2>&1; then
        log_info "Using GNU parallel for test execution"

        # parallel 함수 내보내기
        export -f run_single_test log_info log_success log_error log_test red green yellow blue
        export RESULTS_DIR TIMEOUT VERBOSE

        # 병렬 실행
        printf '%s\n' "${tests_to_run[@]}" | parallel -j "$PARALLEL_JOBS" --line-buffer \
            'run_single_test {} '"${TESTS[{}]}"
    else
        log_warning "GNU parallel not available, using bash background jobs"

        # 백그라운드 작업으로 병렬 실행
        local pids=()
        local running_jobs=0

        for test_name in "${tests_to_run[@]}"; do
            # 최대 병렬 작업 수 제한
            while [[ $running_jobs -ge $PARALLEL_JOBS ]]; do
                # 완료된 작업 확인
                for i in "${!pids[@]}"; do
                    if ! kill -0 "${pids[$i]}" 2>/dev/null; then
                        unset "pids[$i]"
                        ((running_jobs--))
                    fi
                done
                sleep 1
            done

            # 새로운 테스트 시작
            run_single_test "$test_name" "${TESTS[$test_name]}" &
            pids+=($!)
            ((running_jobs++))
        done

        # 모든 작업 완료 대기
        for pid in "${pids[@]}"; do
            wait "$pid" || true
        done
    fi
}

# 결과 수집 및 보고서 생성
generate_report() {
    log_info "Generating test report..."

    local report_file="$RESULTS_DIR/test-report.html"
    local summary_file="$RESULTS_DIR/test-summary.txt"

    # 결과 수집
    local total_tests=0
    local passed_tests=0
    local failed_tests=0
    local timeout_tests=0
    local total_duration=0

    cat > "$summary_file" << EOF
=== E2E Test Execution Summary ===
Execution Time: $(date)
Test Mode: $TEST_MODE
Parallel Jobs: $PARALLEL_JOBS
Timeout: ${TIMEOUT}s

=== Individual Test Results ===
EOF

    for result_file in "$RESULTS_DIR/reports"/*.result; do
        if [[ -f "$result_file" ]]; then
            local test_name=$(basename "$result_file" .result)
            local result=$(cat "$result_file")
            local timing_file="$RESULTS_DIR/reports/${test_name}.timing"
            local duration=0

            if [[ -f "$timing_file" ]]; then
                duration=$(cat "$timing_file")
                total_duration=$((total_duration + duration))
            fi

            ((total_tests++))

            case "$result" in
                PASSED)
                    ((passed_tests++))
                    echo "✅ $test_name: PASSED (${duration}s)" >> "$summary_file"
                    ;;
                FAILED)
                    ((failed_tests++))
                    echo "❌ $test_name: FAILED (${duration}s)" >> "$summary_file"
                    ;;
                TIMEOUT)
                    ((timeout_tests++))
                    echo "⏰ $test_name: TIMEOUT (${duration}s)" >> "$summary_file"
                    ;;
            esac
        fi
    done

    # 요약 정보 추가
    cat >> "$summary_file" << EOF

=== Summary Statistics ===
Total Tests: $total_tests
Passed: $passed_tests
Failed: $failed_tests
Timeout: $timeout_tests
Success Rate: $(( total_tests > 0 ? passed_tests * 100 / total_tests : 0 ))%
Total Duration: ${total_duration}s
Average Duration: $(( total_tests > 0 ? total_duration / total_tests : 0 ))s
EOF

    # HTML 보고서 생성
    cat > "$report_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>ProxyND E2E Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { background: #f5f5f5; padding: 20px; border-radius: 5px; }
        .stats { display: flex; gap: 20px; margin: 20px 0; }
        .stat { background: #e9f7ef; padding: 15px; border-radius: 5px; text-align: center; }
        .passed { color: green; }
        .failed { color: red; }
        .timeout { color: orange; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h1>ProxyND E2E Test Report</h1>
        <p>Generated: $(date)</p>
        <p>Mode: $TEST_MODE | Parallel Jobs: $PARALLEL_JOBS | Timeout: ${TIMEOUT}s</p>
    </div>

    <div class="stats">
        <div class="stat">
            <h3>Total Tests</h3>
            <div style="font-size: 24px;">$total_tests</div>
        </div>
        <div class="stat">
            <h3 class="passed">Passed</h3>
            <div style="font-size: 24px;">$passed_tests</div>
        </div>
        <div class="stat">
            <h3 class="failed">Failed</h3>
            <div style="font-size: 24px;">$failed_tests</div>
        </div>
        <div class="stat">
            <h3>Success Rate</h3>
            <div style="font-size: 24px;">$(( total_tests > 0 ? passed_tests * 100 / total_tests : 0 ))%</div>
        </div>
    </div>

    <table>
        <tr>
            <th>Test Name</th>
            <th>Result</th>
            <th>Duration (s)</th>
            <th>Log</th>
        </tr>
EOF

    # 각 테스트 결과를 테이블에 추가
    for result_file in "$RESULTS_DIR/reports"/*.result; do
        if [[ -f "$result_file" ]]; then
            local test_name=$(basename "$result_file" .result)
            local result=$(cat "$result_file")
            local timing_file="$RESULTS_DIR/reports/${test_name}.timing"
            local duration=0

            if [[ -f "$timing_file" ]]; then
                duration=$(cat "$timing_file")
            fi

            local result_class="passed"
            if [[ "$result" == "FAILED" ]]; then
                result_class="failed"
            elif [[ "$result" == "TIMEOUT" ]]; then
                result_class="timeout"
            fi

            cat >> "$report_file" << EOF
        <tr>
            <td>$test_name</td>
            <td class="$result_class">$result</td>
            <td>$duration</td>
            <td><a href="logs/${test_name}.log">View Log</a></td>
        </tr>
EOF
        fi
    done

    cat >> "$report_file" << EOF
    </table>
</body>
</html>
EOF

    # 결과 출력
    echo ""
    cat "$summary_file"
    echo ""
    log_success "Test report generated: $report_file"
    log_success "Summary available: $summary_file"

    # 헬스 체크 파일 생성
    touch "$RESULTS_DIR/health"

    # 전체 성공률 반환
    return $(( total_tests > 0 && passed_tests == total_tests ? 0 : 1 ))
}

# 메인 실행 함수
main() {
    local start_time=$(date +%s)

    log_info "=== Parallel E2E Test Execution Starting ==="

    # ProxyND 헬스 체크
    log_test "Checking ProxyND health..."
    if ! curl -sf "http://${PROXYND_HOST:-proxynd}:${PROXYND_PORT:-8080}/healthz" >/dev/null 2>&1; then
        log_error "ProxyND is not healthy, cannot proceed with tests"
        return 1
    fi
    log_success "ProxyND is healthy"

    # 병렬 테스트 실행
    run_parallel_tests

    # 결과 보고서 생성
    generate_report
    local report_result=$?

    local end_time=$(date +%s)
    local total_duration=$((end_time - start_time))

    log_info "=== Test Execution Completed ==="
    log_info "Total execution time: ${total_duration}s"

    if [[ $report_result -eq 0 ]]; then
        log_success "All tests passed successfully!"
        return 0
    else
        log_error "Some tests failed or timed out"
        return 1
    fi
}

# 스크립트 실행
main "$@"
