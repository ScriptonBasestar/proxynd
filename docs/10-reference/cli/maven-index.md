# Maven Index 관리

ProxyND CLI의 Maven 인덱스 관리 기능을 사용하여 Maven 저장소의 검색 인덱스를 생성, 관리, 검색할 수 있습니다.

## 개요

Maven 인덱스는 대용량 Maven 저장소에서 빠른 패키지 검색을 위한 기능입니다. 모든 아티팩트의 메타데이터를 미리 수집하여 JSON 형태의 검색 인덱스를 생성합니다.

### 주요 기능
- **인덱스 빌드**: Maven 저장소 전체 스캔하여 검색 인덱스 생성
- **인덱스 정보**: 현재 인덱스 상태 및 통계 확인
- **인덱스 검색**: 생성된 인덱스에서 빠른 패키지 검색

## 명령어 구조

```bash
proxyndctl maven-index <subcommand> [flags]
```

### 하위 명령어
- `build` - 인덱스 빌드 또는 리빌드
- `info` - 인덱스 정보 표시
- `search` - 인덱스에서 검색

## 명령어 상세

### build - 인덱스 빌드

Maven 저장소를 스캔하여 검색 인덱스를 생성합니다.

```bash
proxyndctl maven-index build [flags]
```

**플래그**:
- `-s, --storage <dir>`: 인덱스 파일 저장 디렉토리
- `-f, --force`: 기존 인덱스가 있어도 강제로 리빌드
- `-v, --verbose`: 상세한 진행 상황 표시

**사용 예시**:
```bash
# 기본 설정으로 인덱스 빌드
proxyndctl maven-index build

# 특정 디렉토리에 인덱스 저장
proxyndctl maven-index build --storage /var/lib/proxynd/index

# 기존 인덱스 강제 리빌드
proxyndctl maven-index build --force --verbose
```

**출력 예시**:
```
Building Maven search index...
Storage directory: /storage/maven-index
Configured mirrors: 3

[1/245] Indexing org.springframework...
[2/245] Indexing com.fasterxml.jackson... (1250 entries)
...

Index build completed!
Total entries: 12,547
Time taken: 2m 15s
Index saved to: /storage/maven-index/maven-search-index.json
Index file size: 5.24 MB
```

### info - 인덱스 정보

현재 인덱스의 상태와 통계를 표시합니다.

```bash
proxyndctl maven-index info [flags]
```

**플래그**:
- `-s, --storage <dir>`: 인덱스 파일 위치 지정
- `-v, --verbose`: 샘플 엔트리 표시

**사용 예시**:
```bash
# 기본 인덱스 정보 표시
proxyndctl maven-index info

# 샘플 엔트리 포함하여 상세 정보 표시
proxyndctl maven-index info --verbose
```

**출력 예시**:
```
Index file: /storage/maven-index/maven-search-index.json
Size: 5.24 MB
Last modified: 2025-01-01 10:30:15 (2 hours ago)
Total entries: 12,547

Entry types:
  group: 245
  artifact: 3,892
  version: 8,410

Sample entries:
  [group] org.springframework - /org/springframework
  [artifact] spring-core - /org/springframework/spring-core
  [version] 5.3.21 - /org/springframework/spring-core/5.3.21
```

### search - 인덱스 검색

생성된 인덱스에서 패키지를 검색합니다.

```bash
proxyndctl maven-index search <query> [flags]
```

**플래그**:
- `-s, --storage <dir>`: 인덱스 파일 위치 지정
- `-v, --verbose`: 모든 매치 결과 표시 (기본: 상위 20개)

**사용 예시**:
```bash
# 스프링 관련 패키지 검색
proxyndctl maven-index search spring

# Jackson 라이브러리 검색 (모든 결과 표시)
proxyndctl maven-index search jackson --verbose

# 특정 그룹 ID로 검색
proxyndctl maven-index search com.fasterxml
```

**출력 예시**:
```
Searching for: spring

[group] /org/springframework
  Group: org.springframework

[artifact] /org/springframework/spring-core
  Group: org.springframework
  Artifact: spring-core

[artifact] /org/springframework/spring-boot
  Group: org.springframework
  Artifact: spring-boot

[version] /org/springframework/spring-core/5.3.21
  Group: org.springframework
  Artifact: spring-core

Found 847 matches
(showing first 20, use --verbose to see all)
```

## 설정

### Maven 프록시 설정

인덱스 기능을 사용하려면 Maven 프록시가 설정되어 있어야 합니다:

```yaml
# maven-proxy.yaml
proxies:
  - name: "central"
    upstream_url: "https://repo1.maven.org/maven2/"
    enabled: true
  - name: "spring"
    upstream_url: "https://repo.spring.io/release/"
    enabled: true

# 인덱스 설정 (선택사항)
search_index:
  storage_path: "/var/lib/proxynd/maven-index"
  auto_build: false
  build_interval: "24h"
```

### 환경 변수

- `STORAGE_DIR`: 기본 저장 디렉토리 (인덱스 저장 위치 결정)

## 성능 및 최적화

### 인덱스 빌드 최적화

**빌드 시간**: 저장소 크기에 따라 수분에서 수십분 소요
- 소규모 (< 1,000 패키지): 1-2분
- 중규모 (1,000-10,000 패키지): 5-15분  
- 대규모 (> 10,000 패키지): 15-60분

**메모리 사용량**: 약 50,000개 엔트리당 약 100MB RAM

### 인덱스 관리 권장사항

1. **정기적 갱신**: 새로운 패키지가 자주 추가되는 경우 일일 갱신 권장
2. **저장 공간**: 인덱스 파일 크기는 대략 엔트리당 500바이트
3. **성능**: 검색 성능은 인덱스 크기와 무관하게 일정 (O(n) 선형 검색)

## 문제 해결

### 일반적인 문제들

**인덱스 빌드 실패**:
```bash
# 설정 확인
proxyndctl config show maven

# 저장소 연결 테스트
proxyndctl test --proxy maven

# 권한 확인
ls -la /storage/maven-index/
```

**인덱스 파일 찾을 수 없음**:
```bash
# 인덱스 위치 확인
proxyndctl maven-index info

# 수동으로 경로 지정
proxyndctl maven-index info --storage /custom/path
```

**검색 결과 없음**:
```bash
# 인덱스 상태 확인
proxyndctl maven-index info --verbose

# 인덱스 리빌드
proxyndctl maven-index build --force
```

## 고급 사용법

### 자동화 스크립트

인덱스 자동 갱신을 위한 cron 스크립트:

```bash
#!/bin/bash
# /etc/cron.daily/proxynd-index-update

export STORAGE_DIR="/var/lib/proxynd"
export CONFIG_DIR="/etc/proxynd"

# 24시간 이상 오래된 인덱스만 리빌드
proxyndctl maven-index build 2>&1 | logger -t proxynd-index
```

### 모니터링

인덱스 상태 모니터링:

```bash
# 인덱스 나이 확인
proxyndctl maven-index info | grep "ago"

# 인덱스 크기 모니터링
du -h /storage/maven-index/maven-search-index.json
```

## 관련 문서

- [Maven 프록시 설정](../proxy-types/maven/README.md)
- [CLI 전체 참조](./PROXYNDCTL_REFERENCE.md)
- [성능 최적화 가이드](../performance-optimization.md)