package apk

import (
	"context"
	"fmt"
	"os"
	"strings"

	"proxynd/internal/domain/apk"
	"proxynd/logging"
	apkVerification "proxynd/verification/apk"
)

type signatureVerifierImpl struct {
	config   apk.ProxyConfig
	logger   logging.Logger
	verifier *apkVerification.SignatureVerifier
}

func NewSignatureVerifier(config apk.ProxyConfig, logger logging.Logger) apk.SignatureVerifier {
	return &signatureVerifierImpl{
		config:   config,
		logger:   logger,
		verifier: apkVerification.NewSignatureVerifier(),
	}
}

func (s *signatureVerifierImpl) VerifyPackage(ctx context.Context, packagePath string) (*apk.SignatureInfo, error) {
	if !s.config.GetVerificationEnabled() {
		return &apk.SignatureInfo{IsValid: true}, nil
	}

	result := s.verifier.VerifyApkSignature(packagePath)

	signatureInfo := &apk.SignatureInfo{
		IsValid:        result.IsValid,
		SignatureFile:  result.SignatureFile,
		KeyFingerprint: result.KeyFingerprint,
		Error:          result.Error,
	}

	s.logger.Debug("APK 서명 검증 완료",
		logging.F("package", packagePath),
		logging.F("valid", result.IsValid),
		logging.F("key", result.KeyFingerprint))

	return signatureInfo, nil
}

func (s *signatureVerifierImpl) LoadTrustedKeys(ctx context.Context, keyDirectory string) error {
	if keyDirectory == "" {
		keyDirectory = s.config.GetVerificationKeyDirectory()
	}

	if _, err := os.Stat(keyDirectory); os.IsNotExist(err) {
		return fmt.Errorf("key directory not found: %s", keyDirectory)
	}

	if err := s.verifier.LoadTrustedKeys(keyDirectory); err != nil {
		s.logger.Error("APK 신뢰 키 로드 실패", logging.F("directory", keyDirectory), logging.F("error", err.Error()))
		return fmt.Errorf("failed to load trusted keys: %w", err)
	}

	s.logger.Info("APK 신뢰 키 로드 완료", logging.F("directory", keyDirectory))
	return nil
}

func (s *signatureVerifierImpl) ValidateSignature(ctx context.Context, signaturePath string) (*apk.SignatureInfo, error) {
	if !s.IsSignatureFile(signaturePath) {
		return nil, fmt.Errorf("not a signature file: %s", signaturePath)
	}

	// 간단한 서명 파일 유효성 검증
	if _, err := os.Stat(signaturePath); os.IsNotExist(err) {
		return &apk.SignatureInfo{
			IsValid: false,
			Error:   "signature file not found",
		}, nil
	}

	return &apk.SignatureInfo{IsValid: true}, nil
}

func (s *signatureVerifierImpl) GetTrustedKeys(ctx context.Context) ([]string, error) {
	keyDir := s.config.GetVerificationKeyDirectory()

	entries, err := os.ReadDir(keyDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read key directory: %w", err)
	}

	var keys []string
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".pub") || strings.HasSuffix(entry.Name(), ".rsa")) {
			keys = append(keys, entry.Name())
		}
	}

	return keys, nil
}

func (s *signatureVerifierImpl) IsPackageFile(filename string) bool {
	return strings.HasSuffix(filename, ".apk")
}

func (s *signatureVerifierImpl) IsSignatureFile(filename string) bool {
	return strings.HasSuffix(filename, ".asc") || strings.HasSuffix(filename, ".rsa")
}
