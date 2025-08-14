// Package health provides security system health checking functionality
package health

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"proxynd/internal/config"
)

// SecuritySystemHealthChecker 보안 시스템 건강성 체커
type SecuritySystemHealthChecker struct {
	config *config.RootConfig
}

// NewSecuritySystemHealthChecker 새 보안 시스템 헬스체커 생성
func NewSecuritySystemHealthChecker(config *config.RootConfig) *SecuritySystemHealthChecker {
	return &SecuritySystemHealthChecker{
		config: config,
	}
}

// Name 체커 이름 반환
func (sshc *SecuritySystemHealthChecker) Name() string {
	return "security_system"
}

// Check 보안 시스템 전체 건강성 확인
func (sshc *SecuritySystemHealthChecker) Check(ctx context.Context) *CheckResult {
	start := time.Now()
	result := &CheckResult{
		Name:        sshc.Name(),
		Status:      StatusHealthy,
		LastChecked: time.Now(),
		Details:     make(map[string]interface{}),
	}

	// TLS 설정 체크
	tlsResult := sshc.checkTLSConfiguration()
	result.Details["tls"] = tlsResult

	// 인증 시스템 체크
	authResult := sshc.checkAuthenticationSystem()
	result.Details["authentication"] = authResult

	// 접근 제어 체크
	accessControlResult := sshc.checkAccessControl()
	result.Details["access_control"] = accessControlResult

	// 패키지 필터 체크
	packageFilterResult := sshc.checkPackageFilter()
	result.Details["package_filter"] = packageFilterResult

	// 해시 검증 체크
	hashVerificationResult := sshc.checkHashVerification()
	result.Details["hash_verification"] = hashVerificationResult

	// 전체 상태 결정
	allResults := []map[string]interface{}{
		tlsResult, authResult, accessControlResult,
		packageFilterResult, hashVerificationResult,
	}

	var healthyCount, degradedCount, unhealthyCount int
	var criticalIssues []string
	var warnings []string

	for _, res := range allResults {
		switch res["status"].(string) {
		case string(StatusHealthy):
			healthyCount++
		case string(StatusDegraded):
			degradedCount++
			if warning, ok := res["message"].(string); ok {
				warnings = append(warnings, warning)
			}
		case string(StatusUnhealthy):
			unhealthyCount++
			if issue, ok := res["message"].(string); ok {
				criticalIssues = append(criticalIssues, issue)
			}
		}
	}

	result.Details["summary"] = map[string]interface{}{
		"total_checks":     len(allResults),
		"healthy_checks":   healthyCount,
		"degraded_checks":  degradedCount,
		"unhealthy_checks": unhealthyCount,
		"critical_issues":  criticalIssues,
		"warnings":         warnings,
	}

	// 상태 결정
	if unhealthyCount > 0 {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("보안 시스템에 심각한 문제가 있습니다 (%d개 실패)", unhealthyCount)
	} else if degradedCount > 0 {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("보안 시스템에 개선이 필요합니다 (%d개 경고)", degradedCount)
	} else {
		result.Message = "보안 시스템이 정상적으로 구성되어 있습니다"
	}

	result.Duration = time.Since(start)
	return result
}

