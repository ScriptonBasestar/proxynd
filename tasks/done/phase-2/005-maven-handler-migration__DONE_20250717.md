---
phase: 2
order: 5
source_plan: /docs/refactoring/01-dependency-injection.md
priority: high
tags: [maven, handler, migration, di]
---

# 📌 작업: Maven 핸들러 의존성 주입 마이그레이션

## 개요
Maven 핸들러를 새로운 의존성 주입 패턴으로 마이그레이션하고 체크섬 검증 기능을 강화합니다.

## 구현 내용

### 1. Maven 핸들러 구조체
```go
// handlers/proxy/maven_handler.go
package proxy

import (
    "crypto/sha1"
    "crypto/md5"
    "encoding/hex"
    "fmt"
    "path/filepath"
    "strings"

    "github.com/gofiber/fiber/v2"
    "proxynd/internal/app"
    "proxynd/internal/handlers"
    "proxynd/cache"
)

type MavenHandler struct {
    *handlers.BaseHandler
    cache            cache.Cache
    client           *http.Client
    checksumVerifier *ChecksumVerifier
}

func NewMavenHandler(container *app.Container) *MavenHandler {
    return &MavenHandler{
        BaseHandler:      handlers.NewBaseHandler(container, "maven-proxy", "maven"),
        cache:            container.Get("cache").(cache.Cache),
        client:           container.Get("httpClient").(*http.Client),
        checksumVerifier: NewChecksumVerifier(),
    }
}
```

### 2. Handle 메서드 구현
```go
func (h *MavenHandler) Handle(c *fiber.Ctx) error {
    // 캐시된 설정 사용
    config := h.GetConfig().MavenProxy
    if !config.Enabled {
        return fiber.ErrNotFound
    }

    // 아티팩트 경로 파싱
    artifactPath := c.Params("*")

    // SNAPSHOT 버전 처리
    if strings.Contains(artifactPath, "-SNAPSHOT") {
        return h.handleSnapshotArtifact(c, artifactPath, config)
    }

    // 캐시 확인
    cacheKey := h.generateCacheKey(artifactPath)
    if cached, err := h.cache.Get(cacheKey); err == nil {
        c.Set("X-Cache-Status", "HIT")
        return c.Send(cached)
    }

    // 업스트림 요청
    upstreamURL := fmt.Sprintf("%s/%s",
        strings.TrimRight(config.Repository, "/"), artifactPath)

    resp, err := h.fetchFromUpstream(upstreamURL)
    if err != nil {
        return h.handleUpstreamError(err)
    }

    // 체크섬 검증
    if h.isChecksumFile(artifactPath) {
        if err := h.validateChecksum(resp, artifactPath); err != nil {
            return err
        }
    }

    // POM 파일 처리
    if strings.HasSuffix(artifactPath, ".pom") {
        resp, err = h.processPOM(resp)
        if err != nil {
            return err
        }
    }

    // 캐시 저장
    if h.shouldCache(artifactPath) {
        ttl := h.getCacheTTL(artifactPath)
        h.cache.SetWithTTL(cacheKey, resp, ttl)
    }

    c.Set("X-Cache-Status", "MISS")
    return c.Send(resp)
}
```

### 3. 체크섬 검증 구현
```go
type ChecksumVerifier struct{}

func NewChecksumVerifier() *ChecksumVerifier {
    return &ChecksumVerifier{}
}

func (v *ChecksumVerifier) ValidateChecksum(data []byte, checksumPath string) error {
    // 체크섬 파일에서 예상 해시값 추출
    expectedHash := strings.TrimSpace(string(data))

    // 원본 파일 경로 유추
    originalPath := strings.TrimSuffix(checksumPath, filepath.Ext(checksumPath))

    // 원본 파일 가져오기 (캐시 또는 업스트림)
    originalData, err := v.getOriginalFile(originalPath)
    if err != nil {
        return fmt.Errorf("원본 파일 가져오기 실패: %v", err)
    }

    // 체크섬 계산
    var actualHash string
    if strings.HasSuffix(checksumPath, ".sha1") {
        actualHash = v.calculateSHA1(originalData)
    } else if strings.HasSuffix(checksumPath, ".md5") {
        actualHash = v.calculateMD5(originalData)
    } else {
        return fmt.Errorf("지원하지 않는 체크섬 형식: %s", checksumPath)
    }

    // 해시 비교
    if actualHash != expectedHash {
        return fmt.Errorf("체크섬 불일치: expected=%s, actual=%s",
            expectedHash, actualHash)
    }

    return nil
}

func (v *ChecksumVerifier) calculateSHA1(data []byte) string {
    h := sha1.New()
    h.Write(data)
    return hex.EncodeToString(h.Sum(nil))
}

func (v *ChecksumVerifier) calculateMD5(data []byte) string {
    h := md5.New()
    h.Write(data)
    return hex.EncodeToString(h.Sum(nil))
}
```

