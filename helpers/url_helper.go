package helpers

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// JoinURL 기본 URL과 경로들을 결합합니다
func JoinURL(base string, paths ...string) string {
	if len(paths) == 0 {
		return base
	}
	p := path.Join(paths...)
	if p == "" {
		return strings.TrimRight(base, "/")
	}
	return fmt.Sprintf("%s/%s", strings.TrimRight(base, "/"), strings.TrimLeft(p, "/"))
}

// CleanPath URL 경로를 정리합니다 (중복 슬래시, dot 세그먼트 제거)
func CleanPath(p string) string {
	return path.Clean(p)
}

// ParsePackageName 패키지 경로에서 이름과 버전을 추출합니다
func ParsePackageName(packagePath string) (name, version string) {
	parts := strings.Split(packagePath, "/")
	
	// 스코프 패키지인 경우 (@types/node)
	if len(parts) >= 2 && strings.HasPrefix(parts[0], "@") {
		name = parts[0] + "/" + parts[1]
		if len(parts) > 2 {
			version = parts[2]
		}
		return
	}
	
	// 일반 패키지인 경우
	if len(parts) > 0 {
		name = parts[0]
		if len(parts) > 1 {
			version = parts[1]
		}
	}
	
	return
}

// IsValidURL URL이 유효한지 검증합니다
func IsValidURL(rawURL string) bool {
	if rawURL == "" {
		return false
	}
	
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	
	return u.Scheme == "http" || u.Scheme == "https"
}

// URLEncode URL 인코딩을 수행합니다
func URLEncode(s string) string {
	return url.QueryEscape(s)
}
