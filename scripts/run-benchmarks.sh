#!/bin/bash

# ProxyND 벤치마크 실행 스크립트

set -e

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 기본 설정
BENCHMARK_TIME=${BENCHMARK_TIME:-"30s"}
BENCHMARK_COUNT=${BENCHMARK_COUNT:-1}
OUTPUT_DIR="./reports/benchmarks"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

echo -e "${BLUE}=== ProxyND 벤치마크 실행 ===${NC}"
echo "시간: $(date)"
echo "벤치마크 시간: $BENCHMARK_TIME"
echo "실행 횟수: $BENCHMARK_COUNT"
echo

# 출력 디렉토리 생성
mkdir -p "$OUTPUT_DIR"

# 도움말 함수
show_help() {
    echo "사용법: $0 [옵션] [벤치마크_패턴]"
    echo
    echo "옵션:"
    echo "  -h, --help              이 도움말 표시"
    echo "  -t, --time DURATION     벤치마크 실행 시간 (기본: 30s)"
    echo "  -c, --count N           벤치마크 실행 횟수 (기본: 1)"
    echo "  -o, --output DIR        출력 디렉토리 (기본: ./reports/benchmarks)"
    echo "  -a, --all               모든 벤치마크 실행"
    echo "  -q, --quick             빠른 벤치마크만 실행"
    echo "  --cache                 캐시 벤치마크만 실행"
    echo "  --proxy                 프록시 벤치마크만 실행"
    echo "  --config                설정 벤치마크만 실행"
    echo "  --middleware            미들웨어 벤치마크만 실행"
    echo "  --memory                메모리 사용량 벤치마크 포함"
    echo "  --cpu                   CPU 프로파일링 활성화"
    echo "  --mem                   메모리 프로파일링 활성화"
    echo
    echo "예시:"
    echo "  $0 --all                          # 모든 벤치마크 실행"
    echo "  $0 --cache                        # 캐시 벤치마크만 실행"
    echo "  $0 --quick                        # 빠른 벤치마크만 실행"
    echo "  $0 BenchmarkCacheSimple           # 특정 벤치마크 실행"
    echo "  $0 --time 60s --count 3 --cache   # 캐시 벤치마크를 60초씩 3번 실행"
}

# 벤치마크 실행 함수
run_benchmark() {
    local name="$1"
    local pattern="$2"
    local extra_flags="$3"
    
    echo -e "${YELLOW}=== $name 벤치마크 실행 ===${NC}"
    
    local output_file="$OUTPUT_DIR/benchmark_${name,,}_$TIMESTAMP.txt"
    local cmd="go test -v ./test/benchmark/... -bench=\"$pattern\" -run=^$ -benchtime=$BENCHMARK_TIME -count=$BENCHMARK_COUNT $extra_flags"
    
    echo "명령어: $cmd"
    echo "출력 파일: $output_file"
    echo
    
    if eval "$cmd" | tee "$output_file"; then
        echo -e "${GREEN}✅ $name 벤치마크 완료${NC}"
        
        # 결과 요약 생성
        if command -v benchstat > /dev/null 2>&1; then
            echo -e "${BLUE}📊 결과 요약:${NC}"
            benchstat "$output_file" | head -20
        fi
    else
        echo -e "${RED}❌ $name 벤치마크 실패${NC}"
        return 1
    fi
    echo
}

# 프로파일링 함수
run_with_profiling() {
    local name="$1"
    local pattern="$2"
    local profile_type="$3"
    
    echo -e "${YELLOW}=== $name 프로파일링 ($profile_type) ===${NC}"
    
    local profile_dir="$OUTPUT_DIR/profiles_$TIMESTAMP"
    mkdir -p "$profile_dir"
    
    local profile_file="$profile_dir/${name,,}_${profile_type}.prof"
    local cmd="go test -v ./test/benchmark/... -bench=\"$pattern\" -run=^$ -benchtime=$BENCHMARK_TIME -${profile_type}profile=\"$profile_file\""
    
    echo "프로파일 파일: $profile_file"
    
    if eval "$cmd"; then
        echo -e "${GREEN}✅ $name 프로파일링 완료${NC}"
        echo "프로파일 분석: go tool pprof $profile_file"
    else
        echo -e "${RED}❌ $name 프로파일링 실패${NC}"
        return 1
    fi
    echo
}

