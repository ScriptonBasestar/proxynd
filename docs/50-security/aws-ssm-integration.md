# AWS SSM Parameter Store Integration

> **Enterprise Feature** | Secure secrets management using AWS Systems Manager Parameter Store

## Overview

ProxyND supports integration with AWS Systems Manager (SSM) Parameter Store for secure storage and retrieval of secrets, credentials, and configuration values. This eliminates the need to store sensitive data in configuration files.

## Features

- **Secure Secrets Storage**: Store passwords, API keys, and tokens securely in AWS SSM
- **Automatic Decryption**: Transparent decryption of SecureString parameters
- **Intelligent Caching**: Reduces AWS API calls with configurable TTL
- **IAM Integration**: Use IAM roles for secure credential management
- **Hierarchical Parameters**: Organize parameters using path-based structure
- **Batch Retrieval**: Fetch multiple parameters efficiently

## Configuration

### Basic Configuration

```yaml
# global.yaml
aws_ssm:
  enabled: true
  region: "us-east-1"
  cache_ttl: 300  # 5 minutes
  parameter_prefix: "/proxynd/"
```

### With AWS Credentials

```yaml
aws_ssm:
  enabled: true
  region: "us-east-1"
  access_key_id: "${AWS_ACCESS_KEY_ID}"
  secret_access_key: "${AWS_SECRET_ACCESS_KEY}"
  # session_token: "${AWS_SESSION_TOKEN}"  # Optional for temporary credentials
  cache_ttl: 300
  parameter_prefix: "/proxynd/"
```

### Using IAM Roles (Recommended)

When running on EC2, ECS, or Lambda, omit credentials to use IAM roles:

```yaml
aws_ssm:
  enabled: true
  region: "us-east-1"
  cache_ttl: 300
  parameter_prefix: "/proxynd/"
```

## Parameter Management

### Creating Parameters

#### Using AWS CLI

```bash
# String parameter
aws ssm put-parameter \
  --name "/proxynd/auth/admin/password" \
  --value "your-secure-password" \
  --type "String" \
  --description "ProxyND admin password"

# SecureString (encrypted)
aws ssm put-parameter \
  --name "/proxynd/s3/secret-access-key" \
  --value "wJalrXUtnFEMI/K7MDENG/bPxRfiCY" \
  --type "SecureString" \
  --description "S3 cache secret key"

# With KMS encryption
aws ssm put-parameter \
  --name "/proxynd/oauth2/client-secret" \
  --value "your-oauth2-client-secret" \
  --type "SecureString" \
  --key-id "alias/proxynd-kms" \
  --description "OAuth2 client secret"
```

#### Parameter Naming Convention

Use hierarchical paths for organization:

```
/proxynd/
  ├── auth/
  │   ├── admin/password
  │   ├── ldap/bind-password
  │   └── oauth2/client-secret
  ├── s3/
  │   ├── access-key-id
  │   └── secret-access-key
  ├── database/
  │   └── connection-string
  └── api-keys/
      ├── github-token
      └── gitlab-token
```

### Retrieving Parameters

#### Single Parameter

```go
import (
    "context"
    "proxynd/internal/adapters/aws"
)

// Create AWS client
client, err := aws.NewAWSClient(ctx, aws.ClientConfig{
    Region: "us-east-1",
})
if err != nil {
    // Handle error
}

// Get parameter
value, err := client.GetParameter(ctx, "/proxynd/auth/admin/password", true)
if err != nil {
    // Handle error
}
```

#### Multiple Parameters

```go
names := []string{
    "/proxynd/s3/access-key-id",
    "/proxynd/s3/secret-access-key",
}

params, err := client.GetParameters(ctx, names, true)
if err != nil {
    // Handle error
}

accessKey := params["/proxynd/s3/access-key-id"]
secretKey := params["/proxynd/s3/secret-access-key"]
```

#### Parameters by Path

```go
// Get all parameters under a path (recursive)
params, err := client.GetParametersByPath(ctx, "/proxynd/auth/", true, true)
if err != nil {
    // Handle error
}

// All auth parameters are now in the map
for name, value := range params {
    fmt.Printf("%s = %s\n", name, value)
}
```

## Caching

### SSMCache Usage

The SSMCache provides intelligent caching to reduce AWS API calls:

```go
import (
    "time"
    "proxynd/internal/adapters/aws"
)

// Create cache with 5-minute TTL
cache := aws.NewSSMCache(client, 5*time.Minute)

// Start auto-cleanup of expired entries
cache.StartAutoCleanup(1 * time.Minute)

// Get parameter (cached)
value, err := cache.GetParameter(ctx, "/proxynd/auth/admin/password", true)

// Invalidate specific parameter
cache.Invalidate("/proxynd/auth/admin/password")

// Clear entire cache
cache.InvalidateAll()

// Get cache statistics
stats := cache.GetStats()
fmt.Printf("Total entries: %d\n", stats["total_entries"])
fmt.Printf("Valid entries: %d\n", stats["valid_entries"])
```

### Cache Behavior

- **Cache Hit**: Returns value immediately without AWS API call
- **Cache Miss**: Fetches from SSM and caches for TTL duration
- **Expired Entry**: Automatically refetched on next access
- **Auto Cleanup**: Background goroutine removes expired entries

## Security Best Practices

