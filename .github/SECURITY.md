# Security Policy

## Supported Versions

We actively maintain security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| 0.9.x   | :x:                |
| < 0.9   | :x:                |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security vulnerability in ProxyND, please report it responsibly.

### How to Report

1. **DO NOT** create a public GitHub issue for security vulnerabilities
2. Send an email to `security@proxynd.example.com` with:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if available)

### What to Expect

- **Acknowledgment**: We will acknowledge receipt within 24 hours
- **Initial Assessment**: Initial assessment within 72 hours
- **Progress Updates**: Regular updates every 7 days until resolution
- **Resolution Timeline**:
  - Critical: 24-48 hours
  - High: 3-7 days
  - Medium: 14-30 days
  - Low: 30-90 days

### Disclosure Policy

- We follow coordinated disclosure
- Security fixes will be released as soon as possible
- Public disclosure after fix is available and deployed
- Credit will be given to reporters (unless they prefer anonymity)

## Security Best Practices

### For Users

1. **Keep Updated**: Always use the latest stable version
2. **Secure Configuration**:
   - Use HTTPS for all endpoints
   - Enable authentication for admin endpoints
   - Configure proper firewall rules
3. **Monitor Logs**: Regularly check access and error logs
4. **Resource Limits**: Set appropriate resource limits

### For Developers

1. **Code Security**:
   - Run `gosec` before committing
   - Use `go mod tidy` to keep dependencies clean
   - Validate all user inputs
   - Use prepared statements for database queries

2. **Dependencies**:
   - Regularly update dependencies
   - Use `nancy` to scan for vulnerabilities
   - Pin dependency versions

3. **Secrets Management**:
   - Never commit secrets to version control
   - Use environment variables or secret management systems
   - Rotate credentials regularly

## Security Features

### Built-in Security

- **TLS/HTTPS**: All communications encrypted
- **Authentication**: Multiple authentication methods supported
- **Authorization**: Role-based access control
- **Input Validation**: All inputs validated and sanitized
- **Rate Limiting**: Protection against abuse
- **Security Headers**: Standard security headers included

### Configuration Security

```yaml
# Example secure configuration
server:
  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/proxynd.crt"
    key_file: "/etc/ssl/private/proxynd.key"

auth:
  enabled: true
  method: "jwt"

security:
  rate_limit:
    enabled: true
    requests_per_minute: 100
  cors:
    enabled: true
    allowed_origins: ["https://trusted-domain.com"]
```

### Container Security

- Base images regularly updated
- Non-root user execution
- Minimal attack surface
- Security scanning in CI/CD

## Compliance

ProxyND is designed to help meet:

- **OWASP Top 10**: Protection against common web vulnerabilities
- **CIS Controls**: Implementation of critical security controls
- **NIST Framework**: Alignment with cybersecurity framework

## Security Contacts

- **Security Team**: security@proxynd.example.com
- **PGP Key**: Available at https://proxynd.example.com/.well-known/pgp-key.txt
- **Security Updates**: https://github.com/[repository]/security/advisories

## Hall of Fame

We acknowledge security researchers who have helped improve ProxyND's security:

<!-- Security researchers will be listed here after responsible disclosure -->

---

**Last Updated**: 2024-07-17  
**Review Schedule**: Quarterly
