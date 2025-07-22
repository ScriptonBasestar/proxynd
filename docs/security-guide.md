# ProxyND Security Guide

## 📋 Overview

This guide provides comprehensive security information for ProxyND, covering deployment security, configuration hardening, monitoring, and incident response procedures.

## 🔒 Security Architecture

### Core Security Principles

1. **Defense in Depth**: Multiple layers of security controls
2. **Least Privilege**: Minimum necessary access rights
3. **Fail Secure**: Secure defaults and safe failure modes
4. **Security by Design**: Built-in security from the ground up

### Security Components

- **Authentication**: Multi-method authentication support
- **Authorization**: Role-based access control (RBAC)
- **Encryption**: TLS/HTTPS for all communications
- **Input Validation**: Comprehensive input sanitization
- **Audit Logging**: Detailed security event logging
- **Rate Limiting**: Protection against abuse and DoS

## 🛡️ Secure Deployment

### Container Security

#### Dockerfile Security Best Practices

```dockerfile
# Use specific, minimal base image
FROM golang:1.22-alpine AS builder

# Create non-root user
RUN addgroup -g 1001 -S proxynd && \
    adduser -u 1001 -S proxynd -G proxynd

# Set working directory
WORKDIR /app

# Copy and build application
COPY . .
RUN go mod tidy && \
    go build -ldflags="-w -s" -o proxynd main.go

# Production stage
FROM alpine:latest

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S proxynd && \
    adduser -u 1001 -S proxynd -G proxynd

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/proxynd .

# Change ownership
RUN chown -R proxynd:proxynd /app

# Switch to non-root user
USER proxynd

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ./proxynd healthcheck

# Expose port
EXPOSE 8080

# Run application
CMD ["./proxynd"]
```

#### Container Runtime Security

```yaml
# Kubernetes SecurityContext
apiVersion: v1
kind: Pod
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1001
    runAsGroup: 1001
    fsGroup: 1001
  containers:
  - name: proxynd
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      capabilities:
        drop:
        - ALL
        add:
        - NET_BIND_SERVICE
    resources:
      limits:
        cpu: 500m
        memory: 512Mi
      requests:
        cpu: 100m
        memory: 128Mi
```

### Network Security

#### TLS Configuration

```yaml
# global.yaml
server:
  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/proxynd.crt"
    key_file: "/etc/ssl/private/proxynd.key"
    min_version: "1.2"
    cipher_suites:
      - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
      - "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
      - "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA"
      - "TLS_RSA_WITH_AES_256_GCM_SHA384"
```

#### Firewall Rules

```bash
# Allow HTTPS traffic
iptables -A INPUT -p tcp --dport 443 -j ACCEPT

# Allow HTTP traffic (redirect to HTTPS)
iptables -A INPUT -p tcp --dport 80 -j ACCEPT

# Allow specific management port
iptables -A INPUT -p tcp --dport 8080 -s 10.0.0.0/8 -j ACCEPT

# Drop all other traffic
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT
```

## 🔧 Configuration Security

### Authentication Configuration

#### JWT Authentication

```yaml
# global.yaml
auth:
  method: "jwt"
  jwt:
    secret: "${JWT_SECRET}"  # Use environment variable
    expiration: "24h"
    refresh_expiration: "168h"
    issuer: "proxynd"
    audience: "proxynd-users"
```

#### OAuth2 Configuration

```yaml
# global.yaml
auth:
  method: "oauth2"
  oauth2:
    providers:
      github:
        client_id: "${GITHUB_CLIENT_ID}"
        client_secret: "${GITHUB_CLIENT_SECRET}"
        scopes: ["user:email"]
      google:
        client_id: "${GOOGLE_CLIENT_ID}"
        client_secret: "${GOOGLE_CLIENT_SECRET}"
        scopes: ["email", "profile"]
```

### Authorization Configuration

```yaml
# global.yaml
authorization:
  enabled: true
  rbac:
    roles:
      admin:
        permissions: ["*"]
      user:
        permissions: ["read", "download"]
      readonly:
        permissions: ["read"]

    assignments:
      - user: "admin@example.com"
        role: "admin"
      - group: "developers"
        role: "user"
```

### Rate Limiting

```yaml
# global.yaml
rate_limiting:
  enabled: true
  rules:
    - pattern: "/api/*"
      limit: 100
      window: "1m"
    - pattern: "/download/*"
      limit: 10
      window: "1m"
    - pattern: "/upload/*"
      limit: 5
      window: "1m"
```

### Security Headers

```yaml
# global.yaml
security:
  headers:
    x_frame_options: "DENY"
    x_content_type_options: "nosniff"
    x_xss_protection: "1; mode=block"
    strict_transport_security: "max-age=31536000; includeSubDomains"
    content_security_policy: "default-src 'self'"
    referrer_policy: "strict-origin-when-cross-origin"
```

## 🔍 Security Monitoring

### Audit Logging

```yaml
# global.yaml
logging:
  audit:
    enabled: true
    file: "/var/log/proxynd/audit.log"
    level: "info"
    format: "json"
    events:
      - "authentication"
      - "authorization"
      - "file_access"
      - "configuration_change"
      - "admin_action"
```

### Metrics and Alerting

