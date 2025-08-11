# Maven Proxy E2E Testing Guide

Maven 프록시를 위한 상세한 End-to-End 테스트 가이드입니다.

## 🎯 테스트 목표

- Maven 저장소 프록시 기능 검증
- 아티팩트 다운로드 및 캐시 동작 확인
- Maven 클라이언트 호환성 검증
- POM, JAR, checksum 파일 처리 확인

## 📋 전제조건

```bash
# E2E 환경 구성 확인
cd tests/e2e
make status

# Maven 테스트 스크립트 실행 권한 확인
ls -la scripts/test-maven.sh
```

## 🚀 빠른 시작

### 1. Maven E2E 테스트 실행
```bash
cd tests/e2e
make test-maven
```

### 2. 수동 테스트
```bash
# 환경 시작
make up

# Maven 클라이언트 컨테이너 접속
make shell-maven

# Maven 설정 확인
cat ~/.m2/settings.xml

# 테스트 프로젝트에서 의존성 다운로드
cd /workspace/test-project
mvn dependency:resolve
```

## 📦 테스트 시나리오

### 1. 기본 아티팩트 다운로드
```bash
# junit-4.13.2 다운로드 테스트
curl -v "http://proxynd:8080/api/v1/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar"

# 기대 결과:
# - HTTP 200 응답
# - Content-Type: application/java-archive
# - 올바른 JAR 파일 다운로드
```

### 2. POM 파일 접근
```bash
# POM 파일 다운로드
curl -v "http://proxynd:8080/api/v1/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom"

# 기대 결과:
# - HTTP 200 응답
# - Content-Type: application/xml
# - 유효한 XML POM 파일
```

### 3. 체크섬 검증
```bash
# SHA1 체크섬 파일
curl -v "http://proxynd:8080/api/v1/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar.sha1"

# MD5 체크섬 파일
curl -v "http://proxynd:8080/api/v1/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar.md5"

# 기대 결과:
# - HTTP 200 응답
# - Content-Type: text/plain
# - 올바른 체크섬 값
```

### 4. 메타데이터 접근
```bash
# maven-metadata.xml
curl -v "http://proxynd:8080/api/v1/proxy/maven/junit/junit/maven-metadata.xml"

# 기대 결과:
# - HTTP 200 응답
# - Content-Type: application/xml
# - 버전 목록 포함
```

### 5. 디렉토리 브라우징 (활성화된 경우)
```bash
# 디렉토리 목록
curl -v "http://proxynd:8080/api/v1/proxy/maven/junit/junit/"

# 기대 결과:
# - HTTP 200 응답 또는 403 (설정에 따라)
# - HTML 디렉토리 목록 또는 금지 메시지
```

### 6. SNAPSHOT 아티팩트
```bash
# SNAPSHOT 버전 접근 (upstream에 있는 경우)
curl -v "http://proxynd:8080/api/v1/proxy/maven/com/example/test-artifact/1.0-SNAPSHOT/maven-metadata.xml"

# 기대 결과:
# - HTTP 200 응답
# - 타임스탬프 메타데이터 포함
```

### 7. 캐시 동작 확인
```bash
# 첫 번째 요청 (upstream에서 가져옴)
time curl -s "http://proxynd:8080/api/v1/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar" > /dev/null

# 두 번째 요청 (캐시에서 가져옴)
time curl -s "http://proxynd:8080/api/v1/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar" > /dev/null

# 기대 결과:
# - 두 번째 요청이 현저히 빠름
# - 동일한 응답 내용
```

## 🔍 실제 Maven 클라이언트 테스트

### 1. Maven 프로젝트 설정
```xml
<!-- test-project/pom.xml -->
<project>
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>test-project</artifactId>
    <version>1.0.0</version>
    
    <dependencies>
        <dependency>
            <groupId>junit</groupId>
            <artifactId>junit</artifactId>
            <version>4.13.2</version>
            <scope>test</scope>
        </dependency>
        <dependency>
            <groupId>org.apache.commons</groupId>
            <artifactId>commons-lang3</artifactId>
            <version>3.12.0</version>
        </dependency>
    </dependencies>
</project>
```

