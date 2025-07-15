# Maven 프록시 설정 가이드

## 클라이언트 설정

### 1. settings.xml 수정

#### ~/.m2/settings.xml (사용자별 설정)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<settings xmlns="http://maven.apache.org/SETTINGS/1.0.0"
          xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
          xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.0.0
                              http://maven.apache.org/xsd/settings-1.0.0.xsd">
  
  <mirrors>
    <!-- ProxyND를 모든 저장소의 미러로 설정 -->
    <mirror>
      <id>proxynd</id>
      <mirrorOf>*</mirrorOf>
      <name>ProxyND Central Mirror</name>
      <url>http://your-proxy-server:8080/proxy/maven/</url>
    </mirror>
  </mirrors>
  
  <!-- 프록시 서버 인증이 필요한 경우 -->
  <servers>
    <server>
      <id>proxynd</id>
      <username>your-username</username>
      <password>your-password</password>
    </server>
  </servers>
  
</settings>
```

#### /etc/maven/settings.xml (시스템 전체 설정)

시스템 전체에 적용하려면 Maven 설치 디렉토리의 conf/settings.xml을 수정하거나 /etc/maven/settings.xml을 생성합니다.

### 2. 프로젝트별 설정 (pom.xml)

```xml
<project>
  <!-- ... -->
  
  <repositories>
    <repository>
      <id>proxynd-central</id>
      <url>http://your-proxy-server:8080/proxy/maven/</url>
      <releases>
        <enabled>true</enabled>
      </releases>
      <snapshots>
        <enabled>true</enabled>
      </snapshots>
    </repository>
  </repositories>
  
  <pluginRepositories>
    <pluginRepository>
      <id>proxynd-plugins</id>
      <url>http://your-proxy-server:8080/proxy/maven/</url>
      <releases>
        <enabled>true</enabled>
      </releases>
      <snapshots>
        <enabled>true</enabled>
      </snapshots>
    </pluginRepository>
  </pluginRepositories>
  
</project>
```

## 서버 설정 (ProxyND)

### maven-proxy.yaml 설정 예시

```yaml
path: proxy/maven
use_cache: true

proxies:
  - name: central
    url: https://repo1.maven.org/maven2
  - name: jcenter
    url: https://jcenter.bintray.com
  - name: google
    url: https://maven.google.com
  - name: spring-release
    url: https://repo.spring.io/release
  - name: spring-snapshot
    url: https://repo.spring.io/snapshot
    # 스냅샷 저장소는 캐시 TTL을 짧게 설정
    cache_ttl: 300  # 5분
```

## 사용 예시

### 의존성 다운로드
```bash
# 프로젝트 의존성 다운로드
mvn dependency:resolve

# 특정 아티팩트 다운로드
mvn dependency:get -Dartifact=org.springframework:spring-core:5.3.20

# 소스 코드와 함께 다운로드
mvn dependency:sources
```

### 빌드 및 배포
```bash
# 프로젝트 빌드
mvn clean install

