package common

import (
	"crypto"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"strings"

	"proxynd/internal/ports"
)

// Package manager type constants
const (
	PMTypeMaven    = "maven"
	PMTypeNpm      = "npm"
	PMTypeApt      = "apt"
	PMTypePypi     = "pypi"
	PMTypeYum      = "yum"
	PMTypeApk      = "apk"
	PMTypeDocker   = "docker"
	PMTypePip      = "pip"
	PMTypeRegistry = "registry"
	PMTypeAll      = "all"
)

// Default repository constants
const (
	DefaultRepoAlpine    = "alpine"
	DefaultRepoCentral   = "central"
	DefaultRepoCentos    = "centos"
	DefaultRepoDockerHub = "dockerhub"
	DefaultRepoPyPI      = "pypi"
)

// Common MIME type constants
const (
	MimeTextPlain                = "text/plain"
	MimeApplicationJSON          = "application/json"
	MimeApplicationXML           = "application/xml"
	MimeApplicationXGzip         = "application/x-gzip"
	MimeApplicationZip           = "application/zip"
	MimeApplicationJavaArchive   = "application/java-archive"
	MimeApplicationXRpm          = "application/x-rpm"
	MimeApplicationOctetStream   = "application/octet-stream"
)

// Common URLs and endpoints
const (
	DefaultLocalhostBaseURL = "http://localhost:8080"
)

// User agent categories
const (
	UserAgentDocker    = "docker"
	UserAgentUnknown   = "unknown"
	UserAgentForbidden = "forbidden"
	UserAgentApk       = "apk"
)

// SignatureVerifier implements ports.SignatureVerifier for all package managers
type SignatureVerifier struct{}

// NewSignatureVerifier creates a new signature verifier
func NewSignatureVerifier() ports.SignatureVerifier {
	return &SignatureVerifier{}
}

// VerifySignature verifies package signature
func (v *SignatureVerifier) VerifySignature(pmType string, content []byte, signature []byte) error {
	switch pmType {
	case PMTypeMaven:
		return v.verifyMavenSignature(content, signature)
	case PMTypeNpm:
		return v.verifyNpmSignature(content, signature)
	case PMTypeApt:
		return v.verifyAptSignature(content, signature)
	case PMTypePypi:
		return v.verifyPypiSignature(content, signature)
	case PMTypeYum:
		return v.verifyYumSignature(content, signature)
	case PMTypeApk:
		return v.verifyApkSignature(content, signature)
	case PMTypeRegistry:
		return v.verifyRegistrySignature(content, signature)
	default:
		return v.verifyGenericSignature(content, signature)
	}
}

// GenerateSignature generates signature for content
func (v *SignatureVerifier) GenerateSignature(pmType string, content []byte) ([]byte, error) {
	switch pmType {
	case PMTypeMaven:
		return v.generateMavenSignature(content)
	case PMTypeNpm:
		return v.generateNpmSignature(content)
	case PMTypeApt:
		return v.generateAptSignature(content)
	case PMTypePypi:
		return v.generatePypiSignature(content)
	case PMTypeYum:
		return v.generateYumSignature(content)
	case PMTypeApk:
		return v.generateApkSignature(content)
	case PMTypeRegistry:
		return v.generateRegistrySignature(content)
	default:
		return v.generateGenericSignature(content)
	}
}

// SupportedSignatureTypes returns supported signature types
func (v *SignatureVerifier) SupportedSignatureTypes(pmType string) []string {
	switch pmType {
	case PMTypeMaven:
		return []string{"md5", "sha1", "sha256", "sha512", "asc"} // Maven checksums and PGP
	case PMTypeNpm:
		return []string{"sha1", "sha512", "integrity"} // NPM integrity hashes
	case PMTypeApt:
		return []string{"gpg", "sha256"} // APT GPG signatures
	case PMTypePypi:
		return []string{"md5", "sha256", "gpg"} // PyPI hashes and GPG
	case PMTypeYum:
		return []string{"gpg", "sha256"} // YUM GPG signatures
	case PMTypeApk:
		return []string{"rsa"} // APK RSA signatures
	case PMTypeRegistry:
		return []string{"sha256", "notary"} // Docker content trust
	default:
		return []string{"sha256"} // Default hash
	}
}