// checkTLSConfiguration TLS 설정 확인
func (sshc *SecuritySystemHealthChecker) checkTLSConfiguration() map[string]interface{} {
	result := map[string]interface{}{
		"status":  string(StatusHealthy),
		"enabled": sshc.config.Server.TLS.Enabled,
		"details": make(map[string]interface{}),
	}

	if !sshc.config.Server.TLS.Enabled {
		result["status"] = string(StatusDegraded)
		result["message"] = "TLS가 비활성화되어 있습니다 (프로덕션 환경에서는 권장하지 않음)"
		result["recommendation"] = "TLS를 활성화하여 통신을 암호화하세요"
		return result
	}

	// TLS 인증서 파일 확인
	certFile := sshc.config.Server.TLS.CertFile
	keyFile := sshc.config.Server.TLS.KeyFile

	result["details"].(map[string]interface{})["cert_file"] = certFile
	result["details"].(map[string]interface{})["key_file"] = keyFile
	result["details"].(map[string]interface{})["min_version"] = sshc.config.Server.TLS.MinVersion

	// 인증서 파일 존재 확인
	if certFile == "" || keyFile == "" {
		result["status"] = string(StatusUnhealthy)
		result["message"] = "TLS가 활성화되었지만 인증서 파일이 설정되지 않았습니다"
		result["error"] = "cert_file 또는 key_file이 비어있음"
		return result
	}

	// 파일 접근 확인
	certInfo, certErr := os.Stat(certFile)
	keyInfo, keyErr := os.Stat(keyFile)

	if certErr != nil || keyErr != nil {
		result["status"] = string(StatusUnhealthy)
		result["message"] = "TLS 인증서 파일에 접근할 수 없습니다"
		if certErr != nil {
			result["cert_error"] = certErr.Error()
		}
		if keyErr != nil {
			result["key_error"] = keyErr.Error()
		}
		return result
	}

	// 인증서 유효성 확인
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		result["status"] = string(StatusUnhealthy)
		result["message"] = fmt.Sprintf("TLS 인증서 로드 실패: %v", err)
		result["error"] = err.Error()
		return result
	}

	// 인증서 정보 추가
	if len(cert.Certificate) > 0 {
		// 인증서 만료일 확인 등은 x509 파싱이 필요하므로 기본 정보만 제공
		result["details"].(map[string]interface{})["cert_loaded"] = true
		result["details"].(map[string]interface{})["cert_mod_time"] = certInfo.ModTime()
		result["details"].(map[string]interface{})["key_mod_time"] = keyInfo.ModTime()
	}

	// TLS 버전 보안성 확인
	minVersion := sshc.config.Server.TLS.MinVersion
	if minVersion == "" {
		minVersion = "TLS1.2" // 기본값
	}

	switch minVersion {
	case "TLS1.3":
		result["details"].(map[string]interface{})["security_level"] = "최고"
	case "TLS1.2":
		result["details"].(map[string]interface{})["security_level"] = "양호"
		result["status"] = string(StatusDegraded)
		result["message"] = "TLS 1.3 사용을 권장합니다"
		result["recommendation"] = "min_version을 TLS1.3으로 업그레이드하세요"
	default:
		result["status"] = string(StatusUnhealthy)
		result["message"] = fmt.Sprintf("안전하지 않은 TLS 버전: %s", minVersion)
		result["error"] = "TLS 1.2 이상이 필요합니다"
		return result
	}

	if result["status"] == string(StatusHealthy) {
		result["message"] = "TLS 설정이 안전하게 구성되어 있습니다"
	}

	return result
}

// checkAuthenticationSystem 인증 시스템 확인
func (sshc *SecuritySystemHealthChecker) checkAuthenticationSystem() map[string]interface{} {
	result := map[string]interface{}{
		"status":  string(StatusHealthy),
		"details": make(map[string]interface{}),
	}

	authConfig := sshc.config.Security.Authentication
	result["details"].(map[string]interface{})["auth_config"] = map[string]interface{}{
		"basic_auth_enabled": authConfig.BasicAuth != nil &&
			authConfig.BasicAuth.Enabled != nil &&
			*authConfig.BasicAuth.Enabled,
		"oauth2_enabled": authConfig.OAuth2 != nil && authConfig.OAuth2.Enabled,
		"jwt_enabled":    false, // JWT 지원은 현재 미구현
	}

	// 기본 인증 확인
	if authConfig.BasicAuth != nil && authConfig.BasicAuth.Enabled != nil && *authConfig.BasicAuth.Enabled {
		basicAuthResult := sshc.checkBasicAuth(authConfig.BasicAuth)
		result["details"].(map[string]interface{})["basic_auth"] = basicAuthResult
		if basicAuthResult["status"] == string(StatusUnhealthy) {
			result["status"] = string(StatusUnhealthy)
			result["message"] = "기본 인증 설정에 문제가 있습니다"
			return result
		}
	}

	// OAuth2 확인
	if authConfig.OAuth2 != nil && authConfig.OAuth2.Enabled {
		oauth2Result := sshc.checkOAuth2(authConfig.OAuth2)
		result["details"].(map[string]interface{})["oauth2"] = oauth2Result

		if oauth2Result["status"] == string(StatusUnhealthy) {
			result["status"] = string(StatusUnhealthy)
			result["message"] = "OAuth2 설정에 문제가 있습니다"
			return result
		}
	}

	// (JWT 체크는 미구현 상태이며, AuthenticationConfig 확장 시 구현 예정)

	// 전체적으로 인증이 비활성화된 경우
	noAuthEnabled := (authConfig.BasicAuth == nil ||
		authConfig.BasicAuth.Enabled == nil ||
		!*authConfig.BasicAuth.Enabled) &&
		(authConfig.OAuth2 == nil || !authConfig.OAuth2.Enabled)

	if noAuthEnabled {
		result["status"] = string(StatusDegraded)
		result["message"] = "인증이 전혀 활성화되지 않았습니다 (보안 위험)"
		result["recommendation"] = "최소한 하나의 인증 방식을 활성화하세요"
		return result
	}

	result["message"] = "인증 시스템이 적절히 구성되어 있습니다"
	return result
}

