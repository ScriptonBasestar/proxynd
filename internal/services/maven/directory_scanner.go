package maven

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// parsedEntry HTML 파싱 결과
type parsedEntry struct {
	Name         string
	Size         int64
	LastModified time.Time
}

// directoryScanner HTML 디렉토리 리스팅 스캐너
type directoryScanner struct {
	// Apache/Nginx 스타일 디렉토리 리스팅 패턴들
	patterns []*regexp.Regexp
}

// parseHTML HTML 디렉토리 리스팅 파싱
func (s *directoryScanner) parseHTML(reader io.Reader) ([]parsedEntry, error) {
	s.initPatterns()

	var entries []parsedEntry
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if entry := s.extractEntryFromLine(line); entry != nil {
			entries = append(entries, *entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

// initPatterns 정규식 패턴 초기화
func (s *directoryScanner) initPatterns() {
	if s.patterns != nil {
		return
	}

	// 다양한 웹서버의 디렉토리 리스팅 패턴들
	patternStrings := []string{
		// Apache 스타일: <a href="filename">filename</a> 13-Mar-2023 10:30 1234
		`<a\s+href="([^"]+)"[^>]*>([^<]+)</a>\s+(\d{2}-\w{3}-\d{4}\s+\d{2}:\d{2})\s+(\d+|-)`,

		// Nginx 스타일: <a href="filename">filename</a> 2023-03-13 10:30 1234
		`<a\s+href="([^"]+)"[^>]*>([^<]+)</a>\s+(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2})\s+(\d+|-)`,

		// 간단한 형태: <a href="filename">filename</a>
		`<a\s+href="([^"]+)"[^>]*>([^<]+)</a>`,

		// Artifactory 스타일
		`href="([^"]+)"[^>]*title="([^"]*)"[^>]*>\s*([^<]+)\s*</a>.*?(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}).*?(\d+\s*bytes)?`,
	}

	s.patterns = make([]*regexp.Regexp, len(patternStrings))
	for i, pattern := range patternStrings {
		s.patterns[i] = regexp.MustCompile(pattern)
	}
}

// extractEntryFromLine HTML 라인에서 엔트리 추출
func (s *directoryScanner) extractEntryFromLine(line string) *parsedEntry {
	// 상위 디렉토리 링크 제외
	if strings.Contains(line, "../") || strings.Contains(line, "Parent Directory") {
		return nil
	}

	// 각 패턴으로 시도
	for _, pattern := range s.patterns {
		if matches := pattern.FindStringSubmatch(line); matches != nil {
			return s.createEntryFromMatches(matches)
		}
	}

	return nil
}

// createEntryFromMatches 정규식 매치 결과로부터 엔트리 생성
func (s *directoryScanner) createEntryFromMatches(matches []string) *parsedEntry {
	if len(matches) < 3 {
		return nil
	}

	href := matches[1]
	name := matches[2]

	// href가 상대 경로가 아닌 경우 스킵
	if strings.HasPrefix(href, "http") || strings.HasPrefix(href, "//") {
		return nil
	}

	// 이름 정리
	name = strings.TrimSpace(name)
	if name == "" {
		name = href
	}

	entry := &parsedEntry{
		Name: name,
	}

	// 날짜와 크기 파싱 (사용 가능한 경우)
	if len(matches) >= 4 {
		if date := s.parseDateTime(matches[3]); !date.IsZero() {
			entry.LastModified = date
		}
	}

	if len(matches) >= 5 {
		if size := s.parseSize(matches[4]); size > 0 {
			entry.Size = size
		}
	}

	return entry
}

// parseDateTime 날짜/시간 문자열 파싱
func (s *directoryScanner) parseDateTime(dateStr string) time.Time {
	// 다양한 날짜 형식 시도
	formats := []string{
		"02-Jan-2006 15:04",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"Jan 02, 2006 15:04:05",
		"Monday, 02-Jan-06 15:04:05 MST",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	return time.Time{}
}

// parseSize 크기 문자열 파싱
func (s *directoryScanner) parseSize(sizeStr string) int64 {
	sizeStr = strings.TrimSpace(sizeStr)

	// "-" 는 디렉토리를 의미
	if sizeStr == "-" || sizeStr == "" {
		return 0
	}

	// "bytes" 제거
	sizeStr = strings.ReplaceAll(sizeStr, "bytes", "")
	sizeStr = strings.ReplaceAll(sizeStr, ",", "")
	sizeStr = strings.TrimSpace(sizeStr)

	if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
		return size
	}

	return 0
}
