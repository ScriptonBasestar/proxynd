package proxy

import "strings"

// getAptContentType APT 파일 타입에 따른 Content-Type 반환
func getAptContentType(filename string) string {
	switch {
	case strings.HasSuffix(filename, ".gz"):
		return MimeApplicationXGzip
	case strings.HasSuffix(filename, ".bz2"):
		return MimeApplicationXBzip2
	case strings.HasSuffix(filename, ".xz"):
		return "application/x-xz"
	case strings.HasSuffix(filename, ".deb"):
		return "application/vnd.debian.binary-package"
	case filename == "Release" || filename == "InRelease":
		return MimeTextPlain
	case filename == "Release.gpg":
		return MimeApplicationPGPSignature
	case filename == "Packages" || strings.HasPrefix(filename, "Packages."):
		return MimeTextPlain
	case strings.HasSuffix(filename, ".dsc"):
		return MimeTextPlain
	case strings.HasSuffix(filename, ".changes"):
		return MimeTextPlain
	default:
		return MimeApplicationOctetStream
	}
}

// isInlineFile 파일이 inline으로 표시되어야 하는지 확인
func isInlineFile(filename string) bool {
	inlineFiles := []string{
		"Release",
		"InRelease",
		"Release.gpg",
		"Packages",
		"Packages.gz",
		"Packages.bz2",
		"Packages.xz",
		"Sources",
		"Sources.gz",
		"Sources.bz2",
		"Sources.xz",
	}

	for _, f := range inlineFiles {
		if filename == f {
			return true
		}
	}
	return false
}
