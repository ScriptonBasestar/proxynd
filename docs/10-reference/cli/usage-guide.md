# ProxyND CLI 사용 가이드

이 문서는 `proxyndctl` 명령어의 전체적인 사용법을 설명합니다.

## 목차

- [설치](#설치)
- [기본 사용법](#기본-사용법)
- [주요 명령어](#주요-명령어)
- [실제 사용 예제](#실제-사용-예제)
- [고급 기능](#고급-기능)
- [문제 해결](#문제-해결)

## 설치

### 바이너리 다운로드

```bash
# 최신 릴리스 다운로드
curl -L https://github.com/scriptonbasestar/proxynd/releases/latest/download/proxyndctl-linux-amd64 -o proxyndctl
chmod +x proxyndctl
sudo mv proxyndctl /usr/local/bin/
```

### 소스에서 빌드

```bash
# Go 1.24 이상 필요
go build -o proxyndctl ./cmd/proxyndctl
sudo mv proxyndctl /usr/local/bin/
```

### 자동완성 설치

```bash
# 자동 설치 스크립트 사용
./scripts/install-completion.sh

# 또는 수동 설치
source <(proxyndctl completion bash)  # Bash
source <(proxyndctl completion zsh)   # Zsh
```

### Man Page 설치

```bash
# 자동 설치 스크립트 사용
sudo ./scripts/install-man-pages.sh

# 또는 수동 설치
proxyndctl docs man ./man
sudo cp ./man/*.1 /usr/share/man/man1/
sudo mandb
```

## 기본 사용법

### 도움말 확인

```bash
# 전체 도움말
proxyndctl --help

# 특정 명령어 도움말
proxyndctl cache --help
proxyndctl cache list --help

# Man page 보기
man proxyndctl
man proxyndctl-cache-list
```

### 전역 옵션

모든 명령어에서 사용 가능한 옵션:

```bash
# 다른 서버에 연결
proxyndctl --server http://proxy.example.com:8080 status

# 출력 형식 변경
proxyndctl cache list --format json
proxyndctl cache list --format yaml

# 상세 출력
proxyndctl -v status

# 타임아웃 설정
proxyndctl --timeout 60s test all
```

## 주요 명령어

### 서버 상태 관리

#### 상태 확인

```bash
# 서버 전반적인 상태
proxyndctl status

# 상세 정보 포함
proxyndctl status -d

# JSON 형식으로 출력
proxyndctl status --format json
```

#### 헬스체크

```bash
# 기본 헬스체크
proxyndctl health

# 의존성 상태 포함
proxyndctl health -d
```

#### 메트릭 조회

```bash
# 전체 메트릭
proxyndctl metrics

# 특정 카테고리만
proxyndctl metrics -c cache
proxyndctl metrics -c system
proxyndctl metrics -c proxy

# 상세 메트릭
proxyndctl metrics -d
```

### 캐시 관리

#### 캐시 목록 조회

```bash
# 전체 캐시 목록
proxyndctl cache list

# 특정 프록시 타입만
proxyndctl cache list -t apt
proxyndctl cache list -t npm

# 패턴으로 필터링
proxyndctl cache list -p "*.json"
proxyndctl cache list -p "express-*"

# 정렬 옵션
proxyndctl cache list --sort size
proxyndctl cache list --sort time
```

#### 캐시 크기 확인

```bash
# 전체 캐시 크기
proxyndctl cache size

# 프록시별 크기
proxyndctl cache size -t apt
proxyndctl cache size -t npm

# 사람이 읽기 쉬운 형식
proxyndctl cache size -h
```

#### 캐시 삭제

```bash
# 전체 캐시 삭제 (확인 필요)
proxyndctl cache clear

# 강제 삭제 (확인 없이)
proxyndctl cache clear -f

# 특정 프록시 타입만 삭제
proxyndctl cache clear -t apt
proxyndctl cache clear -t npm -f

# 패턴으로 삭제
proxyndctl cache clear -p "*.tmp"
proxyndctl cache clear -p "old-*" -f

# 시간 기반 캐시 정리
proxyndctl cache clear --older-than 7d    # 7일보다 오래된 캐시 정리
proxyndctl cache clear --older-than 2h    # 2시간보다 오래된 캐시 정리
proxyndctl cache clear --older-than 30m   # 30분보다 오래된 캐시 정리

# 크기 기반 캐시 정리
proxyndctl cache clear --size-limit 1GB   # 총 캐시 크기를 1GB로 제한
proxyndctl cache clear --size-limit 500MB # 총 캐시 크기를 500MB로 제한

# 고급 조합 사용
proxyndctl cache clear -t apt --older-than 30d -f
proxyndctl cache clear -t npm --size-limit 2GB -f
```

### 설정 관리

#### 설정 검증

```bash
# 모든 설정 파일 검증
proxyndctl config validate

# 특정 파일만 검증
proxyndctl config validate -f apt-proxy.yaml
proxyndctl config validate -f global.yaml

# 상세 검증 결과
proxyndctl config validate -v
```

#### 설정 조회

```bash
# 모든 설정 파일 목록
proxyndctl config show

# 특정 설정 파일 내용
proxyndctl config show global.yaml
proxyndctl config show apt-proxy.yaml

# 병합된 설정 보기
proxyndctl config show --merged
```

### 사용자 관리

#### 사용자 목록

```bash
# 기본 목록
proxyndctl user list

# 상세 정보 포함
proxyndctl user list -d

# JSON 형식
proxyndctl user list --format json
```

#### 사용자 추가

```bash
# 명령줄 옵션으로 추가
proxyndctl user add -u john -p secret123 -r admin -d "관리자"

# 대화형 모드 (안전한 비밀번호 입력)
proxyndctl user add -i
# Enter username: developer
# Enter password: [입력 내용 숨김]
# Confirm password: [입력 내용 숨김]
# Enter role (user/admin): user
# Enter description: Development team member

# 대화형 모드와 일부 옵션 조합
proxyndctl user add -i -r admin
proxyndctl user add -i -u newuser

# 사용자 설명 추가
proxyndctl user add -u jane -p pass456 -r user -d "QA 엔지니어"
```

#### 사용자 삭제

```bash
# 확인 후 삭제
proxyndctl user delete john

# 강제 삭제
proxyndctl user delete john -f
```

#### 사용자 정보 조회

```bash
# 특정 사용자 상세 정보
proxyndctl user info john

# JSON 형식으로 출력
proxyndctl user info john --format json

# 여러 사용자 정보 확인
proxyndctl user info admin
proxyndctl user info developer
```

### 프록시 테스트

#### 전체 테스트

```bash
# 모든 프록시 테스트
proxyndctl test all

# 상세 결과 표시
proxyndctl test all -d

# 타임아웃 설정
proxyndctl test all -t 60
```

#### 개별 프록시 테스트

```bash
# APT 프록시 테스트
proxyndctl test apt

# NPM 프록시 테스트 (특정 경로)
proxyndctl test npm -u /proxy/npm/express

# Docker 프록시 테스트
proxyndctl test docker -t 30
```

#### 연결성 테스트

```bash
# 특정 프록시 연결성만 확인
proxyndctl test connectivity apt
proxyndctl test connectivity npm
```

#### 지원 프록시 타입 확인

```bash
proxyndctl test types
```

### 문서 생성

#### Man Page 생성

```bash
# 현재 디렉토리에 생성
proxyndctl docs man .

# 특정 디렉토리에 생성
proxyndctl docs man /tmp/man
```

#### Markdown 문서 생성

```bash
# CLI 참조 문서 생성
proxyndctl docs markdown ./docs/cli

# GitHub Wiki용
proxyndctl docs markdown ./wiki
```

#### 기타 형식

```bash
# RestructuredText (Sphinx용)
proxyndctl docs rest ./docs/source

# YAML (자동화 도구용)
proxyndctl docs yaml .
```

## 실제 사용 예제

### 시나리오 1: 캐시 관리 자동화

```bash
#!/bin/bash
# 고급 캐시 정리 스크립트

# 캐시 크기 확인
SIZE=$(proxyndctl cache size --format json | jq -r '.total_size')

# 10GB 이상이면 정리
if [ $SIZE -gt 10737418240 ]; then
    echo "캐시 크기가 10GB를 초과했습니다. 정리를 시작합니다..."

    # 1단계: 시간 기반 정리 (30일 이상)
    echo "1단계: 30일 이상된 오래된 캐시 정리"
    proxyndctl cache clear --older-than 30d -f

    # 2단계: 크기 제한 적용 (5GB로 제한)
    echo "2단계: 캐시 크기를 5GB로 제한"
    proxyndctl cache clear --size-limit 5GB -f

    # 3단계: 프록시별 개별 정리
    echo "3단계: 프록시별 세부 정리"
    # NPM 캐시는 7일 이상만 정리
    proxyndctl cache clear -t npm --older-than 7d -f
    
    # APT 캐시는 2GB로 제한
    proxyndctl cache clear -t apt --size-limit 2GB -f
    
    # Maven 캐시는 14일 이상만 정리 (개발 환경을 위해 길게 설정)
    proxyndctl cache clear -t maven --older-than 14d -f

    echo "캐시 정리 완료"
    proxyndctl cache size -h
fi
```

### 시나리오 2: 모니터링 통합

```bash
#!/bin/bash
# Prometheus 메트릭 수집 스크립트

# 메트릭을 JSON으로 가져와서 변환
proxyndctl metrics --format json | jq -r '
    .cache | to_entries[] |
    "proxynd_cache_hit_rate{type=\"\(.key)\"} \(.value.hit_rate)"
'

# 헬스 상태 확인
STATUS=$(proxyndctl health --format json | jq -r '.status')
if [ "$STATUS" != "healthy" ]; then
    # 알림 전송
    curl -X POST $SLACK_WEBHOOK -d "{\"text\": \"ProxyND 상태 이상: $STATUS\"}"
fi
```

### 시나리오 3: 사용자 관리 자동화

```bash
#!/bin/bash
# 고급 사용자 관리 스크립트

# CSV 파일에서 사용자 일괄 추가
echo "=== 사용자 일괄 추가 ==="
while IFS=, read -r username password role description
do
    echo "사용자 추가: $username ($role)"
    proxyndctl user add -u "$username" -p "$password" -r "$role" -d "$description"
done < users.csv

# 기존 사용자 정보 확인 및 업데이트
echo "=== 기존 사용자 정보 확인 ==="
USERS=$(proxyndctl user list --format json | jq -r '.users[].username')
for user in $USERS; do
    echo "사용자 정보: $user"
    proxyndctl user info "$user"
    echo "---"
done

# 대화형 모드로 관리자 계정 생성
echo "=== 관리자 계정 생성 (대화형 모드) ==="
read -p "새 관리자 계정을 생성하시겠습니까? (y/n): " confirm
if [ "$confirm" = "y" ]; then
    echo "대화형 모드로 관리자 계정을 생성합니다..."
    proxyndctl user add -i -r admin
fi

# 비활성 사용자 정리 (예시)
echo "=== 사용자 계정 정리 ==="
# 실제 환경에서는 로그인 기록 등을 확인하여 결정
# proxyndctl user delete inactive_user -f
```

### 시나리오 4: 일일 리포트 생성

```bash
#!/bin/bash
# 일일 사용 리포트 생성 (고급 기능 포함)

DATE=$(date +%Y-%m-%d)
REPORT_FILE="proxynd-report-$DATE.md"

cat > $REPORT_FILE << EOF
# ProxyND 일일 리포트 - $DATE

## 서버 상태
$(proxyndctl status -d)

## 헬스체크 결과
$(proxyndctl health -d)

## 캐시 통계 (상세)
$(proxyndctl cache size -h)

### 프록시 타입별 캐시 크기
$(proxyndctl cache size --format json | jq -r '
.size_by_type | to_entries[] | 
"- \(.key): \(.value | . / 1024 / 1024 | floor)MB"
')

## 프록시별 상태 테스트
$(proxyndctl test all -d)

## 상위 캐시 항목 (크기순)
$(proxyndctl cache list --sort size --limit 10)

## 사용자 계정 현황
활성 사용자 수: $(proxyndctl user list --format json | jq '.users | length')

### 사용자 목록
$(proxyndctl user list)

## 메트릭 요약
### 시스템 메트릭
$(proxyndctl metrics -c system)

### 캐시 메트릭
$(proxyndctl metrics -c cache)

### 프록시 메트릭
$(proxyndctl metrics -c proxy)

## 권장 사항
EOF

# 캐시 크기가 큰 경우 권장사항 추가
CACHE_SIZE=$(proxyndctl cache size --format json | jq -r '.total_size')
if [ $CACHE_SIZE -gt 5368709120 ]; then  # 5GB 이상
    cat >> $REPORT_FILE << EOF

⚠️ **캐시 크기 경고**: 캐시 크기가 5GB를 초과했습니다.
다음 명령어로 정리를 고려하세요:
\`\`\`bash
# 30일 이상된 캐시 정리
proxyndctl cache clear --older-than 30d -f

# 또는 크기 제한
proxyndctl cache clear --size-limit 3GB -f
\`\`\`
EOF
fi

cat >> $REPORT_FILE << EOF

---
생성 시각: $(date)
생성 도구: ProxyND CLI (proxyndctl)
EOF

# 이메일로 전송
mail -s "ProxyND 일일 리포트 - $DATE" admin@example.com < $REPORT_FILE

echo "리포트가 생성되었습니다: $REPORT_FILE"
```

## 고급 기능

### 출력 파싱

```bash
# jq를 사용한 JSON 파싱
proxyndctl cache list --format json | jq '.items[] | select(.size > 1000000)'

# 특정 필드만 추출
proxyndctl user list --format json | jq -r '.users[].username'

# YAML 파싱 (yq 사용)
proxyndctl status --format yaml | yq '.server.uptime'
```

### 스크립트에서 사용

```bash
#!/bin/bash
# 에러 처리 예제

if ! proxyndctl health > /dev/null 2>&1; then
    echo "서버가 응답하지 않습니다"
    exit 1
fi

# 종료 코드 확인
proxyndctl config validate
if [ $? -ne 0 ]; then
    echo "설정 파일에 오류가 있습니다"
    exit 1
fi
```

### 환경 변수 활용

```bash
# 기본 서버 설정
export PROXYNDCTL_SERVER="http://proxy.internal:8080"
export PROXYNDCTL_FORMAT="json"

# 이제 --server와 --format 옵션 없이 사용 가능
proxyndctl status
proxyndctl cache list
```

## 문제 해결

### 일반적인 문제

#### 서버 연결 실패

```bash
# 연결 테스트
curl -I http://localhost:8080/healthz

# 상세 오류 확인
proxyndctl -v status

# 다른 포트 시도
proxyndctl --server http://localhost:8090 status
```

#### 권한 문제

```bash
# 사용자 권한 확인
proxyndctl user info $(whoami)

# sudo로 실행 (필요한 경우)
sudo proxyndctl user add -u admin -p secret -r admin
```

#### 타임아웃 문제

```bash
# 타임아웃 증가
proxyndctl --timeout 300s test all

# 개별 테스트로 분리
proxyndctl test apt
proxyndctl test npm
```

### 디버깅

```bash
# 상세 로그 활성화
proxyndctl -v cache list

# HTTP 요청 디버깅
PROXYNDCTL_DEBUG=1 proxyndctl status

# strace로 시스템 호출 추적
strace -e network proxyndctl health
```

### 로그 확인

```bash
# 서버 로그 위치
tail -f /var/log/proxynd/proxynd.log

# systemd 로그
journalctl -u proxynd -f

# Docker 로그
docker logs -f proxynd
```

## 관련 문서

- [CLI 자동완성 가이드](CLI_COMPLETION.md)
- [ProxyND 서버 설정 가이드](../README.md)
- [API 참조 문서](API_REFERENCE.md)
