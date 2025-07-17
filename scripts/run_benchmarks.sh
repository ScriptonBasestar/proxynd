#!/bin/bash
# scripts/run_benchmarks.sh - 벤치마크 테스트 실행 스크립트

set -e

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 기본 설정
BENCHMARK_TIME="5s"
OUTPUT_DIR="./reports/benchmarks"
VERBOSE=false
PROFILE=false
COMPARE=false
BASELINE_FILE=""

# 도움말 출력
show_help() {
    cat << EOF
사용법: $0 [OPTIONS] [BENCHMARK_PATTERN]

벤치마크 테스트 실행 스크립트

OPTIONS:
    -h, --help          이 도움말 표시
    -v, --verbose       상세 출력
    -t, --time DURATION 벤치마크 실행 시간 (기본값: 5s)
    -o, --output DIR    출력 디렉토리 (기본값: ./reports/benchmarks)
    -p, --profile       CPU/메모리 프로파일링 활성화
    -c, --compare FILE  이전 벤치마크 결과와 비교
    --proxy             프록시 벤치마크만 실행
    --cache             캐시 벤치마크만 실행
    --middleware        미들웨어 벤치마크만 실행
    --config            설정 벤치마크만 실행
    --all               모든 벤치마크 실행

BENCHMARK_PATTERN:
    특정 벤치마크 패턴 (예: BenchmarkProxy, BenchmarkCache)

예시:
    $0                                    # 기본 벤치마크 실행
    $0 --proxy                           # 프록시 벤치마크만 실행
    $0 --all -t 10s                      # 모든 벤치마크 10초간 실행
    $0 -p BenchmarkProxyRequest          # 프로파일링과 함께 특정 벤치마크 실행
    $0 -c baseline.txt                   # 이전 결과와 비교
EOF
}

# 로그 출력 함수
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

# 옵션 파싱
BENCHMARK_TYPE=""
BENCHMARK_PATTERN=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -t|--time)
            BENCHMARK_TIME="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -p|--profile)
            PROFILE=true
            shift
            ;;
        -c|--compare)
            COMPARE=true
            BASELINE_FILE="$2"
            shift 2
            ;;
        --proxy)
            BENCHMARK_TYPE="proxy"
            shift
            ;;
        --cache)
            BENCHMARK_TYPE="cache"
            shift
            ;;
        --middleware)
            BENCHMARK_TYPE="middleware"
            shift
            ;;
        --config)
            BENCHMARK_TYPE="config"
            shift
            ;;
        --all)
            BENCHMARK_TYPE="all"
            shift
            ;;
        -*)
            log_error "알 수 없는 옵션: $1"
            show_help
            exit 1
            ;;
        *)
            BENCHMARK_PATTERN="$1"
            shift
            ;;
    esac
done

# 출력 디렉토리 생성
mkdir -p "$OUTPUT_DIR"

# 타임스탬프 생성
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="$OUTPUT_DIR/benchmark_${TIMESTAMP}.txt"

# 벤치마크 실행 함수
run_benchmark() {
    local bench_type="$1"
    local pattern="$2"
    local output_file="$3"
    
    log_info "Running $bench_type benchmarks..."
    
    local cmd="go test"
    local bench_path=""
    
    # 벤치마크 경로 설정
    case $bench_type in
        "proxy")
            bench_path="./tests/benchmark/"
            if [[ -n "$pattern" ]]; then
                cmd="$cmd -bench=BenchmarkProxy.*$pattern"
            else
                cmd="$cmd -bench=BenchmarkProxy"
            fi
            ;;
        "cache")
            bench_path="./tests/benchmark/"
            if [[ -n "$pattern" ]]; then
                cmd="$cmd -bench=BenchmarkCache.*$pattern"
            else
                cmd="$cmd -bench=BenchmarkCache"
            fi
            ;;
        "middleware")
            bench_path="./tests/benchmark/"
            if [[ -n "$pattern" ]]; then
                cmd="$cmd -bench=BenchmarkMiddleware.*$pattern"
            else
                cmd="$cmd -bench=BenchmarkMiddleware"
            fi
            ;;
        "config")
            bench_path="./tests/benchmark/"
            if [[ -n "$pattern" ]]; then
                cmd="$cmd -bench=BenchmarkConfig.*$pattern"
            else
                cmd="$cmd -bench=BenchmarkConfig"
            fi
            ;;
        "all")
            bench_path="./tests/benchmark/ ./internal/services/... ./handlers/proxy/... ./middlewares/..."
            if [[ -n "$pattern" ]]; then
                cmd="$cmd -bench=$pattern"
            else
                cmd="$cmd -bench=."
            fi
            ;;
        *)
            bench_path="./tests/benchmark/"
            if [[ -n "$pattern" ]]; then
                cmd="$cmd -bench=$pattern"
            else
                cmd="$cmd -bench=."
            fi
            ;;
    esac
    
    # 벤치마크 옵션 추가
    cmd="$cmd -benchmem -benchtime=$BENCHMARK_TIME"
    
    if [[ "$VERBOSE" == "true" ]]; then
        cmd="$cmd -v"
    fi
    
    # 프로파일링 옵션
    if [[ "$PROFILE" == "true" ]]; then
        local profile_dir="$OUTPUT_DIR/profiles_${TIMESTAMP}"
        mkdir -p "$profile_dir"
        cmd="$cmd -cpuprofile=$profile_dir/cpu.prof -memprofile=$profile_dir/mem.prof"
        log_info "Profiling enabled. Profiles will be saved to $profile_dir"
    fi
    
    # 벤치마크 실행
    log_info "Executing: $cmd $bench_path"
    
    if [[ -n "$output_file" ]]; then
        eval "$cmd $bench_path" | tee "$output_file"
    else
        eval "$cmd $bench_path"
    fi
}