// checkBasicAuth 기본 인증 확인
func (sshc *SecuritySystemHealthChecker) checkBasicAuth(basicAuth interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"status": string(StatusHealthy),
	}

	// 실제 BasicAuth 구조체의 필드를 확인해야 하지만,
	// 현재는 기본적인 활성화 상태만 확인
	result["message"] = "기본 인증이 활성화되어 있습니다"

	// TODO: 실제 사용자 수, 암호 정책 등 확인 필요
	result["note"] = "사용자 정보는 보안상 표시하지 않습니다"

	return result
}

// checkOAuth2 OAuth2 확인
func (sshc *SecuritySystemHealthChecker) checkOAuth2(oauth2 interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"status": string(StatusHealthy),
	}

	result["message"] = "OAuth2가 활성화되어 있습니다"
	result["note"] = "OAuth2 프로바이더 연결성은 별도 확인이 필요합니다"

	return result
}

// checkAccessControl 접근 제어 확인
func (sshc *SecuritySystemHealthChecker) checkAccessControl() map[string]interface{} {
	result := map[string]interface{}{
		"status":  string(StatusHealthy),
		"details": make(map[string]interface{}),
	}

	accessControl := sshc.config.Security.AccessControl

	// IP 화이트리스트 확인
	if accessControl.IPWhitelist.Enabled {
		ipResult := sshc.checkIPWhitelist(accessControl.IPWhitelist)
		result["details"].(map[string]interface{})["ip_whitelist"] = ipResult

		if ipResult["status"] == string(StatusUnhealthy) {
			result["status"] = string(StatusUnhealthy)
			result["message"] = "IP 화이트리스트 설정에 문제가 있습니다"
			return result
		}
	} else {
		result["details"].(map[string]interface{})["ip_whitelist"] = map[string]interface{}{
			"enabled": false,
			"message": "IP 화이트리스트가 비활성화되어 있습니다",
		}
	}

	// 권한 규칙 확인
	permissionResult := sshc.checkPermissions(accessControl.Permissions)
	result["details"].(map[string]interface{})["permissions"] = permissionResult

	result["message"] = "접근 제어 설정이 구성되어 있습니다"
	return result
}

