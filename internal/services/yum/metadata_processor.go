package yum

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"proxynd/internal/domain/yum"
	"proxynd/logging"
)

// metadataProcessorImpl YUM 메타데이터 처리 서비스 구현
type metadataProcessorImpl struct {
	config yum.ProxyConfig
	logger logging.Logger
}

// RepomdXML repomd.xml 구조체
type RepomdXML struct {
	XMLName xml.Name `xml:"repomd"`
	Data    []struct {
		Type     string `xml:"type,attr"`
		Location struct {
			Href string `xml:"href,attr"`
		} `xml:"location"`
		Checksum struct {
			Type  string `xml:"type,attr"`
			Value string `xml:",chardata"`
		} `xml:"checksum"`
		Size      int64  `xml:"size"`
		Timestamp string `xml:"timestamp"`
	} `xml:"data"`
}

// PrimaryXML primary.xml 구조체
type PrimaryXML struct {
	XMLName  xml.Name `xml:"metadata"`
	Packages []struct {
		Name    string `xml:"name"`
		Arch    string `xml:"arch"`
		Version struct {
			Epoch string `xml:"epoch,attr"`
			Ver   string `xml:"ver,attr"`
			Rel   string `xml:"rel,attr"`
		} `xml:"version"`
		Summary     string `xml:"summary"`
		Description string `xml:"description"`
		Packager    string `xml:"packager"`
		URL         string `xml:"url"`
		Time        struct {
			File  string `xml:"file,attr"`
			Build string `xml:"build,attr"`
		} `xml:"time"`
		Size struct {
			Package   int64 `xml:"package,attr"`
			Installed int64 `xml:"installed,attr"`
			Archive   int64 `xml:"archive,attr"`
		} `xml:"size"`
		Location struct {
			Href string `xml:"href,attr"`
		} `xml:"location"`
		Format struct {
			License     string `xml:"license"`
			Vendor      string `xml:"vendor"`
			Group       string `xml:"group"`
			BuildHost   string `xml:"buildhost"`
			SourceRpm   string `xml:"sourcerpm"`
			HeaderRange struct {
				Start int64 `xml:"start,attr"`
				End   int64 `xml:"end,attr"`
			} `xml:"header-range"`
			Requires []struct {
				Name  string `xml:"name,attr"`
				Flags string `xml:"flags,attr"`
				Epoch string `xml:"epoch,attr"`
				Ver   string `xml:"ver,attr"`
				Rel   string `xml:"rel,attr"`
			} `xml:"requires>entry"`
			Provides []struct {
				Name  string `xml:"name,attr"`
				Flags string `xml:"flags,attr"`
				Epoch string `xml:"epoch,attr"`
				Ver   string `xml:"ver,attr"`
				Rel   string `xml:"rel,attr"`
			} `xml:"provides>entry"`
		} `xml:"format"`
	} `xml:"package"`
}

// NewMetadataProcessor YUM 메타데이터 처리기 생성
func NewMetadataProcessor(
	config yum.ProxyConfig,
	logger logging.Logger,
) yum.MetadataProcessor {
	return &metadataProcessorImpl{
		config: config,
		logger: logger,
	}
}

// ProcessRepomd repomd.xml 처리
func (m *metadataProcessorImpl) ProcessRepomd(ctx context.Context, data []byte) (*yum.RepoMetadata, error) {
	var repomd RepomdXML
	if err := xml.Unmarshal(data, &repomd); err != nil {
		m.logger.Error("YUM repomd.xml 파싱 실패", logging.F("error", err.Error()))
		return nil, fmt.Errorf("failed to parse repomd.xml: %w", err)
	}

	// primary 메타데이터 정보 추출
	for _, dataItem := range repomd.Data {
		if dataItem.Type == "primary" {
			metadata := &yum.RepoMetadata{
				Repository:   "unknown", // 호출자에서 설정
				MetadataType: "repomd",
				Path:         dataItem.Location.Href,
				Size:         dataItem.Size,
				Checksum:     fmt.Sprintf("%s:%s", dataItem.Checksum.Type, dataItem.Checksum.Value),
				LastModified: m.parseTimestamp(dataItem.Timestamp),
			}

			m.logger.Debug("YUM repomd.xml 처리 완료",
				logging.F("path", dataItem.Location.Href),
				logging.F("size", dataItem.Size))

			return metadata, nil
		}
	}

	return nil, fmt.Errorf("primary metadata not found in repomd.xml")
}

