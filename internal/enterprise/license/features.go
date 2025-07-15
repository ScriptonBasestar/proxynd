// Package license defines available enterprise features
package license

// Feature constants - 모든 엔터프라이즈 기능 정의
const (
	// 인증 기능
	FeatureLDAPAuth      = "ldap_auth"
	FeatureSAMLAuth      = "saml_auth"
	FeatureOAuth2        = "oauth2_advanced"
	FeatureRBAC          = "rbac"
	FeatureMFA           = "mfa"

	// 고급 프록시 기능
	FeatureMultiDatacenter   = "multi_datacenter"
	FeatureGeoRouting        = "geo_routing"
	FeatureSmartCaching      = "smart_caching"
	FeaturePredictiveCache   = "predictive_cache"
	FeatureBandwidthControl  = "bandwidth_control"

	// 보안 기능
	FeatureVulnerabilityScanning = "vulnerability_scanning"
	FeatureLicenseScanning       = "license_scanning"
	FeatureMalwareScanning       = "malware_scanning"
	FeatureSecurityPolicies      = "security_policies"
	FeatureAuditLog              = "audit_log"

	// 분석 및 모니터링
	FeatureAdvancedAnalytics = "advanced_analytics"
	FeatureCustomDashboards  = "custom_dashboards"
	FeatureReporting         = "reporting"
	FeatureAlerting          = "advanced_alerting"
	FeatureSLAMonitoring     = "sla_monitoring"

	// API 및 자동화
	FeatureGraphQLAPI        = "graphql_api"
	FeatureWebhooks          = "webhooks_advanced"
	FeatureTerraformProvider = "terraform_provider"
	FeatureKubernetesOperator = "kubernetes_operator"

	// 컴플라이언스
	FeatureGDPRCompliance = "gdpr_compliance"
	FeatureSOXCompliance  = "sox_compliance"
	FeatureHIPAACompliance = "hipaa_compliance"
	FeatureDataRetention  = "data_retention"

	// 지원
	FeaturePrioritySupport = "priority_support"
	Feature247Support      = "24x7_support"
	FeatureDedicatedCSM    = "dedicated_csm"
	FeatureSLA             = "sla_guarantee"
)

// FeatureSets 라이센스 타입별 기능 세트
var FeatureSets = map[string][]string{
	"trial": {
		FeatureLDAPAuth,
		FeatureSmartCaching,
		FeatureAdvancedAnalytics,
		FeatureWebhooks,
	},
	"starter": {
		FeatureLDAPAuth,
		FeatureSAMLAuth,
		FeatureSmartCaching,
		FeatureAdvancedAnalytics,
		FeatureWebhooks,
		FeatureAuditLog,
		FeaturePrioritySupport,
	},
	"professional": {
		// Starter의 모든 기능 포함
		FeatureLDAPAuth,
		FeatureSAMLAuth,
		FeatureOAuth2,
		FeatureRBAC,
		FeatureSmartCaching,
		FeaturePredictiveCache,
		FeatureVulnerabilityScanning,
		FeatureLicenseScanning,
		FeatureSecurityPolicies,
		FeatureAdvancedAnalytics,
		FeatureCustomDashboards,
		FeatureReporting,
		FeatureWebhooks,
		FeatureGraphQLAPI,
		FeatureAuditLog,
		FeatureGDPRCompliance,
		FeaturePrioritySupport,
		FeatureSLA,
	},
	"enterprise": {
		// Professional의 모든 기능 포함
		FeatureLDAPAuth,
		FeatureSAMLAuth,
		FeatureOAuth2,
		FeatureRBAC,
		FeatureMFA,
		FeatureMultiDatacenter,
		FeatureGeoRouting,
		FeatureSmartCaching,
		FeaturePredictiveCache,
		FeatureBandwidthControl,
		FeatureVulnerabilityScanning,
		FeatureLicenseScanning,
		FeatureMalwareScanning,
		FeatureSecurityPolicies,
		FeatureAdvancedAnalytics,
		FeatureCustomDashboards,
		FeatureReporting,
		FeatureAlerting,
		FeatureSLAMonitoring,
		FeatureWebhooks,
		FeatureGraphQLAPI,
		FeatureTerraformProvider,
		FeatureAuditLog,
		FeatureGDPRCompliance,
		FeatureSOXCompliance,
		FeatureDataRetention,
		Feature247Support,
		FeatureSLA,
	},
	"ultimate": {
		"*", // 모든 기능
	},
}

// FeatureInfo 기능 정보
type FeatureInfo struct {
	ID          string
	Name        string
	Description string
	Category    string
	MinLicense  string // 최소 필요 라이센스
}

// FeatureCatalog 모든 기능 카탈로그
var FeatureCatalog = map[string]FeatureInfo{
	FeatureLDAPAuth: {
		ID:          FeatureLDAPAuth,
		Name:        "LDAP Authentication",
		Description: "기업 LDAP/Active Directory 통합",
		Category:    "인증",
		MinLicense:  "starter",
	},
	FeatureMultiDatacenter: {
		ID:          FeatureMultiDatacenter,
		Name:        "Multi-Datacenter Replication",
		Description: "여러 데이터센터 간 자동 복제 및 동기화",
		Category:    "인프라",
		MinLicense:  "enterprise",
	},
	FeatureVulnerabilityScanning: {
		ID:          FeatureVulnerabilityScanning,
		Name:        "Vulnerability Scanning",
		Description: "패키지 취약점 자동 스캔 및 알림",
		Category:    "보안",
		MinLicense:  "professional",
	},
	// ... 더 많은 기능 정의
}

// GetFeatureInfo 기능 정보 조회
func GetFeatureInfo(featureID string) (FeatureInfo, bool) {
	info, exists := FeatureCatalog[featureID]
	return info, exists
}

// GetFeaturesForLicense 라이센스 타입별 기능 목록
func GetFeaturesForLicense(licenseType string) []string {
	features, exists := FeatureSets[licenseType]
	if !exists {
		return []string{}
	}
	return features
}

// IsFeatureInLicense 특정 라이센스에 기능 포함 여부
func IsFeatureInLicense(feature, licenseType string) bool {
	features := GetFeaturesForLicense(licenseType)
	for _, f := range features {
		if f == feature || f == "*" {
			return true
		}
	}
	return false
}