### 1. Use IAM Roles

**Recommended** for EC2, ECS, EKS, Lambda:

```yaml
aws_ssm:
  enabled: true
  region: "us-east-1"
  # No credentials - uses IAM role
```

**IAM Policy Example**:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ssm:GetParameter",
        "ssm:GetParameters",
        "ssm:GetParametersByPath",
        "ssm:DescribeParameters"
      ],
      "Resource": "arn:aws:ssm:us-east-1:123456789012:parameter/proxynd/*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "kms:Decrypt"
      ],
      "Resource": "arn:aws:kms:us-east-1:123456789012:key/*"
    }
  ]
}
```

### 2. Use SecureString Parameters

Always use `SecureString` type for sensitive data:

```bash
aws ssm put-parameter \
  --name "/proxynd/sensitive/value" \
  --value "secret" \
  --type "SecureString"  # Encrypted at rest
```

### 3. Enable Parameter Encryption

Use AWS KMS for additional encryption:

```bash
aws ssm put-parameter \
  --name "/proxynd/db/password" \
  --value "db-password" \
  --type "SecureString" \
  --key-id "alias/proxynd-kms"
```

### 4. Implement Least Privilege

Limit access to specific parameter paths:

- Use path-based IAM policies
- Separate parameters by environment
- Use tags for access control

### 5. Enable CloudTrail

Monitor parameter access:

```bash
# CloudTrail logs all SSM API calls
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=ResourceType,AttributeValue=AWS::SSM::Parameter
```

## Use Cases

### 1. Database Credentials

```yaml
# Store in SSM
aws ssm put-parameter \
  --name "/proxynd/database/connection-string" \
  --value "postgres://user:pass@host:5432/db" \
  --type "SecureString"

# Reference in code
connectionString, _ := client.GetParameter(ctx, "/proxynd/database/connection-string", true)
```

### 2. OAuth2 Secrets

```yaml
# Store client secret
aws ssm put-parameter \
  --name "/proxynd/oauth2/github/client-secret" \
  --value "github-client-secret-value" \
  --type "SecureString"

# Retrieve for OAuth2 configuration
clientSecret, _ := cache.GetParameter(ctx, "/proxynd/oauth2/github/client-secret", true)
```

### 3. API Keys

```yaml
# Store multiple API keys
aws ssm put-parameter --name "/proxynd/api-keys/service-a" --value "key-a" --type "SecureString"
aws ssm put-parameter --name "/proxynd/api-keys/service-b" --value "key-b" --type "SecureString"

# Retrieve all at once
apiKeys, _ := client.GetParametersByPath(ctx, "/proxynd/api-keys/", true, false)
```

### 4. S3 Credentials for Cache

```yaml
# Store S3 credentials
aws ssm put-parameter --name "/proxynd/s3/access-key-id" --value "AKIAIO..." --type "SecureString"
aws ssm put-parameter --name "/proxynd/s3/secret-access-key" --value "wJalr..." --type "SecureString"

# Use in S3 cache configuration
keys, _ := cache.GetParameters(ctx, []string{
    "/proxynd/s3/access-key-id",
    "/proxynd/s3/secret-access-key",
}, true)
```

## Troubleshooting

### Permission Denied

**Error**: `AccessDeniedException: User is not authorized to perform: ssm:GetParameter`

**Solution**: Add SSM permissions to IAM role/user:

```json
{
  "Effect": "Allow",
  "Action": ["ssm:GetParameter", "ssm:GetParameters"],
  "Resource": "arn:aws:ssm:*:*:parameter/proxynd/*"
}
```

### Parameter Not Found

**Error**: `ParameterNotFound: Parameter /proxynd/... not found`

**Solution**: Create the parameter first:

```bash
aws ssm put-parameter --name "/proxynd/param" --value "value" --type "String"
```

### Decryption Failed

**Error**: `KMS.NotFoundException: Key 'arn:aws:kms:...' does not exist`

**Solution**: Add KMS decrypt permission:

```json
{
  "Effect": "Allow",
  "Action": ["kms:Decrypt"],
  "Resource": "arn:aws:kms:*:*:key/*"
}
```

### Region Mismatch

**Error**: Parameters not found in configured region

**Solution**: Ensure region matches where parameters are stored:

```yaml
aws_ssm:
  region: "us-east-1"  # Must match parameter region
```

## Performance Considerations

1. **Enable Caching**: Use SSMCache to reduce API calls
2. **Batch Retrieval**: Use GetParameters or GetParametersByPath for multiple parameters
3. **Tune TTL**: Balance between freshness and API costs
4. **Auto Cleanup**: Enable to prevent memory leaks

## Monitoring

### CloudWatch Metrics

Monitor SSM API usage:

- `GetParameter` call count
- `GetParameters` call count
- API error rates
- Latency

### Application Metrics

Track cache performance:

```go
stats := cache.GetStats()
log.Printf("Cache stats: %+v", stats)
```

## Related Documentation

- [AWS SSM Parameter Store Documentation](https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html)
- [ProxyND Security Guide](./security-overview.md)
- [Configuration Management](../20-configuration/configuration-reference.md)
- [AWS IAM Best Practices](https://docs.aws.amazon.com/IAM/latest/UserGuide/best-practices.html)

---

**Last Updated**: 2025-11-27
**Version**: 1.0