// checkIPWhitelist IP 화이트리스트 확인
func (sshc *SecuritySystemHealthChecker) checkIPWhitelist(ipWhitelist config.IPWhitelistConfig) map[string]interface{} {
	result := map[string]interface{}{
		"status":  string(StatusHealthy),
		"enabled": ipWhitelist.Enabled,
		"details": make(map[string]interface{}),
	}

	if !ipWhitelist.Enabled {
		result["message"] = "IP 화이트리스트가 비활성화되어 있습니다"
		return result
	}

	// IP 주소 유효성 검사
	validIPs := 0
	invalidIPs := []string{}
	for _, ip := range ipWhitelist.IPs {
		if net.ParseIP(ip) != nil {
			validIPs++
		} else {
			invalidIPs = append(invalidIPs, ip)
		}
	}

	// CIDR 블록 유효성 검사
	validCIDRs := 0
	invalidCIDRs := []string{}
	for _, cidr := range ipWhitelist.CIDRs {
		if _, _, err := net.ParseCIDR(cidr); err == nil {
			validCIDRs++
		} else {
			invalidCIDRs = append(invalidCIDRs, cidr)
		}
	}

	result["details"].(map[string]interface{})["valid_ips"] = validIPs
	result["details"].(map[string]interface{})["valid_cidrs"] = validCIDRs
	result["details"].(map[string]interface{})["total_ips"] = len(ipWhitelist.IPs)
	result["details"].(map[string]interface{})["total_cidrs"] = len(ipWhitelist.CIDRs)

	if len(invalidIPs) > 0 || len(invalidCIDRs) > 0 {
		result["status"] = string(StatusUnhealthy)
		result["message"] = "일부 IP 화이트리스트 항목이 유효하지 않습니다"
		result["invalid_ips"] = invalidIPs
		result["invalid_cidrs"] = invalidCIDRs
		return result
	}

	if validIPs == 0 && validCIDRs == 0 {
		result["status"] = string(StatusDegraded)
		result["message"] = "IP 화이트리스트가 활성화되었지만 항목이 없습니다"
		result["recommendation"] = "허용할 IP 주소 또는 CIDR 블록을 추가하세요"
		return result
	}

	result["message"] = fmt.Sprintf("IP 화이트리스트가 정상적으로 구성되어 있습니다 (%d IPs, %d CIDRs)",
		validIPs, validCIDRs)
	return result
}

// checkPermissions 권한 규칙 확인
func (sshc *SecuritySystemHealthChecker) checkPermissions(permissions []config.PermissionRule) map[string]interface{} {
	result := map[string]interface{}{
		"status":  string(StatusHealthy),
		"details": make(map[string]interface{}),
	}

	if len(permissions) == 0 {
		result["status"] = string(StatusDegraded)
		result["message"] = "권한 규칙이 설정되지 않았습니다"
		result["recommendation"] = "사용자 또는 그룹별 권한 규칙을 설정하세요"
		return result
	}

	// 권한 규칙 분석
	userCount := 0
	groupCount := 0
	actionTypes := make(map[string]int)
	registryTypes := make(map[string]int)

	for _, rule := range permissions {
		if rule.User != "" {
			userCount++
		}
		groupCount += len(rule.Groups)

		for _, action := range rule.Actions {
			actionTypes[action]++
		}

		for _, registry := range rule.Registries {
			registryTypes[registry]++
		}
	}

	result["details"].(map[string]interface{})["total_rules"] = len(permissions)
	result["details"].(map[string]interface{})["user_rules"] = userCount
	result["details"].(map[string]interface{})["group_rules"] = groupCount
	result["details"].(map[string]interface{})["action_distribution"] = actionTypes
	result["details"].(map[string]interface{})["registry_distribution"] = registryTypes

	result["message"] = fmt.Sprintf("권한 규칙이 구성되어 있습니다 (%d개 규칙)", len(permissions))
	return result
}

// checkPackageFilter 패키지 필터 확인
func (sshc *SecuritySystemHealthChecker) checkPackageFilter() map[string]interface{} {
	result := map[string]interface{}{
		"status":  string(StatusHealthy),
		"enabled": sshc.config.Security.PackageFilter.Enabled,
		"details": make(map[string]interface{}),
	}

	if !sshc.config.Security.PackageFilter.Enabled {
		result["message"] = "패키지 필터가 비활성화되어 있습니다"
		return result
	}

	packageFilter := sshc.config.Security.PackageFilter
	result["details"].(map[string]interface{})["mode"] = packageFilter.Mode
	result["details"].(map[string]interface{})["rule_count"] = len(packageFilter.Rules)

	// 필터 모드 검증
	validModes := map[string]bool{
		"allowlist": true,
		"blocklist": true,
	}

	if !validModes[packageFilter.Mode] {
		result["status"] = string(StatusUnhealthy)
		result["message"] = fmt.Sprintf("잘못된 패키지 필터 모드: %s", packageFilter.Mode)
		result["error"] = "allowlist 또는 blocklist만 허용됩니다"
		return result
	}

	// 필터 규칙 검증
	if len(packageFilter.Rules) == 0 {
		result["status"] = string(StatusDegraded)
		result["message"] = "패키지 필터가 활성화되었지만 규칙이 없습니다"
		result["recommendation"] = "패키지 필터 규칙을 추가하거나 필터를 비활성화하세요"
		return result
	}

	// 규칙 분석
	ruleAnalysis := make(map[string]int)
	for _, rule := range packageFilter.Rules {
		ruleAnalysis[rule.Action]++
	}

	result["details"].(map[string]interface{})["rule_analysis"] = ruleAnalysis
	result["message"] = fmt.Sprintf("패키지 필터가 구성되어 있습니다 (%s 모드, %d개 규칙)",
		packageFilter.Mode, len(packageFilter.Rules))

	return result
}

