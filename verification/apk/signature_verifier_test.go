package apk

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-playground/assert/v2"
)

// TestNewSignatureVerifier 서명 검증기 생성 테스트
func TestNewSignatureVerifier(t *testing.T) {
	verifier := NewSignatureVerifier()

	assert.NotEqual(t, verifier, nil)
	assert.Equal(t, verifier.GetTrustedKeyCount(), 0)
}

// TestLoadTrustedKeys 신뢰할 수 있는 키 로드 테스트
func TestLoadTrustedKeys(t *testing.T) {
	// 임시 키 디렉토리 생성
	tempDir, err := os.MkdirTemp("", "apk_keys_test")
	assert.Equal(t, err, nil)
	defer os.RemoveAll(tempDir)

	// 테스트용 RSA 키 쌍 생성
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.Equal(t, err, nil)

	publicKey := &privateKey.PublicKey

	// 공개키를 PEM 형식으로 저장
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	assert.Equal(t, err, nil)

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	keyFile := filepath.Join(tempDir, "test_key.pem")
	err = os.WriteFile(keyFile, publicKeyPEM, 0644)
	assert.Equal(t, err, nil)

	// 서명 검증기로 키 로드
	verifier := NewSignatureVerifier()
	err = verifier.LoadTrustedKeys(tempDir)
	assert.Equal(t, err, nil)
	assert.Equal(t, verifier.GetTrustedKeyCount(), 1)

	// 키 지문 확인
	fingerprints := verifier.GetTrustedKeyFingerprints()
	assert.Equal(t, len(fingerprints), 1)
	assert.NotEqual(t, fingerprints[0], "")
}

// TestLoadTrustedKeysEmptyDirectory 빈 디렉토리 테스트
func TestLoadTrustedKeysEmptyDirectory(t *testing.T) {
	// 빈 임시 디렉토리 생성
	tempDir, err := os.MkdirTemp("", "apk_keys_empty_test")
	assert.Equal(t, err, nil)
	defer os.RemoveAll(tempDir)

	verifier := NewSignatureVerifier()
	err = verifier.LoadTrustedKeys(tempDir)
	assert.Equal(t, err, nil)
	assert.Equal(t, verifier.GetTrustedKeyCount(), 0)
}

// TestLoadTrustedKeysNonExistentDirectory 존재하지 않는 디렉토리 테스트
func TestLoadTrustedKeysNonExistentDirectory(t *testing.T) {
	verifier := NewSignatureVerifier()
	err := verifier.LoadTrustedKeys("/non/existent/directory")
	assert.Equal(t, err, nil) // 경고만 하고 에러는 없어야 함
	assert.Equal(t, verifier.GetTrustedKeyCount(), 0)
}

// TestCalculateFingerprint 지문 계산 테스트
func TestCalculateFingerprint(t *testing.T) {
	// 테스트용 RSA 키 생성
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.Equal(t, err, nil)

	verifier := NewSignatureVerifier()
	fingerprint1 := verifier.calculateFingerprint(&privateKey.PublicKey)
	fingerprint2 := verifier.calculateFingerprint(&privateKey.PublicKey)

	// 같은 키에 대해 같은 지문이 생성되어야 함
	assert.Equal(t, fingerprint1, fingerprint2)
	assert.Equal(t, len(fingerprint1), 16) // 16자 지문
}

// TestVerifyApkSignatureFileNotFound 파일이 없는 경우 테스트
func TestVerifyApkSignatureFileNotFound(t *testing.T) {
	verifier := NewSignatureVerifier()
	result := verifier.VerifyApkSignature("/non/existent/file.apk")

	assert.Equal(t, result.IsValid, false)
	assert.NotEqual(t, result.Error, "")
}

// TestFindSignatureFile 서명 파일 찾기 테스트
func TestFindSignatureFile(t *testing.T) {
	// 임시 디렉토리 생성
	tempDir, err := os.MkdirTemp("", "apk_signature_test")
	assert.Equal(t, err, nil)
	defer os.RemoveAll(tempDir)

	// 테스트 APK 파일 생성
	apkFile := filepath.Join(tempDir, "test.apk")
	err = os.WriteFile(apkFile, []byte("fake apk content"), 0644)
	assert.Equal(t, err, nil)

	// 서명 파일 생성
	signatureFile := filepath.Join(tempDir, "test.apk.SIGN.RSA.alpine")
	err = os.WriteFile(signatureFile, []byte("fake signature"), 0644)
	assert.Equal(t, err, nil)

	verifier := NewSignatureVerifier()
	foundFile := verifier.findSignatureFile(apkFile)
	assert.Equal(t, foundFile, signatureFile)
}

// TestCalculateFileHash 파일 해시 계산 테스트
func TestCalculateFileHash(t *testing.T) {
	// 임시 파일 생성
	tempFile, err := os.CreateTemp("", "hash_test")
	assert.Equal(t, err, nil)
	defer os.Remove(tempFile.Name())

	testContent := []byte("test content for hashing")
	_, err = tempFile.Write(testContent)
	assert.Equal(t, err, nil)
	tempFile.Close()

	verifier := NewSignatureVerifier()
	hash, err := verifier.calculateFileHash(tempFile.Name())
	assert.Equal(t, err, nil)

	// 예상 해시와 비교
	expectedHash := sha256.Sum256(testContent)
	assert.Equal(t, hash, expectedHash[:])
}

// TestVerificationResult 검증 결과 구조체 테스트
func TestVerificationResult(t *testing.T) {
	result := &VerificationResult{
		IsValid:        true,
		SignatureFile:  "/path/to/signature",
		KeyFingerprint: "abcd1234",
		Details:        []string{"test detail"},
	}

	assert.Equal(t, result.IsValid, true)
	assert.Equal(t, result.SignatureFile, "/path/to/signature")
	assert.Equal(t, result.KeyFingerprint, "abcd1234")
	assert.Equal(t, len(result.Details), 1)
	assert.Equal(t, result.Details[0], "test detail")
}
