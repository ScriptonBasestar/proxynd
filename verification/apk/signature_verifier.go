package apk

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"proxynd/logging"
)

// SignatureVerifier APK 서명 검증기
type SignatureVerifier struct {
	trustedKeys map[string]*rsa.PublicKey // 신뢰할 수 있는 공개키 저장
	logger      logging.Logger
}

// VerificationResult 검증 결과
type VerificationResult struct {
	IsValid        bool     `json:"is_valid"`
	SignatureFile  string   `json:"signature_file,omitempty"`
	KeyFingerprint string   `json:"key_fingerprint,omitempty"`
	Error          string   `json:"error,omitempty"`
	Details        []string `json:"details,omitempty"`
}

// NewSignatureVerifier 새로운 서명 검증기 생성
func NewSignatureVerifier() *SignatureVerifier {
	return &SignatureVerifier{
		trustedKeys: make(map[string]*rsa.PublicKey),
		logger:      logging.GetLogger(),
	}
}

// LoadTrustedKeys 신뢰할 수 있는 공개키 로드
func (sv *SignatureVerifier) LoadTrustedKeys(keyDir string) error {
	if keyDir == "" {
		sv.logger.Warn("키 디렉토리가 지정되지 않았습니다")
		return nil
	}

	// 키 디렉토리 존재 확인
	if _, err := os.Stat(keyDir); os.IsNotExist(err) {
		sv.logger.Warn("키 디렉토리가 존재하지 않습니다", logging.F("directory", keyDir))
		return nil
	}

	sv.logger.Info("신뢰할 수 있는 키 로드 중", logging.F("directory", keyDir))

	// 키 파일들 로드
	err := filepath.Walk(keyDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// .pub 또는 .pem 확장자를 가진 파일만 처리
		if !info.IsDir() && (strings.HasSuffix(path, ".pub") || strings.HasSuffix(path, ".pem")) {
			if err := sv.loadPublicKey(path); err != nil {
				sv.logger.Error("공개키 로드 실패",
					logging.F("file", path),
					logging.F("error", err.Error()))
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("키 디렉토리 탐색 실패: %v", err)
	}

	sv.logger.Info("신뢰할 수 있는 키 로드 완료",
		logging.F("count", len(sv.trustedKeys)))

	return nil
}

// loadPublicKey 개별 공개키 파일 로드
func (sv *SignatureVerifier) loadPublicKey(keyPath string) error {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("키 파일 읽기 실패: %v", err)
	}

	// PEM 블록이 있는지 먼저 확인
	if block, _ := pem.Decode(keyData); block != nil {
		keyData = block.Bytes
	}

	// PKCS#1 형식 파싱 시도 (RSA 공개키)
	if publicKey, err := x509.ParsePKCS1PublicKey(keyData); err == nil {
		fingerprint := sv.calculateFingerprint(publicKey)
		sv.trustedKeys[fingerprint] = publicKey
		sv.logger.Debug("RSA 공개키 로드됨",
			logging.F("file", keyPath),
			logging.F("fingerprint", fingerprint))
		return nil
	}

	// PKIX 형식 파싱 시도 (일반적인 공개키)
	if publicKeyInterface, err := x509.ParsePKIXPublicKey(keyData); err == nil {
		if publicKey, ok := publicKeyInterface.(*rsa.PublicKey); ok {
			fingerprint := sv.calculateFingerprint(publicKey)
			sv.trustedKeys[fingerprint] = publicKey
			sv.logger.Debug("PKIX 공개키 로드됨",
				logging.F("file", keyPath),
				logging.F("fingerprint", fingerprint))
			return nil
		}
	}

	return fmt.Errorf("지원되지 않는 공개키 형식: %s", keyPath)
}

// calculateFingerprint 공개키의 지문 계산
func (sv *SignatureVerifier) calculateFingerprint(publicKey *rsa.PublicKey) string {
	keyBytes, _ := x509.MarshalPKIXPublicKey(publicKey)
	hash := sha256.Sum256(keyBytes)
	return hex.EncodeToString(hash[:])[:16] // 처음 16자만 사용
}

// VerifyApkSignature APK 파일의 서명 검증
func (sv *SignatureVerifier) VerifyApkSignature(apkPath string) *VerificationResult {
	result := &VerificationResult{
		IsValid: false,
		Details: []string{},
	}

	// APK 파일 존재 확인
	if _, err := os.Stat(apkPath); os.IsNotExist(err) {
		result.Error = fmt.Sprintf("APK 파일이 존재하지 않습니다: %s", apkPath)
		return result
	}

	// 서명 파일 찾기
	signatureFile := sv.findSignatureFile(apkPath)
	if signatureFile == "" {
		result.Error = "서명 파일을 찾을 수 없습니다"
		result.Details = append(result.Details, "지원되는 서명 파일: .SIGN.RSA.*, .rsa")
		return result
	}

	result.SignatureFile = signatureFile
	result.Details = append(result.Details, fmt.Sprintf("서명 파일 발견: %s", signatureFile))

	// 서명 검증 수행
	if err := sv.verifySignatureFile(apkPath, signatureFile, result); err != nil {
		result.Error = err.Error()
		return result
	}

	return result
}

// findSignatureFile APK 파일과 연관된 서명 파일 찾기
func (sv *SignatureVerifier) findSignatureFile(apkPath string) string {
	baseDir := filepath.Dir(apkPath)
	apkName := filepath.Base(apkPath)

	// .SIGN.RSA.* 패턴 확인
	patterns := []string{
		fmt.Sprintf("%s.SIGN.RSA.*", apkName),
		fmt.Sprintf("%s.rsa", apkName),
		strings.Replace(apkName, ".apk", ".SIGN.RSA.alpine", 1),
		strings.Replace(apkName, ".apk", ".rsa", 1),
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(filepath.Join(baseDir, pattern))
		if len(matches) > 0 {
			return matches[0]
		}
	}

	// 동일 디렉토리에서 .SIGN.RSA.* 패턴 검색
	files, _ := os.ReadDir(baseDir)
	for _, file := range files {
		if strings.Contains(file.Name(), ".SIGN.RSA.") {
			return filepath.Join(baseDir, file.Name())
		}
	}

	return ""
}

// verifySignatureFile 실제 서명 검증 수행
func (sv *SignatureVerifier) verifySignatureFile(apkPath, signatureFile string, result *VerificationResult) error {
	// 서명 파일 읽기
	signatureData, err := os.ReadFile(signatureFile)
	if err != nil {
		return fmt.Errorf("서명 파일 읽기 실패: %v", err)
	}

	// APK 파일 해시 계산
	apkHash, err := sv.calculateFileHash(apkPath)
	if err != nil {
		return fmt.Errorf("APK 파일 해시 계산 실패: %v", err)
	}

	result.Details = append(result.Details, fmt.Sprintf("APK 해시: %x", apkHash))

	// 신뢰할 수 있는 키들로 검증 시도
	for fingerprint, publicKey := range sv.trustedKeys {
		if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, apkHash, signatureData); err == nil {
			result.IsValid = true
			result.KeyFingerprint = fingerprint
			result.Details = append(result.Details, fmt.Sprintf("서명 검증 성공 (키: %s)", fingerprint))

			sv.logger.Info("APK 서명 검증 성공",
				logging.F("apk", apkPath),
				logging.F("key_fingerprint", fingerprint))
			return nil
		}
	}

	// 모든 키로 검증 실패
	if len(sv.trustedKeys) == 0 {
		result.Details = append(result.Details, "신뢰할 수 있는 키가 없습니다")
		sv.logger.Warn("신뢰할 수 있는 키가 없어 서명 검증을 건너뜁니다",
			logging.F("apk", apkPath))
		// 키가 없는 경우는 경고만 하고 통과
		result.IsValid = true
		return nil
	}

	return fmt.Errorf("서명 검증 실패: 신뢰할 수 있는 키로 검증되지 않음 (시도한 키: %d개)", len(sv.trustedKeys))
}

// calculateFileHash 파일의 SHA256 해시 계산
func (sv *SignatureVerifier) calculateFileHash(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, err
	}

	return hasher.Sum(nil), nil
}

// GetTrustedKeyCount 로드된 신뢰할 수 있는 키 개수 반환
func (sv *SignatureVerifier) GetTrustedKeyCount() int {
	return len(sv.trustedKeys)
}

// GetTrustedKeyFingerprints 로드된 키들의 지문 목록 반환
func (sv *SignatureVerifier) GetTrustedKeyFingerprints() []string {
	fingerprints := make([]string, 0, len(sv.trustedKeys))
	for fp := range sv.trustedKeys {
		fingerprints = append(fingerprints, fp)
	}
	return fingerprints
}