// ProcessPrimaryXML primary.xml 처리
func (m *metadataProcessorImpl) ProcessPrimaryXML(ctx context.Context, data []byte) ([]*yum.PackageInfo, error) {
	var primary PrimaryXML
	if err := xml.Unmarshal(data, &primary); err != nil {
		m.logger.Error("YUM primary.xml 파싱 실패", logging.F("error", err.Error()))
		return nil, fmt.Errorf("failed to parse primary.xml: %w", err)
	}

	var packages []*yum.PackageInfo
	for _, pkg := range primary.Packages {
		packageInfo := &yum.PackageInfo{
			Name:         pkg.Name,
			Version:      pkg.Version.Ver,
			Release:      pkg.Version.Rel,
			Architecture: pkg.Arch,
			Summary:      pkg.Summary,
			Description:  pkg.Description,
			Size:         pkg.Size.Package,
			Vendor:       pkg.Format.Vendor,
			License:      pkg.Format.License,
			Group:        pkg.Format.Group,
			URL:          pkg.URL,
			BuildTime:    pkg.Time.Build,
		}

		// 의존성 정보 추가
		for _, req := range pkg.Format.Requires {
			if req.Name != "" {
				packageInfo.Requires = append(packageInfo.Requires, req.Name)
			}
		}

		// 제공 정보 추가
		for _, prov := range pkg.Format.Provides {
			if prov.Name != "" {
				packageInfo.Provides = append(packageInfo.Provides, prov.Name)
			}
		}

		packages = append(packages, packageInfo)
	}

	m.logger.Debug("YUM primary.xml 처리 완료",
		logging.F("package_count", len(packages)))

	return packages, nil
}

// ProcessFilelistsXML filelists.xml 처리
func (m *metadataProcessorImpl) ProcessFilelistsXML(ctx context.Context, data []byte) (map[string][]string, error) {
	// 간단한 구현 - 실제로는 복잡한 XML 파싱이 필요
	result := make(map[string][]string)

	// 파일 목록 추출 로직 (실제 구현에서는 더 정교한 XML 파싱 필요)
	content := string(data)
	lines := strings.Split(content, "\n")

	var currentPackage string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, `<package`) && strings.Contains(line, `name=`) {
			// 패키지 이름 추출
			start := strings.Index(line, `name="`) + 6
			end := strings.Index(line[start:], `"`)
			if end > 0 {
				currentPackage = line[start : start+end]
			}
		} else if strings.Contains(line, `<file>`) && currentPackage != "" {
			// 파일 경로 추출
			start := strings.Index(line, `<file>`) + 6
			end := strings.Index(line, `</file>`)
			if end > start {
				filePath := line[start:end]
				result[currentPackage] = append(result[currentPackage], filePath)
			}
		}
	}

	m.logger.Debug("YUM filelists.xml 처리 완료",
		logging.F("package_count", len(result)))

	return result, nil
}

// ProcessOtherXML other.xml 처리
func (m *metadataProcessorImpl) ProcessOtherXML(ctx context.Context, data []byte) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// other.xml은 주로 changelog 정보를 포함
	content := string(data)

	// 간단한 통계 정보 추출
	result["size"] = len(data)
	result["changelog_entries"] = strings.Count(content, "<changelog>")
	result["package_count"] = strings.Count(content, "<package")
	result["processed_at"] = time.Now().Unix()

	m.logger.Debug("YUM other.xml 처리 완료",
		logging.F("size", len(data)),
		logging.F("changelog_entries", result["changelog_entries"]))

	return result, nil
}

// ValidateMetadata 메타데이터 유효성 검증
func (m *metadataProcessorImpl) ValidateMetadata(ctx context.Context, metadataType string, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("metadata is empty")
	}

	switch metadataType {
	case "repomd":
		var repomd RepomdXML
		if err := xml.Unmarshal(data, &repomd); err != nil {
			return fmt.Errorf("invalid repomd.xml format: %w", err)
		}

		if len(repomd.Data) == 0 {
			return fmt.Errorf("repomd.xml contains no data entries")
		}

	case "primary":
		var primary PrimaryXML
		if err := xml.Unmarshal(data, &primary); err != nil {
			return fmt.Errorf("invalid primary.xml format: %w", err)
		}

	default:
		// 기본 XML 유효성 검증
		var doc interface{}
		if err := xml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("invalid XML format: %w", err)
		}
	}

	m.logger.Debug("YUM 메타데이터 유효성 검증 완료",
		logging.F("type", metadataType),
		logging.F("size", len(data)))

	return nil
}

// 헬퍼 메서드들

func (m *metadataProcessorImpl) parseTimestamp(timestamp string) time.Time {
	// Unix 타임스탬프 파싱
	if timestamp == "" {
		return time.Now()
	}

	// 다양한 타임스탬프 형식 지원
	formats := []string{
		"1136239445", // Unix timestamp
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestamp); err == nil {
			return t
		}
	}

	// 파싱 실패 시 현재 시간 반환
	return time.Now()
}