### 2. Maven 설정
```xml
<!-- ~/.m2/settings.xml -->
<settings>
    <mirrors>
        <mirror>
            <id>proxynd</id>
            <mirrorOf>central</mirrorOf>
            <url>http://proxynd:8080/api/v1/proxy/maven</url>
        </mirror>
    </mirrors>
</settings>
```

### 3. 의존성 해결 테스트
```bash
cd /workspace/test-project

# 의존성 다운로드
mvn dependency:resolve

# 컴파일 테스트
mvn compile

# 테스트 실행
mvn test

# 기대 결과:
# - 모든 의존성이 ProxyND를 통해 다운로드됨
# - 성공적인 컴파일 및 테스트 실행
```

## 📊 성능 테스트

### 1. 다중 동시 요청
```bash
# 10개 동시 요청
seq 1 10 | xargs -n1 -P10 bash -c 'curl -s "http://proxynd:8080/api/v1/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar" > /dev/null'

# 기대 결과:
# - 모든 요청이 성공적으로 처리됨
# - 응답 시간이 합리적 범위 내
```

### 2. 대용량 파일 다운로드
```bash
# 큰 JAR 파일 다운로드 (예: Spring Framework)
time curl -s "http://proxynd:8080/api/v1/proxy/maven/org/springframework/spring-core/5.3.10/spring-core-5.3.10.jar" > /dev/null

# 기대 결과:
# - 안정적인 다운로드
# - 타임아웃 없음
```

## 🔧 문제 해결

### 1. 아티팩트를 찾을 수 없음 (404)
```bash
# ProxyND 로그 확인
make logs-proxynd | grep maven

# upstream 연결 확인
curl -v http://nginx-upstream:8081/maven/junit/junit/4.13.2/junit-4.13.2.jar

# 해결책:
# - upstream 데이터 확인
# - 네트워크 연결 확인
# - 프록시 설정 검토
```

### 2. 체크섬 불일치
```bash
# 로컬 체크섬 계산
sha1sum /path/to/downloaded/file.jar

# ProxyND에서 제공하는 체크섬과 비교
curl -s "http://proxynd:8080/api/v1/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar.sha1"

# 해결책:
# - upstream 데이터 무결성 확인
# - 캐시 클리어 후 재시도
```

### 3. Maven 클라이언트 오류
```bash
# Maven 디버그 모드
mvn dependency:resolve -X

# 설정 확인
mvn help:effective-settings

# 해결책:
# - settings.xml 구성 검토
# - 네트워크 연결 확인
# - 프록시 URL 검증
```

## 📈 모니터링

### 1. 메트릭 확인 (구현된 경우)
```bash
# Maven 프록시 메트릭
curl http://proxynd:8080/metrics | grep maven

# 기대 메트릭:
# - maven_requests_total
# - maven_cache_hits_total
# - maven_response_time_seconds
```

### 2. 로그 분석
```bash
# Maven 관련 로그 필터링
make logs-proxynd | grep -i maven

# 액세스 패턴 분석
grep "GET.*maven" /storage/access.log | head -10
```

## ✅ 성공 기준

- [ ] 모든 기본 아티팩트 다운로드 성공
- [ ] POM 파일 올바른 제공
- [ ] 체크섬 파일 정확한 제공
- [ ] 메타데이터 XML 유효성 검증
- [ ] Maven 클라이언트 완전 호환성
- [ ] 캐시 기능 정상 동작
- [ ] 동시 요청 안정적 처리
- [ ] 에러 상황 적절한 처리

## 🔄 자동화

이 테스트들은 `tests/e2e/scripts/test-maven.sh`에 자동화되어 있으며, CI/CD 파이프라인에서 자동 실행됩니다.

```bash
# 전체 자동화 테스트 실행
cd tests/e2e
./scripts/test-maven.sh

# 기대 출력:
# ✅ Maven proxy basic functionality
# ✅ Maven client compatibility
# ✅ Cache behavior
# ✅ Performance requirements
```