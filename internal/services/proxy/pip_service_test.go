package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPipService_classifyRequest(t *testing.T) {
	// Create a minimal service for testing classification
	service := &PipService{
		BaseProxyService: &BaseProxyService{ProxyType: "pip"},
	}

	tests := []struct {
		name     string
		path     string
		expected PipRequestType
	}{
		// Simple Index
		{
			name:     "root path",
			path:     "/",
			expected: PipRequestTypeSimpleIndex,
		},
		{
			name:     "simple index root",
			path:     "/simple",
			expected: PipRequestTypeSimpleIndex,
		},
		{
			name:     "simple index with trailing slash",
			path:     "/simple/",
			expected: PipRequestTypeSimpleIndex,
		},

		// Package Index
		{
			name:     "package simple index",
			path:     "/simple/requests/",
			expected: PipRequestTypePackageIndex,
		},
		{
			name:     "scoped package simple index",
			path:     "/simple/flask/",
			expected: PipRequestTypePackageIndex,
		},

		// JSON API
		{
			name:     "pypi json api",
			path:     "/pypi/requests/json",
			expected: PipRequestTypeJSON,
		},
		{
			name:     "pypi json api with version",
			path:     "/pypi/requests/2.28.0/json",
			expected: PipRequestTypeJSON,
		},

		// Package Files
		{
			name:     "wheel file",
			path:     "/packages/requests-2.28.0-py3-none-any.whl",
			expected: PipRequestTypePackageFile,
		},
		{
			name:     "tarball file",
			path:     "/packages/requests-2.28.0.tar.gz",
			expected: PipRequestTypePackageFile,
		},
		{
			name:     "zip file",
			path:     "/packages/requests-2.28.0.zip",
			expected: PipRequestTypePackageFile,
		},
		{
			name:     "egg file",
			path:     "/packages/requests-2.28.0.egg",
			expected: PipRequestTypePackageFile,
		},
		{
			name:     "bz2 tarball",
			path:     "/packages/requests-2.28.0.tar.bz2",
			expected: PipRequestTypePackageFile,
		},

		// Unknown
		{
			name:     "unknown path",
			path:     "/api/v1/health",
			expected: PipRequestTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.classifyRequest(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPipService_getContentType(t *testing.T) {
	service := &PipService{
		BaseProxyService: &BaseProxyService{ProxyType: "pip"},
	}

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "wheel file",
			path:     "/packages/requests-2.28.0-py3-none-any.whl",
			expected: PipContentTypeWheel,
		},
		{
			name:     "tar.gz file",
			path:     "/packages/requests-2.28.0.tar.gz",
			expected: PipContentTypeTarGz,
		},
		{
			name:     "zip file",
			path:     "/packages/requests-2.28.0.zip",
			expected: PipContentTypeZip,
		},
		{
			name:     "egg file",
			path:     "/packages/requests-2.28.0.egg",
			expected: PipContentTypeEgg,
		},
		{
			name:     "json api",
			path:     "/pypi/requests/json",
			expected: PipContentTypeJSON,
		},
		{
			name:     "simple index html",
			path:     "/simple/requests/",
			expected: PipContentTypeHTML,
		},
		{
			name:     "unknown",
			path:     "/unknown/file.bin",
			expected: PipContentTypeDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.getContentType(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