// checkHashVerification 해시 검증 확인
func (sshc *SecuritySystemHealthChecker) checkHashVerification() map[string]interface{} {
	result := map[string]interface{}{
		"status":  string(StatusHealthy),
		"enabled": sshc.config.Security.HashVerification.Enabled,
		"details": make(map[string]interface{}),
	}

	if !sshc.config.Security.HashVerification.Enabled {
		result["status"] = string(StatusDegraded)
		result["message"] = "해시 검증이 비활성화되어 있습니다 (보안 위험)"
		result["recommendation"] = "패키지 무결성을 위해 해시 검증을 활성화하세요"
		return result
	}

	hashVerification := sshc.config.Security.HashVerification
	result["details"].(map[string]interface{})["required_headers"] = hashVerification.RequiredHashHeaders
	result["details"].(map[string]interface{})["fail_on_mismatch"] = hashVerification.FailOnMismatch

	if len(hashVerification.RequiredHashHeaders) == 0 {
		result["status"] = string(StatusDegraded)
		result["message"] = "해시 검증이 활성화되었지만 필수 헤더가 지정되지 않았습니다"
		result["recommendation"] = "SHA-256 등의 해시 헤더를 필수로 설정하세요"
		return result
	}

	// 권장 해시 알고리즘 확인
	recommendedHeaders := []string{"SHA-256", "SHA-512"}
	hasRecommended := false
	for _, header := range hashVerification.RequiredHashHeaders {
		for _, recommended := range recommendedHeaders {
			if strings.EqualFold(header, recommended) {
				hasRecommended = true
				break
			}
		}
		if hasRecommended {
			break
		}
	}

	if !hasRecommended {
		result["status"] = string(StatusDegraded)
		result["message"] = "권장하지 않는 해시 알고리즘만 사용 중입니다"
		result["recommendation"] = "SHA-256 또는 SHA-512 사용을 권장합니다"
		return result
	}

	result["message"] = fmt.Sprintf("해시 검증이 안전하게 구성되어 있습니다 (%d개 헤더)",
		len(hashVerification.RequiredHashHeaders))

	return result
}

// GetSecurityMetrics 보안 관련 메트릭 반환
func (sshc *SecuritySystemHealthChecker) GetSecurityMetrics() map[string]interface{} {
	return map[string]interface{}{
		"tls_enabled":               sshc.config.Server.TLS.Enabled,
		"authentication_methods":    sshc.countAuthenticationMethods(),
		"access_control_rules":      len(sshc.config.Security.AccessControl.Permissions),
		"ip_whitelist_enabled":      sshc.config.Security.AccessControl.IPWhitelist.Enabled,
		"package_filter_enabled":    sshc.config.Security.PackageFilter.Enabled,
		"hash_verification_enabled": sshc.config.Security.HashVerification.Enabled,
	}
}

// countAuthenticationMethods 활성화된 인증 방법 수 계산
func (sshc *SecuritySystemHealthChecker) countAuthenticationMethods() int {
	count := 0

	if sshc.config.Security.Authentication.BasicAuth != nil &&
		sshc.config.Security.Authentication.BasicAuth.Enabled != nil &&
		*sshc.config.Security.Authentication.BasicAuth.Enabled {
		count++
	}

	if sshc.config.Security.Authentication.OAuth2 != nil &&
		sshc.config.Security.Authentication.OAuth2.Enabled {
		count++
	}

	// JWT 지원은 현재 미구현
	// TODO: JWT 지원 추가 시 이 주석 제거

	return count
}
