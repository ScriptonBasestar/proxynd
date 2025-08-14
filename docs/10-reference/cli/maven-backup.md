# Maven Backup 관리

ProxyND CLI의 Maven 백업 관리 기능을 사용하여 Maven 저장소 캐시의 증분 백업을 생성하고 복원할 수 있습니다.

## 개요

Maven 백업 기능은 ProxyND의 Maven 캐시 데이터를 안전하게 보관하고 필요시 복원할 수 있는 증분 백업 시스템입니다. 변경된 파일만 백업하여 저장 공간과 시간을 절약합니다.

### 주요 기능
- **증분 백업**: 변경된 파일만 백업하여 효율성 극대화
- **체크섬 검증**: SHA-256 체크섬으로 데이터 무결성 보장
- **병렬 처리**: 다중 워커로 백업/복원 속도 향상
- **메타데이터 관리**: 백업 정보 및 이력 추적

## 명령어 구조

```bash
proxyndctl maven-backup <subcommand> [flags]
```

### 하위 명령어
- `create` - 증분 백업 생성
- `restore` - 백업에서 복원
- `info` - 백업 정보 표시

## 명령어 상세

### create - 백업 생성

Maven 캐시의 증분 백업을 생성합니다.

```bash
proxyndctl maven-backup create [flags]
```

**플래그**:
- `-s, --source <dir>`: 소스 디렉토리 (Maven 캐시 위치)
- `-t, --target <dir>`: 백업 대상 디렉토리 (필수)
- `-c, --checksum`: 체크섬 기반 변경 감지 (기본: true)
- `-w, --workers <num>`: 병렬 워커 수 (기본: 4)

**사용 예시**:
```bash
# 기본 설정으로 백업 생성
proxyndctl maven-backup create --target /backup/maven

# 특정 소스와 8개 워커로 백업
proxyndctl maven-backup create \
  --source /storage/cache/maven \
  --target /backup/maven \
  --workers 8

# 체크섬 없이 빠른 백업 (수정시간 기반)
proxyndctl maven-backup create \
  --target /backup/maven \
  --checksum=false
```

**출력 예시**:
```
Starting incremental backup...
Source: /storage/cache/maven
Target: /backup/maven
Use checksum: true
Workers: 4

[1/2547] org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar
[2/2547] org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom
...
[2547/2547] Complete

Backup completed successfully!
Backup ID: backup-20250101-103045
Total files: 2,547
Total size: 1.23 GB
Duration: 45s
```

### restore - 백업 복원

백업에서 Maven 캐시를 복원합니다.

```bash
proxyndctl maven-backup restore [flags]
```

**플래그**:
- `-s, --source <dir>`: 백업 소스 디렉토리 (필수)
- `-t, --target <dir>`: 복원 대상 디렉토리 (필수)
- `-o, --overwrite`: 기존 파일 덮어쓰기
- `-v, --verify`: 복원 전 체크섬 검증 (기본: true)

**사용 예시**:
```bash
# 기본 복원
proxyndctl maven-backup restore \
  --source /backup/maven \
  --target /storage/cache/maven

# 기존 파일 덮어쓰기로 완전 복원
proxyndctl maven-backup restore \
  --source /backup/maven \
  --target /storage/cache/maven \
  --overwrite

# 체크섬 검증 없이 빠른 복원
proxyndctl maven-backup restore \
  --source /backup/maven \
  --target /storage/cache/maven \
  --verify=false
```

**출력 예시**:
```
Starting restore...
Backup source: /backup/maven
Restore target: /storage/cache/maven
Overwrite: false
Verify: true

[1/2547] org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar
[2/2547] org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom
...
[2547/2547] Complete

Restore completed successfully!
Duration: 32s
```

### info - 백업 정보

백업의 상태와 메타데이터를 표시합니다.

```bash
proxyndctl maven-backup info [flags]
```

**플래그**:
- `-b, --backup <dir>`: 백업 디렉토리 위치 (필수)

**사용 예시**:
```bash
# 백업 정보 표시
proxyndctl maven-backup info --backup /backup/maven
```

**출력 예시**:
```
Backup Information
==================
Backup ID: backup-20250101-103045
Last backup: 2025-01-01 10:30:45
Total files: 2,547
Total size: 1.23 GB
Files tracked: 2,547
Backup age: 2 hours
```

## 백업 구조

### 백업 디렉토리 구조

```
/backup/maven/
├── .backup-metadata.json    # 백업 메타데이터
├── .file-checksums.json     # 파일 체크섬 정보
└── data/                    # 실제 백업 데이터
    ├── org/
    │   └── springframework/
    │       └── spring-core/
    └── com/
        └── fasterxml/
            └── jackson/
```

### 메타데이터 파일

**`.backup-metadata.json`**:
```json
{
  "backup_id": "backup-20250101-103045",
  "created_at": "2025-01-01T10:30:45Z",
  "source_path": "/storage/cache/maven",
  "total_files": 2547,
  "total_size": 1320402944,
  "checksum_method": "sha256"
}
```

