package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestYumService_classifyRequest(t *testing.T) {
	// Create a minimal service for testing classification
	service := &YumService{
		BaseProxyService: &BaseProxyService{ProxyType: "yum"},
	}

	tests := []struct {
		name     string
		path     string
		expected YumRequestType
	}{
		// RepoMD
		{
			name:     "repomd.xml",
			path:     "/centos/7/os/x86_64/repodata/repomd.xml",
			expected: YumRequestTypeRepoMD,
		},
		{
			name:     "repomd.xml direct",
			path:     "/repodata/repomd.xml",
			expected: YumRequestTypeRepoMD,
		},

		// Primary metadata
		{
			name:     "primary.xml.gz",
			path:     "/repodata/primary.xml.gz",
			expected: YumRequestTypePrimary,
		},
		{
			name:     "primary.sqlite.bz2",
			path:     "/repodata/primary.sqlite.bz2",
			expected: YumRequestTypePrimary,
		},
		{
			name:     "primary.xml.xz",
			path:     "/repodata/primary.xml.xz",
			expected: YumRequestTypePrimary,
		},

		// Filelists metadata
		{
			name:     "filelists.xml.gz",
			path:     "/repodata/filelists.xml.gz",
			expected: YumRequestTypeFilelists,
		},
		{
			name:     "filelists.sqlite.bz2",
			path:     "/repodata/filelists.sqlite.bz2",
			expected: YumRequestTypeFilelists,
		},

		// Other metadata
		{
			name:     "other.xml.gz",
			path:     "/repodata/other.xml.gz",
			expected: YumRequestTypeOther,
		},
		{
			name:     "other.sqlite.bz2",
			path:     "/repodata/other.sqlite.bz2",
			expected: YumRequestTypeOther,
		},

		// Comps/Group
		{
			name:     "comps.xml",
			path:     "/repodata/comps.xml",
			expected: YumRequestTypeComps,
		},
		{
			name:     "group.xml",
			path:     "/repodata/group.xml",
			expected: YumRequestTypeComps,
		},

		// Modules
		{
			name:     "modules.yaml.gz",
			path:     "/repodata/modules.yaml.gz",
			expected: YumRequestTypeModules,
		},

		// RPM packages
		{
			name:     "rpm package",
			path:     "/Packages/nginx-1.20.0-1.el8.x86_64.rpm",
			expected: YumRequestTypeRPM,
		},
		{
			name:     "rpm package nested",
			path:     "/centos/8/AppStream/x86_64/os/Packages/n/nginx-1.20.0.rpm",
			expected: YumRequestTypeRPM,
		},

		// DRPM packages
		{
			name:     "drpm package",
			path:     "/drpms/nginx-1.19.0_1.20.0-1.el8.x86_64.drpm",
			expected: YumRequestTypeDRPM,
		},

		// Unknown
		{
			name:     "unknown path",
			path:     "/api/v1/health",
			expected: YumRequestTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.classifyRequest(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestYumService_getContentType(t *testing.T) {
	service := &YumService{
		BaseProxyService: &BaseProxyService{ProxyType: "yum"},
	}

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "rpm file",
			path:     "/Packages/nginx-1.20.0.rpm",
			expected: YumContentTypeRPM,
		},
		{
			name:     "drpm file",
			path:     "/drpms/nginx.drpm",
			expected: YumContentTypeRPM,
		},
		{
			name:     "gzip file",
			path:     "/repodata/primary.xml.gz",
			expected: YumContentTypeGzip,
		},
		{
			name:     "xz file",
			path:     "/repodata/primary.xml.xz",
			expected: YumContentTypeXz,
		},
		{
			name:     "bz2 file",
			path:     "/repodata/primary.sqlite.bz2",
			expected: YumContentTypeBz2,
		},
		{
			name:     "xml file",
			path:     "/repodata/repomd.xml",
			expected: YumContentTypeXML,
		},
		{
			name:     "unknown",
			path:     "/unknown/file.bin",
			expected: YumContentTypeDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.getContentType(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