// Maven signature verification
func (v *SignatureVerifier) verifyMavenSignature(content []byte, signature []byte) error {
	signatureStr := strings.TrimSpace(string(signature))

	// Try different hash algorithms based on signature length
	switch len(signatureStr) {
	case 32: // MD5
		return v.verifyHashSignature(content, signatureStr, crypto.MD5)
	case 40: // SHA1
		return v.verifyHashSignature(content, signatureStr, crypto.SHA1)
	case 64: // SHA256
		return v.verifyHashSignature(content, signatureStr, crypto.SHA256)
	case 128: // SHA512
		return v.verifyHashSignature(content, signatureStr, crypto.SHA512)
	default:
		// Assume PGP signature
		return v.verifyPGPSignature(content, signature)
	}
}

func (v *SignatureVerifier) generateMavenSignature(content []byte) ([]byte, error) {
	// Generate SHA1 hash (Maven default)
	hasher := sha1.New()
	hasher.Write(content)
	hash := hasher.Sum(nil)
	return []byte(hex.EncodeToString(hash)), nil
}

// NPM signature verification
func (v *SignatureVerifier) verifyNpmSignature(content []byte, signature []byte) error {
	signatureStr := strings.TrimSpace(string(signature))

	// NPM integrity format: "sha512-base64hash" or "sha1-hexhash"
	if strings.HasPrefix(signatureStr, "sha512-") {
		// TODO: Implement base64 SHA512 verification
		return fmt.Errorf("sha512 base64 verification not implemented")
	} else if strings.HasPrefix(signatureStr, "sha1-") {
		hashStr := strings.TrimPrefix(signatureStr, "sha1-")
		return v.verifyHashSignature(content, hashStr, crypto.SHA1)
	} else {
		// Assume SHA1 hex
		return v.verifyHashSignature(content, signatureStr, crypto.SHA1)
	}
}

func (v *SignatureVerifier) generateNpmSignature(content []byte) ([]byte, error) {
	// Generate SHA1 hash (NPM default)
	hasher := sha1.New()
	hasher.Write(content)
	hash := hasher.Sum(nil)
	return []byte("sha1-" + hex.EncodeToString(hash)), nil
}

// APT signature verification
func (v *SignatureVerifier) verifyAptSignature(content []byte, signature []byte) error {
	// APT uses GPG signatures
	return v.verifyPGPSignature(content, signature)
}

func (v *SignatureVerifier) generateAptSignature(content []byte) ([]byte, error) {
	// Generate SHA256 hash
	hasher := sha256.New()
	hasher.Write(content)
	hash := hasher.Sum(nil)
	return []byte(hex.EncodeToString(hash)), nil
}

// PyPI signature verification
func (v *SignatureVerifier) verifyPypiSignature(content []byte, signature []byte) error {
	signatureStr := strings.TrimSpace(string(signature))

	switch len(signatureStr) {
	case 32: // MD5
		return v.verifyHashSignature(content, signatureStr, crypto.MD5)
	case 64: // SHA256
		return v.verifyHashSignature(content, signatureStr, crypto.SHA256)
	default:
		// Assume PGP signature
		return v.verifyPGPSignature(content, signature)
	}
}

func (v *SignatureVerifier) generatePypiSignature(content []byte) ([]byte, error) {
	// Generate SHA256 hash (PyPI default)
	hasher := sha256.New()
	hasher.Write(content)
	hash := hasher.Sum(nil)
	return []byte(hex.EncodeToString(hash)), nil
}

// YUM signature verification
func (v *SignatureVerifier) verifyYumSignature(content []byte, signature []byte) error {
	// YUM uses GPG signatures
	return v.verifyPGPSignature(content, signature)
}

