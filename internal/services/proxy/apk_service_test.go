package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApkService_classifyRequest(t *testing.T) {
	// Create a minimal service for testing classification
	service := &ApkService{
		BaseProxyService: &BaseProxyService{ProxyType: "apk"},
	}

	tests := []struct {
		name     string
		path     string
		expected ApkRequestType
	}{
		// APKINDEX
		{
			name:     "apkindex file",
			path:     "/v3.18/main/x86_64/APKINDEX.tar.gz",
			expected: ApkRequestTypeAPKINDEX,
		},
		{
			name:     "apkindex uppercase",
			path:     "/v3.18/community/APKINDEX.tar.gz",
			expected: ApkRequestTypeAPKINDEX,
		},

		// Package files
		{
			name:     "apk package",
			path:     "/v3.18/main/x86_64/alpine-base-3.18.0-r0.apk",
			expected: ApkRequestTypePackage,
		},
		{
			name:     "apk package nested",
			path:     "/alpine/v3.18/main/x86_64/busybox-1.36.0-r0.apk",
			expected: ApkRequestTypePackage,
		},

		// Signature files
		{
			name:     "sig file",
			path:     "/v3.18/main/x86_64/alpine-base-3.18.0-r0.apk.sig",
			expected: ApkRequestTypeSignature,
		},
		{
			name:     "rsa pub key",
			path:     "/alpine-keys/alpine-devel@lists.alpinelinux.org-5e69ca50.rsa.pub",
			expected: ApkRequestTypeSignature,
		},

		// Description files
		{
			name:     "txt file",
			path:     "/v3.18/main/x86_64/DESCRIPTION.txt",
			expected: ApkRequestTypeDescription,
		},
		{
			name:     "description file",
			path:     "/v3.18/main/description",
			expected: ApkRequestTypeDescription,
		},

		// Unknown
		{
			name:     "unknown path",
			path:     "/api/v1/health",
			expected: ApkRequestTypeUnknown,
		},
		{
			name:     "unknown file",
			path:     "/some/random/file.bin",
			expected: ApkRequestTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.classifyRequest(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApkService_getContentType(t *testing.T) {
	service := &ApkService{
		BaseProxyService: &BaseProxyService{ProxyType: "apk"},
	}

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "apk package",
			path:     "/v3.18/main/x86_64/alpine-base-3.18.0-r0.apk",
			expected: ApkContentTypeAPK,
		},
		{
			name:     "apkindex tar.gz",
			path:     "/v3.18/main/x86_64/APKINDEX.tar.gz",
			expected: ApkContentTypeTarGz,
		},
		{
			name:     "gzip file",
			path:     "/some/file.gz",
			expected: ApkContentTypeGzip,
		},
		{
			name:     "sig file",
			path:     "/alpine-base.apk.sig",
			expected: ApkContentTypeSig,
		},
		{
			name:     "rsa pub key",
			path:     "/keys/alpine.rsa.pub",
			expected: ApkContentTypeSig,
		},
		{
			name:     "txt file",
			path:     "/DESCRIPTION.txt",
			expected: ApkContentTypeText,
		},
		{
			name:     "unknown",
			path:     "/unknown/file.bin",
			expected: ApkContentTypeDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.getContentType(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
