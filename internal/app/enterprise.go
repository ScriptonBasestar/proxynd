// Package app provides enterprise feature integration
package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/enterprise/license"
)

// EnterpriseFeatures 엔터프라이즈 기능 관리
type EnterpriseFeatures struct {
	validator   *license.Validator
	featureGate *license.FeatureGate
	enabled     bool
}

// InitEnterpriseFeatures 엔터프라이즈 기능 초기화
func InitEnterpriseFeatures(configDir string) *EnterpriseFeatures {
	ef := &EnterpriseFeatures{
		enabled: false,
	}

	// 라이센스 검증기 생성
	validator, err := license.NewValidator()
	if err != nil {
		fmt.Printf("Enterprise features disabled: %v\n", err)
		return ef
	}

	// 라이센스 파일 확인
	licensePath := filepath.Join(configDir, "license.json")
	if _, err := os.Stat(licensePath); os.IsNotExist(err) {
		licensePath = "/etc/proxynd/license.json"
	}

	// 환경 변수에서도 확인
	if envPath := os.Getenv("PROXYND_LICENSE_FILE"); envPath != "" {
		licensePath = envPath
	}

	// 라이센스 로드
	if licenseData, err := os.ReadFile(licensePath); err == nil {
		if err := validator.LoadLicense(licenseData); err != nil {
			fmt.Printf("Invalid license: %v\n", err)
		} else {
			ef.validator = validator
			ef.featureGate = license.NewFeatureGate(validator)
			ef.enabled = true

			info := validator.GetLicenseInfo()
			fmt.Printf("Enterprise license loaded: %s (%s)\n", info["company"], info["type"])
		}
	}

	return ef
}

// IsEnabled 엔터프라이즈 기능 활성화 여부
func (ef *EnterpriseFeatures) IsEnabled() bool {
	return ef.enabled && ef.validator != nil && ef.validator.IsValid()
}

// HasFeature 특정 기능 활성화 여부
func (ef *EnterpriseFeatures) HasFeature(feature string) bool {
	if !ef.IsEnabled() {
		return false
	}
	return ef.featureGate.IsEnabled(feature)
}

// RequireFeature 기능 필수 확인 미들웨어
func (ef *EnterpriseFeatures) RequireFeature(feature string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !ef.HasFeature(feature) {
			return c.Status(402).JSON(fiber.Map{
				"error":       "Feature not licensed",
				"feature":     feature,
				"message":     "This feature requires an enterprise license",
				"upgrade_url": "https://proxynd.io/pricing",
			})
		}
		return c.Next()
	}
}

// GetLicenseInfo 라이센스 정보 조회
func (ef *EnterpriseFeatures) GetLicenseInfo() map[string]interface{} {
	if !ef.IsEnabled() {
		return map[string]interface{}{
			"licensed": false,
			"type":     "community",
			"features": []string{},
		}
	}

	info := ef.validator.GetLicenseInfo()
	info["licensed"] = true

	// 활성화된 기능 목록 추가
	if licenseType, ok := info["type"].(string); ok {
		info["available_features"] = license.GetFeaturesForLicense(licenseType)
	}

	return info
}

// RegisterEnterpriseRoutes 엔터프라이즈 API 라우트 등록
func (ef *EnterpriseFeatures) RegisterEnterpriseRoutes(app *fiber.App) {
	// 라이센스 정보 조회
	app.Get("/api/v1/license", func(c *fiber.Ctx) error {
		return c.JSON(ef.GetLicenseInfo())
	})

	// 기능 카탈로그 조회
	app.Get("/api/v1/features", func(c *fiber.Ctx) error {
		features := make([]fiber.Map, 0)

		for id, info := range license.FeatureCatalog {
			features = append(features, fiber.Map{
				"id":          id,
				"name":        info.Name,
				"description": info.Description,
				"category":    info.Category,
				"min_license": info.MinLicense,
				"enabled":     ef.HasFeature(id),
			})
		}

		return c.JSON(fiber.Map{
			"features":     features,
			"license_type": ef.GetLicenseInfo()["type"],
		})
	})

	// GraphQL API (엔터프라이즈 기능)
	if ef.HasFeature(license.FeatureGraphQLAPI) {
		app.Post("/graphql", ef.RequireFeature(license.FeatureGraphQLAPI), func(c *fiber.Ctx) error {
			// GraphQL 핸들러
			return c.JSON(fiber.Map{
				"message": "GraphQL endpoint (enterprise feature)",
			})
		})
	}
}

// CheckFeatureUsage 기능 사용량 체크 (로깅/분석용)
func (ef *EnterpriseFeatures) CheckFeatureUsage(feature string) {
	if ef.HasFeature(feature) {
		// 기능 사용 통계 기록
		// 추후 사용량 기반 과금이나 분석에 활용
		fmt.Printf("Enterprise feature used: %s\n", feature)
	}
}