func (v *SignatureVerifier) generateYumSignature(content []byte) ([]byte, error) {
	// Generate SHA256 hash
	hasher := sha256.New()
	hasher.Write(content)
	hash := hasher.Sum(nil)
	return []byte(hex.EncodeToString(hash)), nil
}

// APK signature verification
func (v *SignatureVerifier) verifyApkSignature(content []byte, signature []byte) error {
	// APK uses RSA signatures embedded in packages
	// TODO: Implement APK-specific RSA verification
	return fmt.Errorf("apk signature verification not implemented")
}

func (v *SignatureVerifier) generateApkSignature(content []byte) ([]byte, error) {
	// Generate SHA256 hash
	hasher := sha256.New()
	hasher.Write(content)
	hash := hasher.Sum(nil)
	return []byte(hex.EncodeToString(hash)), nil
}

// Docker registry signature verification
func (v *SignatureVerifier) verifyRegistrySignature(content []byte, signature []byte) error {
	// Docker uses content trust and notary
	// TODO: Implement Docker content trust verification
	return fmt.Errorf("docker signature verification not implemented")
}

func (v *SignatureVerifier) generateRegistrySignature(content []byte) ([]byte, error) {
	// Generate SHA256 hash (Docker default)
	hasher := sha256.New()
	hasher.Write(content)
	hash := hasher.Sum(nil)
	return []byte("sha256:" + hex.EncodeToString(hash)), nil
}

// Generic signature verification
func (v *SignatureVerifier) verifyGenericSignature(content []byte, signature []byte) error {
	signatureStr := strings.TrimSpace(string(signature))

	// Try to detect hash type by length
	switch len(signatureStr) {
	case 32:
		return v.verifyHashSignature(content, signatureStr, crypto.MD5)
	case 40:
		return v.verifyHashSignature(content, signatureStr, crypto.SHA1)
	case 64:
		return v.verifyHashSignature(content, signatureStr, crypto.SHA256)
	case 128:
		return v.verifyHashSignature(content, signatureStr, crypto.SHA512)
	default:
		return fmt.Errorf("unknown signature format")
	}
}

func (v *SignatureVerifier) generateGenericSignature(content []byte) ([]byte, error) {
	// Generate SHA256 hash (generic default)
	hasher := sha256.New()
	hasher.Write(content)
	hash := hasher.Sum(nil)
	return []byte(hex.EncodeToString(hash)), nil
}

// Helper methods
func (v *SignatureVerifier) verifyHashSignature(content []byte, expectedHash string, hashType crypto.Hash) error {
	switch hashType {
	case crypto.MD5:
		h := md5.New()
		h.Write(content)
		actualHash := hex.EncodeToString(h.Sum(nil))
		if actualHash != expectedHash {
			return fmt.Errorf("md5 hash mismatch: expected %s, got %s", expectedHash, actualHash)
		}
	case crypto.SHA1:
		h := sha1.New()
		h.Write(content)
		actualHash := hex.EncodeToString(h.Sum(nil))
		if actualHash != expectedHash {
			return fmt.Errorf("sha1 hash mismatch: expected %s, got %s", expectedHash, actualHash)
		}
	case crypto.SHA256:
		h := sha256.New()
		h.Write(content)
		actualHash := hex.EncodeToString(h.Sum(nil))
		if actualHash != expectedHash {
			return fmt.Errorf("sha256 hash mismatch: expected %s, got %s", expectedHash, actualHash)
		}
	case crypto.SHA512:
		h := sha512.New()
		h.Write(content)
		actualHash := hex.EncodeToString(h.Sum(nil))
		if actualHash != expectedHash {
			return fmt.Errorf("sha512 hash mismatch: expected %s, got %s", expectedHash, actualHash)
		}
	default:
		return fmt.Errorf("unsupported hash type: %v", hashType)
	}

	return nil
}

func (v *SignatureVerifier) verifyPGPSignature(content []byte, signature []byte) error {
	// TODO: Implement PGP signature verification
	// This would require integrating with a PGP library like golang.org/x/crypto/openpgp
	return fmt.Errorf("pgp signature verification not implemented")
}
