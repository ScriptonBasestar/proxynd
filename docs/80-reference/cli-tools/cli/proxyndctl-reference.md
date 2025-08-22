# ProxyND CLI (proxyndctl) 전체 참조 문서

`proxyndctl`은 ProxyND 서버를 관리하기 위한 강력한 명령줄 도구입니다.

## 목차

1. [설치 및 설정](#설치-및-설정)
2. [전역 옵션](#전역-옵션)
3. [캐시 관리](#캐시-관리)
4. [설정 관리](#설정-관리)
5. [사용자 관리](#사용자-관리)
6. [서버 상태 관리](#서버-상태-관리)
7. [프록시 테스트](#프록시-테스트)
8. [Maven 전용 기능](#maven-전용-기능)
   - [Maven 인덱스 관리](#maven-인덱스-관리)
   - [Maven 백업 관리](#maven-백업-관리)
9. [문서 생성](#문서-생성)
10. [자동완성](#자동완성)
11. [실전 예제](#실전-예제)
12. [문제 해결](#문제-해결)

## 설치 및 설정

### 바이너리 설치

```bash
# 소스에서 빌드
go build -o proxyndctl ./cmd/proxyndctl
sudo mv proxyndctl /usr/local/bin/

# 또는 make 사용
make build-cli
sudo make install-cli
```

### 자동완성 설치

```bash
# Bash
proxyndctl completion bash > /etc/bash_completion.d/proxyndctl

# Zsh
proxyndctl completion zsh > "${fpath[1]}/_proxyndctl"

# Fish
proxyndctl completion fish > ~/.config/fish/completions/proxyndctl.fish
```

### 환경 변수

```bash
export PROXYNDCTL_SERVER="http://localhost:8080"  # 기본 서버 주소
export PROXYNDCTL_FORMAT="table"                   # 기본 출력 형식
export PROXYNDCTL_TIMEOUT="30s"                    # 기본 타임아웃
```

## 전역 옵션

모든 명령어에서 사용 가능한 옵션:

```bash
--server string    ProxyND 서버 URL (기본값: "http://localhost:8080")
--timeout string   요청 타임아웃 (기본값: "30s")
--format string    출력 포맷 (table, json, yaml) (기본값: "table")
-v, --verbose      상세 출력
-h, --help         도움말 표시
--version          버전 정보 표시
```

## 캐시 관리

### cache list - 캐시 목록 조회

```bash
# 기본 사용
proxyndctl cache list

# 프록시 타입별 필터링
proxyndctl cache list --type maven
proxyndctl cache list -t apt

# 패턴 필터링
proxyndctl cache list --pattern "*.jar"
proxyndctl cache list -p "spring-*"

# 정렬 옵션
proxyndctl cache list --sort size    # 크기순
proxyndctl cache list --sort time    # 시간순
proxyndctl cache list --sort name    # 이름순

# 결과 제한
proxyndctl cache list --limit 20

# JSON 출력
proxyndctl cache list --format json
```

### cache size - 캐시 크기 확인

```bash
# 전체 크기
proxyndctl cache size

# 프록시별 크기
proxyndctl cache size --type maven
proxyndctl cache size -t npm

# 사람이 읽기 쉬운 형식
proxyndctl cache size --human-readable
proxyndctl cache size -h
```

### cache clear - 캐시 삭제

```bash
# 전체 캐시 삭제 (확인 필요)
proxyndctl cache clear

# 강제 삭제
proxyndctl cache clear --force
proxyndctl cache clear -f

# 특정 프록시만 삭제
proxyndctl cache clear --type maven -f

# 패턴으로 삭제
proxyndctl cache clear --pattern "*.tmp" -f

# 오래된 파일만 삭제
proxyndctl cache clear --older-than 30d -f
```

## 설정 관리

### config validate - 설정 검증

```bash
# 모든 설정 파일 검증
proxyndctl config validate

# 특정 파일만 검증
proxyndctl config validate --file maven-proxy.yaml
proxyndctl config validate -f global.yaml

# 상세 검증
proxyndctl config validate --verbose
```

### config show - 설정 조회

```bash
# 설정 파일 목록
proxyndctl config show

# 특정 설정 파일 내용
proxyndctl config show maven-proxy.yaml

# 병합된 최종 설정
proxyndctl config show --merged

# YAML 형식으로 출력
proxyndctl config show --format yaml
```

## 사용자 관리

### user list - 사용자 목록

```bash
# 기본 목록
proxyndctl user list

# 상세 정보
proxyndctl user list --detail
proxyndctl user list -d

# JSON 출력
proxyndctl user list --format json
```

### user add - 사용자 추가

```bash
# 명령줄 옵션 사용
proxyndctl user add --username john --password secret123 --role admin --description "관리자"
proxyndctl user add -u jane -p pass456 -r user -d "일반 사용자"

# 대화형 모드
proxyndctl user add --interactive
proxyndctl user add -i
```

### user delete - 사용자 삭제

```bash
# 확인 후 삭제
proxyndctl user delete john

# 강제 삭제
proxyndctl user delete jane --force
```

### user info - 사용자 정보

```bash
proxyndctl user info john
```

## 서버 상태 관리

### status - 서버 상태

```bash
# 기본 상태
proxyndctl status

# 상세 정보
proxyndctl status --detail
proxyndctl status -d

# JSON 출력
proxyndctl status --format json
```

### health - 헬스체크

```bash
# 기본 헬스체크
proxyndctl health

# 의존성 포함
proxyndctl health --dependencies
proxyndctl health -d
```

### metrics - 메트릭 조회

```bash
# 전체 메트릭
proxyndctl metrics

# 카테고리별
proxyndctl metrics --category cache
proxyndctl metrics -c system
proxyndctl metrics -c proxy

# Prometheus 형식
proxyndctl metrics --prometheus
```

## 프록시 테스트

### test - 프록시 테스트

```bash
# 모든 프록시 테스트
proxyndctl test all

# 특정 프록시 테스트
proxyndctl test maven
proxyndctl test apt
proxyndctl test npm

# 상세 결과
proxyndctl test all --detail

# 타임아웃 설정
proxyndctl test all --timeout 60s

# 특정 URL 테스트
proxyndctl test maven --url /proxy/maven/org/springframework/spring-core/maven-metadata.xml
```

## Maven 전용 기능

### Maven 인덱스 관리

Maven 리포지토리의 검색 인덱스를 관리합니다.

#### maven-index build - 인덱스 빌드

```bash
# 기본 빌드 (설정에서 경로 읽기)
proxyndctl maven-index build

# 강제 재빌드
proxyndctl maven-index build --force
proxyndctl maven-index build -f

# 사용자 지정 저장 경로
proxyndctl maven-index build --storage /custom/path/maven-index

# 상세 진행 상황
proxyndctl maven-index build --verbose
proxyndctl maven-index build -v

# 예제: 첫 인덱스 생성
proxyndctl maven-index build -v
# Building Maven search index...
# Storage directory: /storage/maven-index
# Configured mirrors: 5
# [1/321] Indexing org...
# [2/321] Indexing com...
# ...
# Index build completed!
# Total entries: 145231
# Time taken: 5m32s
# Index file size: 23.5 MB
```

#### maven-index info - 인덱스 정보

```bash
# 기본 정보
proxyndctl maven-index info

# 사용자 지정 경로
proxyndctl maven-index info --storage /custom/path/maven-index

# 샘플 엔트리 표시
proxyndctl maven-index info --verbose

# 출력 예제:
# Index file: /storage/maven-index/maven-search-index.json
# Size: 23.5 MB
# Last modified: 2024-01-15 10:30:45 (2 hours ago)
# Total entries: 145231
# Entry types:
#   group: 4521
#   artifact: 32154
#   version: 108556
```

#### maven-index search - 인덱스 검색

```bash
# 기본 검색
proxyndctl maven-index search spring

# 모든 결과 표시
proxyndctl maven-index search spring --verbose

# 사용자 지정 경로
proxyndctl maven-index search jackson --storage /custom/path/maven-index

# 출력 예제:
# Searching for: spring
#
# [group] /proxy/maven/org/springframework/
#   Group: org.springframework
#
# [artifact] /proxy/maven/org/springframework/spring-core/
#   Group: org.springframework
#   Artifact: spring-core
#
# Found 1523 matches
```

### Maven 백업 관리

Maven 리포지토리의 증분백업을 수행합니다.

#### maven-backup create - 백업 생성

```bash
# 기본 백업 (설정에서 소스 경로 읽기)
proxyndctl maven-backup create --target /backup/maven

# 소스 경로 지정
proxyndctl maven-backup create --source /cache/maven --target /backup/maven

# 체크섬 기반 백업 (정확하지만 느림)
proxyndctl maven-backup create --target /backup/maven --checksum

# 타임스탬프 기반 백업 (빠르지만 덜 정확)
proxyndctl maven-backup create --target /backup/maven --no-checksum

# 병렬 워커 수 조정
proxyndctl maven-backup create --target /backup/maven --workers 8

# 진행 상황 예제:
# Starting incremental backup...
# Source: /storage/cache/maven
# Target: /backup/maven-2024-01-15
# Use checksum: true
# Workers: 4
#
# [1523/45231] org/springframework/spring-core/5.3.23/spring-core-5.3.23.jar
# [1524/45231] org/springframework/spring-core/5.3.23/spring-core-5.3.23.pom
# ...
#
# Backup completed successfully!
# Backup ID: backup-1705300245
# Total files: 145231
# Total size: 8.3 GB
# Duration: 12m45s
```

#### maven-backup restore - 백업 복원

```bash
# 기본 복원
proxyndctl maven-backup restore --source /backup/maven --target /cache/maven

# 기존 파일 덮어쓰기
proxyndctl maven-backup restore --source /backup/maven --target /cache/maven --overwrite

# 검증 없이 빠른 복원
proxyndctl maven-backup restore --source /backup/maven --target /cache/maven --no-verify

# 진행 상황 예제:
# Starting restore...
# Backup source: /backup/maven-2024-01-15
# Restore target: /cache/maven
# Overwrite: false
# Verify: true
#
# [523/145231] org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar
# ...
#
# Restore completed successfully!
# Duration: 8m32s
```

#### maven-backup info - 백업 정보

```bash
# 백업 정보 조회
proxyndctl maven-backup info --backup /backup/maven

# 출력 예제:
# Backup Information
# ==================
# Backup ID: backup-1705300245
# Last backup: 2024-01-15 10:30:45
# Total files: 145231
# Total size: 8.3 GB
# Files tracked: 145231
# Backup age: 2h15m
```

## 문서 생성

### docs - 문서 생성

```bash
# Man page 생성
proxyndctl docs man ./man

# Markdown 문서
proxyndctl docs markdown ./docs

# RestructuredText
proxyndctl docs rest ./docs/source

# YAML (자동화용)
proxyndctl docs yaml ./automation
```

## 자동완성

### completion - 자동완성 스크립트

```bash
# Bash
proxyndctl completion bash

# Zsh
proxyndctl completion zsh

# Fish
proxyndctl completion fish

# PowerShell
proxyndctl completion powershell
```

## 실전 예제

### 일일 백업 스크립트

```bash
#!/bin/bash
# Maven 리포지토리 일일 증분백업

DATE=$(date +%Y%m%d)
BACKUP_DIR="/backup/maven-$DATE"

echo "=== Maven 리포지토리 백업 시작 ==="

# 1. 캐시 상태 확인
echo "현재 캐시 상태:"
proxyndctl cache size -t maven -h

# 2. 인덱스 업데이트 (선택사항)
echo "검색 인덱스 업데이트 중..."
proxyndctl maven-index build --force

# 3. 증분백업 수행
echo "증분백업 시작..."
proxyndctl maven-backup create \
    --target "$BACKUP_DIR" \
    --checksum \
    --workers 8

# 4. 백업 정보 저장
proxyndctl maven-backup info --backup "$BACKUP_DIR" > "$BACKUP_DIR/backup-info.txt"

# 5. 오래된 백업 정리 (30일 이상)
find /backup -name "maven-*" -type d -mtime +30 -exec rm -rf {} \;

echo "=== 백업 완료 ==="
```

### 캐시 모니터링 스크립트

```bash
#!/bin/bash
# ProxyND 캐시 모니터링 및 알림

# 임계값 설정
CACHE_SIZE_LIMIT=$((50 * 1024 * 1024 * 1024))  # 50GB
CACHE_HIT_RATE_MIN=0.7  # 70%

# 캐시 크기 확인
CACHE_INFO=$(proxyndctl cache size --format json)
TOTAL_SIZE=$(echo $CACHE_INFO | jq -r '.total_size')

if [ $TOTAL_SIZE -gt $CACHE_SIZE_LIMIT ]; then
    echo "경고: 캐시 크기가 임계값을 초과했습니다!"
    echo "현재 크기: $(proxyndctl cache size -h)"

    # 오래된 파일 정리
    proxyndctl cache clear --older-than 30d -f
fi

# 캐시 히트율 확인
METRICS=$(proxyndctl metrics -c cache --format json)
HIT_RATE=$(echo $METRICS | jq -r '.cache.maven.hit_rate // 0')

if (( $(echo "$HIT_RATE < $CACHE_HIT_RATE_MIN" | bc -l) )); then
    echo "경고: Maven 캐시 히트율이 낮습니다: ${HIT_RATE}"
fi
```

### Maven 검색 활용 예제

```bash
#!/bin/bash
# Maven 아티팩트 검색 및 정보 수집

SEARCH_TERM="spring-boot"

echo "검색어: $SEARCH_TERM"
echo "=================="

# 인덱스가 없으면 빌드
if ! proxyndctl maven-index info >/dev/null 2>&1; then
    echo "인덱스 빌드 중..."
    proxyndctl maven-index build
fi

# 검색 수행
RESULTS=$(proxyndctl maven-index search "$SEARCH_TERM" --format json 2>/dev/null || echo '[]')

# 결과 처리
echo "$RESULTS" | jq -r '.[] | select(.type == "artifact") | "\(.group_id):\(.artifact_id)"' | sort | uniq | while read artifact; do
    echo "- $artifact"
done
```

### 자동화된 서버 상태 체크

```bash
#!/bin/bash
# ProxyND 서버 상태 자동 체크 및 리포트

REPORT_FILE="/var/log/proxynd/daily-report-$(date +%Y%m%d).log"

{
    echo "ProxyND 일일 상태 리포트"
    echo "생성 시각: $(date)"
    echo "========================"

    echo -e "\n## 서버 상태"
    proxyndctl status -d

    echo -e "\n## 헬스체크"
    proxyndctl health -d

    echo -e "\n## 캐시 통계"
    proxyndctl cache size -h

    echo -e "\n## 프록시 테스트 결과"
    proxyndctl test all -d

    echo -e "\n## Maven 인덱스 상태"
    proxyndctl maven-index info

    echo -e "\n## 최근 백업 정보"
    LATEST_BACKUP=$(ls -t /backup/maven-* 2>/dev/null | head -1)
    if [ -n "$LATEST_BACKUP" ]; then
        proxyndctl maven-backup info --backup "$LATEST_BACKUP"
    else
        echo "백업을 찾을 수 없습니다."
    fi

} > "$REPORT_FILE"

# 이메일로 전송 (선택사항)
# mail -s "ProxyND 일일 리포트 - $(date +%Y-%m-%d)" admin@example.com < "$REPORT_FILE"
```

## 문제 해결

### 일반적인 오류와 해결방법

#### 서버 연결 실패

```bash
# 문제 진단
proxyndctl status -v

# 가능한 해결책:
# 1. 서버 주소 확인
export PROXYNDCTL_SERVER="http://correct-server:8080"

# 2. 네트워크 연결 확인
curl -I $PROXYNDCTL_SERVER/healthz

# 3. 타임아웃 증가
proxyndctl status --timeout 60s
```

#### 인덱스 빌드 실패

```bash
# 메모리 부족 문제
# 해결: 시스템 리소스 확인
free -h
df -h /storage

# 권한 문제
# 해결: 저장 디렉토리 권한 확인
ls -la /storage/maven-index/

# 설정 문제
# 해결: Maven 설정 검증
proxyndctl config validate -f maven-proxy.yaml
```

#### 백업 실패

```bash
# 디스크 공간 부족
# 해결: 공간 확보
df -h /backup
find /backup -name "maven-*" -mtime +60 -exec rm -rf {} \;

# 소스 디렉토리 접근 권한
# 해결: 권한 확인
ls -la /storage/cache/maven/

# 체크섬 오류
# 해결: 체크섬 없이 백업
proxyndctl maven-backup create --target /backup/maven --no-checksum
```

### 디버깅 팁

```bash
# 상세 로그 활성화
export PROXYNDCTL_DEBUG=1
proxyndctl maven-index build -v

# HTTP 트래픽 디버깅
proxyndctl --server http://localhost:8080 status -v 2>&1 | tee debug.log

# 설정 파일 문제 확인
proxyndctl config show --merged | grep -i maven

# 프로세스 추적
strace -e network proxyndctl maven-index search spring 2>&1 | grep -E "(connect|send|recv)"
```

## 관련 링크

- [ProxyND 서버 설정 가이드](../../03-configuration/reference.md)
- [Maven 프록시 설정](../../04-proxy-types/maven/README.md)
- [API 참조 문서](../../09-api/README.md)
- [문제 해결 가이드](../../11-troubleshooting/README.md)

---

최종 업데이트: 2024-01-15
버전: ProxyND v2.0.0 / proxyndctl v1.0.0