# 벤치마크 비교 함수
compare_benchmarks() {
    local baseline="$1"
    local current="$2"
    
    log_info "Comparing benchmarks..."
    
    if [[ ! -f "$baseline" ]]; then
        log_error "Baseline file not found: $baseline"
        return 1
    fi
    
    if ! command -v benchcmp >/dev/null 2>&1; then
        log_warning "benchcmp not found. Installing..."
        go install golang.org/x/tools/cmd/benchcmp@latest
    fi
    
    local comparison_file="$OUTPUT_DIR/comparison_${TIMESTAMP}.txt"
    
    echo "=== Benchmark Comparison ===" > "$comparison_file"
    echo "Baseline: $baseline" >> "$comparison_file"
    echo "Current:  $current" >> "$comparison_file"
    echo "Date:     $(date)" >> "$comparison_file"
    echo "" >> "$comparison_file"
    
    benchcmp "$baseline" "$current" >> "$comparison_file" 2>&1 || {
        log_warning "benchcmp failed, using simple diff"
        echo "=== Simple Comparison ===" >> "$comparison_file"
        diff "$baseline" "$current" >> "$comparison_file" 2>&1 || true
    }
    
    log_success "Comparison saved to $comparison_file"
    
    # 주요 개선/악화 사항 요약
    echo ""
    log_info "Performance Summary:"
    if command -v benchcmp >/dev/null 2>&1; then
        echo "Notable changes:"
        benchcmp "$baseline" "$current" 2>/dev/null | grep -E "(faster|slower|MB/s)" | head -10 || echo "No significant changes detected"
    fi
}

# 벤치마크 리포트 생성
generate_report() {
    local report_file="$1"
    
    if [[ ! -f "$report_file" ]]; then
        log_error "Report file not found: $report_file"
        return 1
    fi
    
    local summary_file="$OUTPUT_DIR/summary_${TIMESTAMP}.md"
    
    cat > "$summary_file" << EOF
# 벤치마크 테스트 리포트

**실행 시간**: $(date)  
**벤치마크 시간**: $BENCHMARK_TIME  
**벤치마크 타입**: ${BENCHMARK_TYPE:-default}  

## 요약

### 성능 지표

\`\`\`
$(grep -E "^Benchmark" "$report_file" | head -10)
\`\`\`

### 메모리 사용량

\`\`\`
$(grep -E "allocs/op" "$report_file" | head -5)
\`\`\`

## 상세 결과

\`\`\`
$(cat "$report_file")
\`\`\`

---
*Generated by run_benchmarks.sh at $(date)*
EOF
    
    log_success "Summary report saved to $summary_file"
}

# 메인 실행
main() {
    log_info "Starting benchmark tests..."
    log_info "Benchmark time: $BENCHMARK_TIME"
    log_info "Output directory: $OUTPUT_DIR"
    
    # 벤치마크 실행
    if [[ -n "$BENCHMARK_TYPE" ]]; then
        run_benchmark "$BENCHMARK_TYPE" "$BENCHMARK_PATTERN" "$REPORT_FILE"
    else
        run_benchmark "default" "$BENCHMARK_PATTERN" "$REPORT_FILE"
    fi
    
    # 리포트 생성
    if [[ -f "$REPORT_FILE" ]]; then
        generate_report "$REPORT_FILE"
        log_success "Benchmark completed. Report saved to $REPORT_FILE"
    fi
    
    # 비교 실행
    if [[ "$COMPARE" == "true" && -n "$BASELINE_FILE" ]]; then
        compare_benchmarks "$BASELINE_FILE" "$REPORT_FILE"
    fi
    
    # 프로파일링 결과 안내
    if [[ "$PROFILE" == "true" ]]; then
        log_info "Profile analysis commands:"
        echo "  go tool pprof $OUTPUT_DIR/profiles_${TIMESTAMP}/cpu.prof"
        echo "  go tool pprof $OUTPUT_DIR/profiles_${TIMESTAMP}/mem.prof"
    fi
    
    log_success "All benchmarks completed successfully!"
}

# 스크립트 실행
main "$@"