# 스냅샷 배포 (배포 기능이 활성화된 경우)
mvn deploy
```

### 로컬 저장소 설정
```bash
# 커스텀 로컬 저장소 위치 지정
mvn -Dmaven.repo.local=/custom/path/.m2/repository clean install
```

## Gradle에서 Maven 프록시 사용

### build.gradle
```groovy
repositories {
    maven {
        url 'http://your-proxy-server:8080/proxy/maven/'
        // 인증이 필요한 경우
        credentials {
            username 'your-username'
            password 'your-password'
        }
    }
}
```

### gradle.properties
```properties
# 전역 설정
systemProp.http.proxyHost=your-proxy-server
systemProp.http.proxyPort=8080
systemProp.https.proxyHost=your-proxy-server
systemProp.https.proxyPort=8080
```

## 캐시 동작

- 아티팩트(JAR, POM 등)는 첫 요청 시 업스트림에서 다운로드되어 캐시
- 메타데이터(maven-metadata.xml)는 TTL에 따라 주기적으로 갱신
- SNAPSHOT 버전은 짧은 TTL로 자주 갱신
- SHA1/MD5 체크섬 파일도 함께 캐시

## 문제 해결

### SSL 인증서 오류
```bash
# Maven에서 SSL 검증 비활성화 (권장하지 않음)
mvn -Dmaven.wagon.http.ssl.insecure=true -Dmaven.wagon.http.ssl.allowall=true clean install
```

### 캐시 클리어
```bash
# 로컬 Maven 캐시 클리어
rm -rf ~/.m2/repository/*

# 서버 캐시 클리어 (ProxyND 서버에서)
rm -rf /storage/proxy/maven/*
```

### 디버그 모드
```bash
# 상세 로그 출력
mvn -X clean install

# 네트워크 디버그
mvn -Dorg.slf4j.simpleLogger.defaultLogLevel=debug clean install
```

### 프록시 건너뛰기
```bash
# 특정 호스트에 대해 프록시 건너뛰기
mvn -Dhttp.nonProxyHosts="localhost|127.0.0.1|*.company.com" clean install
```

## CI/CD 환경

### Jenkins
```groovy
pipeline {
    agent any
    
    tools {
        maven 'Maven-3.8.6'
    }
    
    stages {
        stage('Build') {
            steps {
                configFileProvider([configFile(fileId: 'maven-settings', variable: 'MAVEN_SETTINGS')]) {
                    sh 'mvn -s $MAVEN_SETTINGS clean install'
                }
            }
        }
    }
}
```

### GitHub Actions
```yaml
- name: Set up Maven
  uses: actions/setup-java@v3
  with:
    java-version: '11'
    distribution: 'temurin'
    
- name: Configure Maven settings
  run: |
    mkdir -p ~/.m2
    echo '<settings>
      <mirrors>
        <mirror>
          <id>proxynd</id>
          <mirrorOf>*</mirrorOf>
          <url>${{ secrets.MAVEN_PROXY_URL }}</url>
        </mirror>
      </mirrors>
    </settings>' > ~/.m2/settings.xml
    
- name: Build with Maven
  run: mvn clean install
```

### Docker
```dockerfile
# Dockerfile
FROM maven:3.8-openjdk-11 AS build

# Maven 설정 복사
COPY settings.xml /root/.m2/settings.xml

# 소스 코드 복사 및 빌드
COPY . /app
WORKDIR /app
RUN mvn clean package

FROM openjdk:11-jre-slim
COPY --from=build /app/target/*.jar app.jar
ENTRYPOINT ["java", "-jar", "/app.jar"]
```

## 성능 최적화

### 병렬 다운로드
```xml
<!-- settings.xml에 추가 -->
<settings>
  <proxies/>
  <mirrors/>
  <profiles>
    <profile>
      <id>parallel-downloads</id>
      <properties>
        <maven.artifact.threads>5</maven.artifact.threads>
      </properties>
    </profile>
  </profiles>
  <activeProfiles>
    <activeProfile>parallel-downloads</activeProfile>
  </activeProfiles>
</settings>
```

### 오프라인 모드
```bash
# 모든 의존성을 먼저 다운로드
mvn dependency:go-offline

# 오프라인 모드로 빌드
mvn -o clean install
```

## 프라이빗 저장소

프라이빗 Maven 저장소를 프록시하는 경우:

```yaml
# maven-proxy.yaml에 추가
proxies:
  - name: company-releases
    url: https://nexus.company.com/repository/maven-releases/
    auth:
      type: BASIC
      username: ${NEXUS_USER}
      password: ${NEXUS_PASS}
  - name: company-snapshots
    url: https://nexus.company.com/repository/maven-snapshots/
    auth:
      type: BASIC
      username: ${NEXUS_USER}
      password: ${NEXUS_PASS}
```