```yaml
# Prometheus alerts
groups:
- name: proxynd-security
  rules:
  - alert: HighFailedAuthRate
    expr: rate(proxynd_auth_failures_total[5m]) > 0.1
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "High authentication failure rate"
      description: "ProxyND is experiencing high authentication failures"

  - alert: UnauthorizedAccess
    expr: rate(proxynd_unauthorized_attempts_total[5m]) > 0.05
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Unauthorized access attempts detected"
      description: "Multiple unauthorized access attempts detected"
```

### Log Analysis

```bash
# Monitor failed authentication attempts
grep "authentication failed" /var/log/proxynd/audit.log | \
  jq '.timestamp, .user, .ip, .reason' | \
  sort | uniq -c | sort -nr

# Check for suspicious file access patterns
grep "file_access" /var/log/proxynd/audit.log | \
  jq -r '.ip + " " + .file_path' | \
  sort | uniq -c | sort -nr | head -20

# Monitor configuration changes
grep "configuration_change" /var/log/proxynd/audit.log | \
  jq '.timestamp, .user, .change_type, .changed_keys'
```

## 🚨 Incident Response

### Security Incident Types

1. **Unauthorized Access**: Successful breach of authentication
2. **Data Exfiltration**: Unusual download patterns
3. **Malware Upload**: Suspicious file uploads
4. **DoS Attack**: Excessive request rates
5. **Configuration Tampering**: Unauthorized configuration changes

### Response Procedures

#### 1. Detection and Analysis

```bash
# Check recent authentication logs
tail -f /var/log/proxynd/audit.log | grep -E "(authentication|authorization)"

# Monitor current connections
netstat -an | grep :8080 | wc -l

# Check resource usage
top -p $(pgrep proxynd)
```

#### 2. Containment

```bash
# Block suspicious IP addresses
iptables -A INPUT -s <suspicious_ip> -j DROP

# Revoke user sessions
curl -X POST https://proxynd.example.com/admin/revoke-sessions \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"user_id": "suspicious_user"}'

# Enable emergency mode (read-only)
curl -X POST https://proxynd.example.com/admin/emergency-mode \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

#### 3. Investigation

```bash
# Generate security report
./scripts/security_audit.sh

# Check file integrity
find /app -type f -name "*.go" -exec sha256sum {} \; > /tmp/file_hashes.txt
diff /tmp/file_hashes.txt /app/security/baseline_hashes.txt

# Analyze access patterns
grep "file_access" /var/log/proxynd/audit.log | \
  jq -r '.ip + "," + .timestamp + "," + .file_path' | \
  sort > /tmp/access_analysis.csv
```

#### 4. Recovery

```bash
# Restore from backup
kubectl rollout undo deployment/proxynd

# Reset user credentials
./scripts/reset_user_credentials.sh

# Update security policies
kubectl apply -f security/updated-policies.yaml
```

## 🔐 Security Hardening Checklist

### Pre-Deployment

- [ ] Update all dependencies to latest versions
- [ ] Run security scanners (gosec, nancy, trivy)
- [ ] Review and harden configuration files
- [ ] Implement proper secret management
- [ ] Configure TLS with strong ciphers
- [ ] Set up proper logging and monitoring

### Post-Deployment

- [ ] Verify TLS configuration
- [ ] Test authentication and authorization
- [ ] Validate rate limiting
- [ ] Check security headers
- [ ] Monitor logs for anomalies
- [ ] Set up alerting rules

### Ongoing Maintenance

- [ ] Regular security audits
- [ ] Dependency updates
- [ ] Log analysis
- [ ] Backup verification
- [ ] Incident response testing
- [ ] Security training for team

## 🛠️ Security Tools

### Automated Security Scanning

```bash
# Run comprehensive security audit
./scripts/security_audit.sh

# Check for vulnerabilities in dependencies
nancy sleuth < go.list

# Scan for secrets
trufflehog git file://. --only-verified

# Container security scan
trivy image proxynd:latest
```

### Manual Security Testing

```bash
# Test authentication
curl -H "Authorization: Bearer invalid_token" https://proxynd.example.com/api/status

# Test rate limiting
for i in {1..200}; do curl https://proxynd.example.com/api/status; done

# Test input validation
curl -X POST https://proxynd.example.com/api/upload \
  -F "file=@/dev/null" \
  -F "filename=../../../etc/passwd"
```

## 📚 Security Resources

### Standards and Frameworks

- **OWASP**: https://owasp.org/
- **NIST Cybersecurity Framework**: https://www.nist.gov/cyberframework
- **CIS Controls**: https://www.cisecurity.org/controls/

### Go Security Best Practices

- **Go Security Checker**: https://github.com/securecodewarrior/gosec
- **Go Vulnerability Database**: https://vuln.go.dev/
- **Go Security Guide**: https://golang.org/security/

### Container Security

- **CIS Docker Benchmark**: https://www.cisecurity.org/benchmark/docker
- **Kubernetes Security**: https://kubernetes.io/docs/concepts/security/
- **Container Security Best Practices**: https://sysdig.com/blog/container-security-best-practices/

## 🚀 Emergency Contacts

### Security Team

- **Security Lead**: security-lead@example.com
- **On-Call Security**: +1-555-0123
- **Security Slack**: #security-incidents

### Escalation Matrix

1. **Level 1**: Development Team
2. **Level 2**: Security Team
3. **Level 3**: Management/Legal
4. **Level 4**: External Security Consultant

---

**Last Updated**: 2024-07-17  
**Review Schedule**: Monthly  
**Next Review**: 2024-08-17
