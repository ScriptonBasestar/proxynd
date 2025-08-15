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

// SignatureVerifier implements ports.SignatureVerifier for all package managers
type SignatureVerifier struct{}

// NewSignatureVerifier creates a new signature verifier
func NewSignatureVerifier() ports.SignatureVerifier {
	return &SignatureVerifier{}
}

// VerifySignature verifies package signature
func (v *SignatureVerifier) VerifySignature(pmType string, content []byte, signature []byte) error {
	switch pmType {
	case "maven":
		return v.verifyMavenSignature(content, signature)
	case "npm":
		return v.verifyNpmSignature(content, signature)
	case "apt":
		return v.verifyAptSignature(content, signature)
	case "pypi":
		return v.verifyPypiSignature(content, signature)
	case "yum":
		return v.verifyYumSignature(content, signature)
	case "apk":
		return v.verifyApkSignature(content, signature)
	case "registry":
		return v.verifyRegistrySignature(content, signature)
	default:
		return v.verifyGenericSignature(content, signature)
	}
}

// GenerateSignature generates signature for content
func (v *SignatureVerifier) GenerateSignature(pmType string, content []byte) ([]byte, error) {
	switch pmType {
	case "maven":
		return v.generateMavenSignature(content)
	case "npm":
		return v.generateNpmSignature(content)
	case "apt":
		return v.generateAptSignature(content)
	case "pypi":
		return v.generatePypiSignature(content)
	case "yum":
		return v.generateYumSignature(content)
	case "apk":
		return v.generateApkSignature(content)
	case "registry":
		return v.generateRegistrySignature(content)
	default:
		return v.generateGenericSignature(content)
	}
}

// SupportedSignatureTypes returns supported signature types
func (v *SignatureVerifier) SupportedSignatureTypes(pmType string) []string {
	switch pmType {
	case "maven":
		return []string{"md5", "sha1", "sha256", "sha512", "asc"} // Maven checksums and PGP
	case "npm":
		return []string{"sha1", "sha512", "integrity"}           // NPM integrity hashes
	case "apt":
		return []string{"gpg", "sha256"}                         // APT GPG signatures
	case "pypi":
		return []string{"md5", "sha256", "gpg"}                  // PyPI hashes and GPG
	case "yum":
		return []string{"gpg", "sha256"}                         // YUM GPG signatures
	case "apk":
		return []string{"rsa"}                                   // APK RSA signatures
	case "registry":
		return []string{"sha256", "notary"}                      // Docker content trust
	default:
		return []string{"sha256"}                                // Default hash
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