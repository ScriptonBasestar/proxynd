package maven

import (
	"fmt"
	"regexp"
	"strings"

	"proxynd/internal/domain/maven"
	"proxynd/internal/logging"
)

// pathAnalyzerImpl PathAnalyzer 인터페이스 구현
type pathAnalyzerImpl struct {
	logger logging.Logger

	// 캐시된 정규식 패턴들
	versionPattern  *regexp.Regexp
	artifactPattern *regexp.Regexp
	snapshotPattern *regexp.Regexp
}

// NewPathAnalyzer PathAnalyzer 생성자
func NewPathAnalyzer(logger logging.Logger) maven.PathAnalyzer {
	return &pathAnalyzerImpl{
		logger: logger,

		// 버전 패턴 컴파일
		versionPattern: regexp.MustCompile(`^\d+(\.\d+)*(-[a-zA-Z0-9]+)*$`),

		// 아티팩트 패턴 컴파일 (하이픈이나 점이 포함된 식별자)
		artifactPattern: regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-_.]*[a-zA-Z0-9]$`),

		// 스냅샷 패턴 컴파일
		snapshotPattern: regexp.MustCompile(`SNAPSHOT|snapshot`),
	}
}

// ParsePath Maven 경로를 분석하여 구조화된 정보 반환
func (p *pathAnalyzerImpl) ParsePath(path string) (*maven.PathInfo, error) {
	// 경로 정규화 (앞뒤 슬래시 제거)
	path = strings.Trim(path, "/")
	if path == "" {
		return &maven.PathInfo{
			Type:  maven.TypeDirectory,
			Level: 0,
		}, nil
	}

	parts := strings.Split(path, "/")
	level := len(parts)

	p.logger.Debug("Parsing Maven path",
		logging.F("path", path),
		logging.F("parts", parts),
		logging.F("level", level),
	)

	info := &maven.PathInfo{
		Level: level,
	}

	// Maven 표준 구조 분석
	// 예: org/apache/httpcomponents/client5/httpclient5/5.3.1/
	//     ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ 그룹
	//                                   ^^^^^^^^^^^ 아티팩트
	//                                               ^^^^^ 버전

	// 마지막부터 거꾸로 탐색하여 버전 찾기
	versionIndex := -1
	for i := level - 1; i >= 0; i-- {
		if p.IsVersionLike(parts[i]) {
			versionIndex = i
			break
		}
	}

	if versionIndex > 0 {
		// 버전이 발견된 경우
		info.Type = maven.TypeVersion
		info.ArtifactID = parts[versionIndex-1]
		info.Version = parts[versionIndex]
		info.GroupID = strings.Join(parts[:versionIndex-1], ".")
	} else {
		// 버전이 없는 경우 - 아티팩트 패턴 확인
		// 아티팩트는 보통 하이픈을 포함하거나 마지막 부분에 위치
		artifactIndex := -1
		for i := level - 1; i >= 0; i-- {
			if p.IsLikelyArtifact(parts[i]) {
				artifactIndex = i
				break
			}
		}

		if artifactIndex > 0 {
			// 아티팩트가 발견된 경우
			info.Type = maven.TypeArtifact
			info.ArtifactID = parts[artifactIndex]
			info.GroupID = strings.Join(parts[:artifactIndex], ".")
		} else {
			// 그룹 경로로 판단
			info.Type = maven.TypeGroup
			info.GroupID = strings.Join(parts, ".")
		}
	}

	p.logger.Debug("Path analysis completed",
		logging.F("path", path),
		logging.F("type", info.Type),
		logging.F("groupId", info.GroupID),
		logging.F("artifactId", info.ArtifactID),
		logging.F("version", info.Version),
	)

	return info, nil
}

// IsVersionLike 문자열이 버전과 유사한지 확인
func (p *pathAnalyzerImpl) IsVersionLike(s string) bool {
	if s == "" {
		return false
	}

	// 일반적인 버전 키워드 확인
	versionKeywords := []string{
		"SNAPSHOT", "snapshot",
		"RELEASE", "release",
		"Final", "final",
		"GA", "ga",
		"RC", "rc",
		"Alpha", "alpha", "ALPHA",
		"Beta", "beta", "BETA",
		"M", // Milestone
	}

	upperS := strings.ToUpper(s)
	for _, keyword := range versionKeywords {
		if strings.Contains(upperS, strings.ToUpper(keyword)) {
			return true
		}
	}

	// 정규식 패턴 확인
	if p.versionPattern.MatchString(s) {
		return true
	}

	// 숫자로 시작하는지 확인
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		return true
	}

	// 날짜 형식 확인 (20230101, 2023-01-01 등)
	if p.isDateLike(s) {
		return true
	}

	return false
}

// IsLikelyArtifact 이름이 아티팩트와 유사한지 확인
func (p *pathAnalyzerImpl) IsLikelyArtifact(name string) bool {
	if name == "" {
		return false
	}

	// 일반적인 아티팩트 접미사 패턴
	artifactSuffixes := []string{
		"-core", "-api", "-impl", "-client", "-server",
		"-common", "-util", "-utils", "-test", "-tests",
		"-web", "-service", "-dao", "-model", "-entity",
		"-spring", "-boot", "-starter", "-plugin",
		"-parent", "-bom", "-dependencies",
	}

	lowerName := strings.ToLower(name)
	for _, suffix := range artifactSuffixes {
		if strings.HasSuffix(lowerName, suffix) {
			return true
		}
	}

	// 일반적인 아티팩트 접두사 패턴
	artifactPrefixes := []string{
		"spring-", "jackson-", "commons-", "apache-",
		"google-", "junit-", "slf4j-", "logback-",
		"hibernate-", "mybatis-", "redis-",
	}

	for _, prefix := range artifactPrefixes {
		if strings.HasPrefix(lowerName, prefix) {
			return true
		}
	}

	// 정규식 패턴 확인 (영숫자, 하이픈, 점, 언더스코어 조합)
	if p.artifactPattern.MatchString(name) {
		return true
	}

	// 버전이 아니고 그룹도 아닌 경우 아티팩트일 가능성
	if !p.IsVersionLike(name) && !p.isGroupLike(name) {
		return true
	}

	return false
}

// ExtractGAV 경로에서 GroupID, ArtifactID, Version 추출
func (p *pathAnalyzerImpl) ExtractGAV(path string) (groupID, artifactID, version string) {
	pathInfo, err := p.ParsePath(path)
	if err != nil {
		p.logger.Warn("Failed to parse path for GAV extraction",
			logging.F("path", path),
			logging.F("error", err),
		)
		return "", "", ""
	}

	return pathInfo.GroupID, pathInfo.ArtifactID, pathInfo.Version
}

// isDateLike 날짜와 유사한 형식인지 확인
func (p *pathAnalyzerImpl) isDateLike(s string) bool {
	// YYYYMMDD 형식
	if len(s) == 8 {
		for _, r := range s {
			if r < '0' || r > '9' {
				return false
			}
		}
		year := s[:4]
		if year >= "2000" && year <= "2099" {
			return true
		}
	}

	// YYYY-MM-DD 또는 YYYY.MM.DD 형식
	datePatterns := []string{
		`^\d{4}-\d{2}-\d{2}$`,
		`^\d{4}\.\d{2}\.\d{2}$`,
	}

	for _, pattern := range datePatterns {
		if matched, _ := regexp.MatchString(pattern, s); matched {
			return true
		}
	}

	return false
}

// isGroupLike 그룹과 유사한 이름인지 확인
func (p *pathAnalyzerImpl) isGroupLike(name string) bool {
	// 일반적인 그룹 패턴들
	groupPatterns := []string{
		"org", "com", "net", "io", "co", "gov", "edu",
		"apache", "springframework", "google", "microsoft",
		"eclipse", "junit", "slf4j", "hibernate",
	}

	lowerName := strings.ToLower(name)
	for _, pattern := range groupPatterns {
		if lowerName == pattern || strings.Contains(lowerName, pattern) {
			return true
		}
	}

	// 짧은 이름 (보통 3자 이하)은 그룹일 가능성
	if len(name) <= 3 && !p.IsVersionLike(name) {
		return true
	}

	return false
}

// GetPathType 경로의 Maven 타입 결정 (편의 메서드)
func (p *pathAnalyzerImpl) GetPathType(path string) maven.EntryType {
	pathInfo, err := p.ParsePath(path)
	if err != nil {
		return maven.TypeDirectory
	}
	return pathInfo.Type
}

// IsSnapshotVersion 스냅샷 버전인지 확인
func (p *pathAnalyzerImpl) IsSnapshotVersion(version string) bool {
	return p.snapshotPattern.MatchString(version)
}

// CompareVersions 버전 비교 (간단한 구현)
func (p *pathAnalyzerImpl) CompareVersions(v1, v2 string) int {
	// 스냅샷 버전 처리
	v1Clean := p.cleanVersionString(v1)
	v2Clean := p.cleanVersionString(v2)

	// 버전 파트 분할
	parts1 := p.splitVersion(v1Clean)
	parts2 := p.splitVersion(v2Clean)

	// 각 파트별 비교
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var part1, part2 string
		if i < len(parts1) {
			part1 = parts1[i]
		}
		if i < len(parts2) {
			part2 = parts2[i]
		}

		cmp := p.compareVersionPart(part1, part2)
		if cmp != 0 {
			return cmp
		}
	}

	return 0 // 동일한 버전
}

// cleanVersionString 버전 문자열 정리
func (p *pathAnalyzerImpl) cleanVersionString(version string) string {
	// SNAPSHOT 등 접미사 제거
	cleanVersion := strings.ReplaceAll(version, "-SNAPSHOT", "")
	cleanVersion = strings.ReplaceAll(cleanVersion, "-snapshot", "")
	return cleanVersion
}

// splitVersion 버전을 파트별로 분할
func (p *pathAnalyzerImpl) splitVersion(version string) []string {
	// 점과 하이픈으로 분할
	parts := regexp.MustCompile(`[.\-]`).Split(version, -1)

	var result []string
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}

// compareVersionPart 버전 파트 비교
func (p *pathAnalyzerImpl) compareVersionPart(part1, part2 string) int {
	if part1 == part2 {
		return 0
	}

	// 숫자 비교 시도
	if p1, err1 := p.parseVersionNumber(part1); err1 == nil {
		if p2, err2 := p.parseVersionNumber(part2); err2 == nil {
			if p1 < p2 {
				return -1
			} else if p1 > p2 {
				return 1
			}
			return 0
		}
	}

	// 문자열 비교
	if part1 < part2 {
		return -1
	}
	return 1
}

// parseVersionNumber 버전 번호 파싱
func (p *pathAnalyzerImpl) parseVersionNumber(part string) (int, error) {
	// 숫자 부분만 추출
	numStr := ""
	for _, r := range part {
		if r >= '0' && r <= '9' {
			numStr += string(r)
		} else {
			break
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("no number found")
	}

	var result int
	for _, r := range numStr {
		result = result*10 + int(r-'0')
	}

	return result, nil
}
