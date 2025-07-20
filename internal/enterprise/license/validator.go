// Package license provides enterprise license validation
package license

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrInvalidLicense is returned when license is invalid
	ErrInvalidLicense = errors.New("invalid license")
	// ErrExpiredLicense is returned when license has expired
	ErrExpiredLicense = errors.New("license expired")
	// ErrInvalidSignature is returned when license signature is invalid
	ErrInvalidSignature = errors.New("invalid license signature")
	// ErrFeatureNotLicensed is returned when feature is not licensed
	ErrFeatureNotLicensed = errors.New("feature not licensed")
)

// 공개 키 - 실제로는 별도 파일이나 임베드로 관리
const publicKeyPEM = `
-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA1234567890...
실제 공개 키로 교체 필요
-----END PUBLIC KEY-----
`

// License represents license information
type License struct {
	ID         string    `json:"id"`
	Company    string    `json:"company"`
	Email      string    `json:"email"`
	Features   []string  `json:"features"`
	MaxServers int       `json:"max_servers"`
	MaxUsers   int       `json:"max_users"`
	IssuedAt   time.Time `json:"issued_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Type       string    `json:"type"` // "trial", "starter", "professional", "enterprise", "ultimate"
	Signature  string    `json:"signature"`
}

// Validator handles license validation
type Validator struct {
	publicKey *rsa.PublicKey
	license   *License
}

// NewValidator creates a new license validator
func NewValidator() (*Validator, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	publicKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	return &Validator{
		publicKey: publicKey,
	}, nil
}

// LoadLicense loads and validates license data
func (v *Validator) LoadLicense(licenseData []byte) error {
	var license License
	if err := json.Unmarshal(licenseData, &license); err != nil {
		return fmt.Errorf("failed to parse license: %w", err)
	}

	// 서명 검증
	if err := v.verifySignature(&license); err != nil {
		return err
	}

	// 만료 확인
	if time.Now().After(license.ExpiresAt) {
		return ErrExpiredLicense
	}

	v.license = &license
	return nil
}

// verifySignature verifies the license signature
func (v *Validator) verifySignature(license *License) error {
	// 서명 제외한 라이센스 데이터 직렬화
	sig := license.Signature
	license.Signature = ""

	data, err := json.Marshal(license)
	if err != nil {
		return err
	}

	license.Signature = sig

	// 서명 디코드
	signature, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// 해시 계산
	hashed := sha256.Sum256(data)

	// 서명 검증
	err = rsa.VerifyPKCS1v15(v.publicKey, crypto.SHA256, hashed[:], signature)
	if err != nil {
		return ErrInvalidSignature
	}

	return nil
}

// IsValid checks if the license is valid
func (v *Validator) IsValid() bool {
	if v.license == nil {
		return false
	}
	return time.Now().Before(v.license.ExpiresAt)
}

// HasFeature checks if a specific feature is licensed
func (v *Validator) HasFeature(feature string) bool {
	if v.license == nil || !v.IsValid() {
		return false
	}

	// "ultimate" 라이센스는 모든 기능 포함
	if v.license.Type == "ultimate" {
		return true
	}

	for _, f := range v.license.Features {
		if f == feature || f == "*" {
			return true
		}
	}
	return false
}

// GetLicense returns the current license information
func (v *Validator) GetLicense() *License {
	if v.license == nil || !v.IsValid() {
		return nil
	}
	return v.license
}

// GetLicenseInfo returns a summary of license information
func (v *Validator) GetLicenseInfo() map[string]interface{} {
	if v.license == nil {
		return map[string]interface{}{
			"valid": false,
			"type":  "none",
		}
	}

	return map[string]interface{}{
		"valid":       v.IsValid(),
		"type":        v.license.Type,
		"company":     v.license.Company,
		"expires_at":  v.license.ExpiresAt,
		"features":    v.license.Features,
		"max_servers": v.license.MaxServers,
		"max_users":   v.license.MaxUsers,
	}
}

// CheckServerLimit validates server count against license limits
func (v *Validator) CheckServerLimit(currentServers int) error {
	if v.license == nil || !v.IsValid() {
		return ErrInvalidLicense
	}

	if v.license.MaxServers > 0 && currentServers > v.license.MaxServers {
		return fmt.Errorf("server limit exceeded: %d/%d", currentServers, v.license.MaxServers)
	}

	return nil
}

// FeatureGate controls feature access based on license
type FeatureGate struct {
	validator *Validator
}

// NewFeatureGate creates a new feature gate
func NewFeatureGate(validator *Validator) *FeatureGate {
	return &FeatureGate{validator: validator}
}

// IsEnabled checks if a feature is enabled
func (fg *FeatureGate) IsEnabled(feature string) bool {
	if fg.validator == nil {
		return false
	}
	return fg.validator.HasFeature(feature)
}

// RequireFeature ensures a feature is available or returns an error
func (fg *FeatureGate) RequireFeature(feature string) error {
	if !fg.IsEnabled(feature) {
		return fmt.Errorf("%w: %s", ErrFeatureNotLicensed, feature)
	}
	return nil
}