# 메인 실행 로직
ENABLE_CPU_PROFILE=false
ENABLE_MEM_PROFILE=false
INCLUDE_MEMORY=false
BENCHMARK_PATTERN=""
QUICK_MODE=false

# 옵션 파싱
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -t|--time)
            BENCHMARK_TIME="$2"
            shift 2
            ;;
        -c|--count)
            BENCHMARK_COUNT="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -a|--all)
            BENCHMARK_PATTERN="Benchmark"
            shift
            ;;
        -q|--quick)
            QUICK_MODE=true
            shift
            ;;
        --cache)
            BENCHMARK_PATTERN="BenchmarkCache"
            shift
            ;;
        --proxy)
            BENCHMARK_PATTERN="BenchmarkProxy"
            shift
            ;;
        --config)
            BENCHMARK_PATTERN="BenchmarkConfig"
            shift
            ;;
        --middleware)
            BENCHMARK_PATTERN="BenchmarkMiddleware"
            shift
            ;;
        --memory)
            INCLUDE_MEMORY=true
            shift
            ;;
        --cpu)
            ENABLE_CPU_PROFILE=true
            shift
            ;;
        --mem)
            ENABLE_MEM_PROFILE=true
            shift
            ;;
        Benchmark*)
            BENCHMARK_PATTERN="$1"
            shift
            ;;
        *)
            echo -e "${RED}알 수 없는 옵션: $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

# 기본 패턴 설정
if [[ -z "$BENCHMARK_PATTERN" ]]; then
    if [[ "$QUICK_MODE" == "true" ]]; then
        BENCHMARK_PATTERN="BenchmarkCacheSimple|BenchmarkMiddlewareStack/NoMiddleware"
    else
        BENCHMARK_PATTERN="Benchmark"
    fi
fi

echo -e "${BLUE}벤치마크 설정:${NC}"
echo "패턴: $BENCHMARK_PATTERN"
echo "시간: $BENCHMARK_TIME"
echo "횟수: $BENCHMARK_COUNT"
echo "출력: $OUTPUT_DIR"
echo

# 의존성 확인
echo -e "${BLUE}의존성 확인 중...${NC}"
if ! go mod verify; then
    echo -e "${RED}❌ Go 모듈 검증 실패${NC}"
    exit 1
fi

if ! go build -v ./test/benchmark/...; then
    echo -e "${RED}❌ 벤치마크 빌드 실패${NC}"
    exit 1
fi

echo -e "${GREEN}✅ 의존성 확인 완료${NC}"
echo

# 벤치마크 실행
if [[ "$QUICK_MODE" == "true" ]]; then
    echo -e "${YELLOW}=== 빠른 벤치마크 모드 ===${NC}"
    run_benchmark "Quick" "$BENCHMARK_PATTERN" "-short"
else
    # 일반 벤치마크 실행
    extra_flags=""
    if [[ "$INCLUDE_MEMORY" == "true" ]]; then
        extra_flags="-benchmem"
    fi
    
    run_benchmark "Main" "$BENCHMARK_PATTERN" "$extra_flags"
    
    # 프로파일링 실행
    if [[ "$ENABLE_CPU_PROFILE" == "true" ]]; then
        run_with_profiling "Main" "$BENCHMARK_PATTERN" "cpu"
    fi
    
    if [[ "$ENABLE_MEM_PROFILE" == "true" ]]; then
        run_with_profiling "Main" "$BENCHMARK_PATTERN" "mem"
    fi
fi

# 최종 요약
echo -e "${GREEN}=== 벤치마크 완료 ===${NC}"
echo "결과 파일들:"
ls -la "$OUTPUT_DIR/"*"$TIMESTAMP"*
echo
echo -e "${BLUE}결과 분석을 위한 명령어:${NC}"
echo "  benchstat $OUTPUT_DIR/benchmark_*_$TIMESTAMP.txt"
echo
echo -e "${BLUE}프로파일 분석을 위한 명령어:${NC}"
if [[ "$ENABLE_CPU_PROFILE" == "true" || "$ENABLE_MEM_PROFILE" == "true" ]]; then
    find "$OUTPUT_DIR" -name "*$TIMESTAMP*.prof" -exec echo "  go tool pprof {}" \;
fi

echo
echo -e "${GREEN}✅ 모든 벤치마크 실행 완료${NC}"