**`.file-checksums.json`**:
```json
{
  "org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar": {
    "size": 1234567,
    "modified": "2025-01-01T10:15:30Z",
    "checksum": "a1b2c3d4e5f6..."
  }
}
```

## 설정 및 환경

### 환경 변수

- `STORAGE_DIR`: 기본 Maven 캐시 위치 결정

### Maven 프록시 설정

백업할 Maven 캐시는 Maven 프록시 설정에 따라 결정됩니다:

```yaml
# maven-proxy.yaml
cache:
  type: "filesystem"
  path: "/storage/cache/maven"  # 이 경로가 백업 대상
  ttl: "24h"
```

## 백업 전략

### 권장 백업 일정

1. **일일 증분 백업**: 매일 새벽 시간대 실행
2. **주간 전체 검증**: 주 1회 체크섬 기반 백업
3. **월간 아카이브**: 월말 백업을 별도 보관

### 자동화 스크립트

**일일 백업 스크립트** (`/etc/cron.daily/maven-backup`):
```bash
#!/bin/bash
export STORAGE_DIR="/var/lib/proxynd"
export CONFIG_DIR="/etc/proxynd"

BACKUP_BASE="/backup/maven"
BACKUP_DATE=$(date +%Y%m%d)
BACKUP_DIR="${BACKUP_BASE}/${BACKUP_DATE}"

# 백업 디렉토리 생성
mkdir -p "${BACKUP_DIR}"

# 증분 백업 실행
proxyndctl maven-backup create \
  --target "${BACKUP_DIR}" \
  --workers 8 \
  2>&1 | logger -t maven-backup

# 7일 이상 된 백업 정리
find "${BACKUP_BASE}" -type d -name "202*" -mtime +7 -exec rm -rf {} \;
```

## 성능 최적화

### 백업 성능

**병렬 처리**:
- 워커 수는 CPU 코어 수의 2배 권장
- SSD 환경에서는 더 많은 워커 활용 가능
- 네트워크 백업 시에는 대역폭 고려

**체크섬 vs 시간 기반**:
- 체크섬: 정확하지만 CPU 사용량 높음
- 시간 기반: 빠르지만 시간 동기화 필요

### 저장 공간 최적화

**압축 고려사항**:
- Maven JAR 파일은 이미 압축되어 압축 효과 제한적
- 메타데이터 파일 (POM, XML)은 압축 효과 높음

**증분 백업 효율성**:
- 첫 백업: 전체 데이터 복사
- 이후 백업: 변경분만 복사 (보통 5-10%)

## 문제 해결

### 일반적인 문제들

**백업 실패**:
```bash
# 권한 확인
ls -la /storage/cache/maven
ls -la /backup/maven

# 디스크 공간 확인
df -h /backup

# 소스 디렉토리 확인
proxyndctl config show maven
```

**복원 실패**:
```bash
# 백업 무결성 확인
proxyndctl maven-backup info --backup /backup/maven

# 대상 디렉토리 권한 확인
mkdir -p /storage/cache/maven
chown proxynd:proxynd /storage/cache/maven
```

**성능 문제**:
```bash
# I/O 모니터링
iostat -x 1

# 워커 수 조정
proxyndctl maven-backup create --target /backup --workers 2

# 체크섬 비활성화로 속도 향상
proxyndctl maven-backup create --target /backup --checksum=false
```

## 고급 사용법

### 선택적 백업

특정 패키지만 백업하는 스크립트:
```bash
#!/bin/bash
# 스프링 관련 패키지만 백업

SOURCE="/storage/cache/maven"
TARGET="/backup/maven-spring"

# 스프링 관련 디렉토리만 복사
rsync -av \
  "${SOURCE}/org/springframework/" \
  "${SOURCE}/org/spring*/" \
  "${TARGET}/data/"
```

### 원격 백업

네트워크를 통한 원격 백업:
```bash
# SSH를 통한 원격 백업
proxyndctl maven-backup create --target /tmp/maven-backup
rsync -av --delete /tmp/maven-backup/ backup-server:/backup/maven/

# 클라우드 스토리지 동기화 (AWS S3 예시)
aws s3 sync /backup/maven/ s3://company-maven-backup/
```

### 백업 검증

백업 무결성 검증 스크립트:
```bash
#!/bin/bash
# 백업 무결성 검증

BACKUP_DIR="/backup/maven"

echo "Verifying backup integrity..."
proxyndctl maven-backup info --backup "${BACKUP_DIR}"

# 무작위 파일 체크섬 검증
find "${BACKUP_DIR}/data" -name "*.jar" | shuf -n 10 | while read file; do
  echo "Checking $file..."
  # 체크섬 검증 로직
done
```

## 관련 문서

- [Maven 프록시 설정](../../04-proxy-types/maven/README.md)
- [캐시 관리](./cache.md)
- [CLI 전체 참조](./PROXYNDCTL_REFERENCE.md)
- [운영 가이드](../../06-deployment/README.md)