### 4. SNAPSHOT 처리
```go
func (h *MavenHandler) handleSnapshotArtifact(c *fiber.Ctx, path string, config configs.MavenProxyConfig) error {
    // SNAPSHOT은 항상 업스트림에서 최신 버전 확인
    metadataURL := h.buildMetadataURL(path, config.Repository)

    metadata, err := h.fetchMetadata(metadataURL)
    if err != nil {
        return err
    }

    // 최신 스냅샷 버전으로 경로 변경
    latestPath := h.resolveSnapshotPath(path, metadata)

    // 실제 아티팩트 다운로드
    upstreamURL := fmt.Sprintf("%s/%s",
        strings.TrimRight(config.Repository, "/"), latestPath)

    resp, err := h.fetchFromUpstream(upstreamURL)
    if err != nil {
        return err
    }

    // SNAPSHOT은 캐시하지 않음
    return c.Send(resp)
}
```

### 5. 캐시 정책
```go
func (h *MavenHandler) shouldCache(path string) bool {
    // SNAPSHOT 버전은 캐시하지 않음
    if strings.Contains(path, "-SNAPSHOT") {
        return false
    }

    // 메타데이터는 단기 캐시
    if strings.HasSuffix(path, "maven-metadata.xml") {
        return true
    }

    // 릴리즈 아티팩트는 장기 캐시
    return true
}

func (h *MavenHandler) getCacheTTL(path string) time.Duration {
    if strings.HasSuffix(path, "maven-metadata.xml") {
        return 30 * time.Minute
    }

    if strings.Contains(path, "-SNAPSHOT") {
        return 0 // 캐시 안함
    }

    // 릴리즈 아티팩트는 30일
    return 30 * 24 * time.Hour
}

func (h *MavenHandler) generateCacheKey(path string) string {
    // Maven 좌표 기반 캐시 키
    parts := strings.Split(path, "/")
    if len(parts) >= 3 {
        return fmt.Sprintf("maven:%s:%s:%s:%s",
            parts[0], parts[1], parts[2], filepath.Base(path))
    }
    return fmt.Sprintf("maven:%s", strings.ReplaceAll(path, "/", ":"))
}
```

## 실행 명령어
```bash
# 기존 핸들러 백업
cp handlers/proxy/maven_handler.go handlers/proxy/maven_handler.go.bak

# 새로운 핸들러 구현
# (위 코드를 파일에 작성)

# 빌드 테스트
make dev-run-direct

# Maven 프록시 테스트
curl -v http://localhost:8080/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar

# 체크섬 검증 테스트
curl -v http://localhost:8080/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar.sha1
```

## 검증 방법
1. 릴리즈 아티팩트 캐시 동작 확인
2. SNAPSHOT 버전 항상 최신 확인
3. 체크섬 검증 정상 동작
4. POM 파일 처리 확인

## 완료 조건
- [ ] MavenHandler 구조체 구현
- [ ] 의존성 주입 패턴 적용
- [ ] 체크섬 검증 기능 구현
- [ ] SNAPSHOT 버전 처리 구현
- [ ] 캐시 정책 적용
- [ ] 핸들러 등록 및 라우터 연결
- [ ] 단위 테스트 작성
- [ ] 통합 테스트 통과
