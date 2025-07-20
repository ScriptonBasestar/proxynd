// Package license defines available enterprise features
package license

// Feature constants - 모든 엔터프라이즈 기능 정의
const (
	// 인증 기능
	// FeatureLDAPAuth is a const that feature l d a p auth
	// FeatureSAMLAuth is a const that feature s a m l auth
	// FeatureOAuth2 is a const that feature o auth2
	// FeatureRBAC is a const that feature r b a c
	// FeatureMFA is a const that feature m f a
	FeatureLDAPAuth = "ldap_auth"
	FeatureSAMLAuth = "saml_auth"
	// FeatureMultiDatacenter is a const that feature multi datacenter
	// FeatureGeoRouting is a const that feature geo routing
	// FeatureSmartCaching is a const that feature smart caching
	// FeaturePredictiveCache is a const that feature predictive cache
	// FeatureBandwidthControl is a const that feature bandwidth control
	FeatureOAuth2 = "oauth2_advanced"
	FeatureRBAC   = "rbac"
	// FeatureVulnerabilityScanning is a const that feature vulnerability scanning
	// FeatureLicenseScanning is a const that feature license scanning
	// FeatureMalwareScanning is a const that feature malware scanning
	// FeatureSecurityPolicies is a const that feature security policies
	// FeatureAuditLog is a const that feature audit log
	FeatureMFA = "mfa"

	// FeatureAdvancedAnalytics is a const that feature advanced analytics
	// FeatureCustomDashboards is a const that feature custom dashboards
	// FeatureReporting is a const that feature reporting
	// FeatureAlerting is a const that feature alerting
	// FeatureSLAMonitoring is a const that feature s l a monitoring
	// 고급 프록시 기능
	FeatureMultiDatacenter = "multi_datacenter"
	// FeatureGraphQLAPI is a const that feature graph q l a p i
	// FeatureWebhooks is a const that feature webhooks
	// FeatureTerraformProvider is a const that feature terraform provider
	// FeatureKubernetesOperator is a const that feature kubernetes operator
	FeatureGeoRouting   = "geo_routing"
	FeatureSmartCaching = "smart_caching"
	// FeatureGDPRCompliance is a const that feature g d p r compliance
	// FeatureSOXCompliance is a const that feature s o x compliance
	// FeatureHIPAACompliance is a const that feature h i p a a compliance
	// FeatureDataRetention is a const that feature data retention
	FeaturePredictiveCache  = "predictive_cache"
	FeatureBandwidthControl = "bandwidth_control"
	// FeaturePrioritySupport is a const that feature priority support
	// Feature247Support is a const that feature247 support
	// FeatureDedicatedCSM is a const that feature dedicated c s m
	// FeatureSLA is a const that feature s l a

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
	FeatureGraphQLAPI         = "graphql_api"
	FeatureWebhooks           = "webhooks_advanced"
	FeatureTerraformProvider  = "terraform_provider"
	FeatureKubernetesOperator = "kubernetes_operator"

	// 컴플라이언스
	FeatureGDPRCompliance  = "gdpr_compliance"
	FeatureSOXCompliance   = "sox_compliance"
	FeatureHIPAACompliance = "hipaa_compliance"
	FeatureDataRetention   = "data_retention"

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
