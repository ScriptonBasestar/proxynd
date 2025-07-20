// License generator tool - 별도로 안전하게 관리되어야 함
package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"time"
)

// 개인 키 - 실제로는 절대 코드에 포함하면 안됨!
// 별도의 안전한 저장소에서 관리해야 함
const privateKeyPEM = `
-----BEGIN RSA PRIVATE KEY-----
실제 개인 키로 교체 필요 - 절대 커밋하지 말 것!
-----END RSA PRIVATE KEY-----
`

// License is exported
// License represents a data structure
type License struct {
	ID         string    `json:"id"`
	Company    string    `json:"company"`
	Email      string    `json:"email"`
	Features   []string  `json:"features"`
	MaxServers int       `json:"max_servers"`
	MaxUsers   int       `json:"max_users"`
	IssuedAt   time.Time `json:"issued_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Type       string    `json:"type"`
	Signature  string    `json:"signature"`
}

func main() {
	var (
		company     = flag.String("company", "", "Company name")
		email       = flag.String("email", "", "Contact email")
		licenseType = flag.String("type", "trial", "License type: trial, starter, professional, enterprise, ultimate")
		duration    = flag.Duration("duration", 365*24*time.Hour, "License duration")
		maxServers  = flag.Int("servers", 0, "Max servers (0 = unlimited)")
		maxUsers    = flag.Int("users", 0, "Max users (0 = unlimited)")
		output      = flag.String("output", "license.json", "Output file")
	)
	flag.Parse()

	if *company == "" || *email == "" {
		fmt.Println("Error: company and email are required")
		flag.Usage()
		os.Exit(1)
	}

	// 개인 키 로드
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		fmt.Println("Failed to parse PEM block")
		os.Exit(1)
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		fmt.Printf("Failed to parse private key: %v\n", err)
		os.Exit(1)
	}

	// 라이센스 타입별 기능 설정
	features := getFeaturesByType(*licenseType)

	// 라이센스 생성
	license := License{
		ID:         generateID(),
		Company:    *company,
		Email:      *email,
		Features:   features,
		MaxServers: *maxServers,
		MaxUsers:   *maxUsers,
		IssuedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(*duration),
		Type:       *licenseType,
	}

	// 서명 생성
	signature, err := signLicense(&license, privateKey)
	if err != nil {
		fmt.Printf("Failed to sign license: %v\n", err)
		os.Exit(1)
	}
	license.Signature = signature

	// 라이센스 저장
	data, err := json.MarshalIndent(license, "", "  ")
	if err != nil {
		fmt.Printf("Failed to marshal license: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*output, data, 0644); err != nil {
		fmt.Printf("Failed to write license file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("License generated successfully!\n")
	fmt.Printf("Company: %s\n", license.Company)
	fmt.Printf("Type: %s\n", license.Type)
	fmt.Printf("Valid until: %s\n", license.ExpiresAt.Format("2006-01-02"))
	fmt.Printf("Output: %s\n", *output)
}

func generateID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// rand.Read은 crypto/rand를 사용하므로 실패하기 어렵지만,
		// 만약 실패하면 패닉을 발생시킴 (라이센스 생성에서 중요한 보안 요소)
		panic(fmt.Sprintf("failed to generate random ID: %v", err))
	}
	return fmt.Sprintf("%x", b)
}

func signLicense(license *License, privateKey *rsa.PrivateKey) (string, error) {
	// 서명을 제외한 라이센스 데이터 직렬화
	license.Signature = ""
	data, err := json.Marshal(license)
	if err != nil {
		return "", err
	}

	// 해시 계산
	hashed := sha256.Sum256(data)

	// 서명 생성
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}

	// Base64 인코딩
	return base64.StdEncoding.EncodeToString(signature), nil
}

func getFeaturesByType(licenseType string) []string {
	switch licenseType {
	case "trial":
		return []string{
			"ldap_auth",
			"smart_caching",
			"advanced_analytics",
			"webhooks",
		}
	case "starter":
		return []string{
			"ldap_auth",
			"saml_auth",
			"smart_caching",
			"advanced_analytics",
			"webhooks",
			"audit_log",
			"priority_support",
		}
	case "professional":
		return []string{
			"ldap_auth",
			"saml_auth",
			"oauth2_advanced",
			"rbac",
			"smart_caching",
			"predictive_cache",
			"vulnerability_scanning",
			"license_scanning",
			"security_policies",
			"advanced_analytics",
			"custom_dashboards",
			"reporting",
			"webhooks",
			"graphql_api",
			"audit_log",
			"gdpr_compliance",
			"priority_support",
			"sla_guarantee",
		}
	case "enterprise":
		return []string{
			"ldap_auth",
			"saml_auth",
			"oauth2_advanced",
			"rbac",
			"mfa",
			"multi_datacenter",
			"geo_routing",
			"smart_caching",
			"predictive_cache",
			"bandwidth_control",
			"vulnerability_scanning",
			"license_scanning",
			"malware_scanning",
			"security_policies",
			"advanced_analytics",
			"custom_dashboards",
			"reporting",
			"alerting",
			"sla_monitoring",
			"webhooks",
			"graphql_api",
			"terraform_provider",
			"audit_log",
			"gdpr_compliance",
			"sox_compliance",
			"data_retention",
			"24x7_support",
			"sla_guarantee",
		}
	case "ultimate":
		return []string{"*"} // 모든 기능
	default:
		return []string{}
	}
}
