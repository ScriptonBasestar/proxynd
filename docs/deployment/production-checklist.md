# Production Deployment Checklist

Comprehensive checklist for deploying ProxyND Enterprise API to production environments.

## 📋 Overview

This checklist ensures all critical aspects are covered before, during, and after production deployment.

**Deployment Stages**:
1. **Pre-Deployment** - Validation and preparation (Est: 2-4 hours)
2. **Deployment** - Actual deployment process (Est: 1-2 hours)
3. **Post-Deployment** - Verification and monitoring (Est: 1-2 hours)
4. **Rollback Plan** - Emergency procedures (Reference only)

**Total Estimated Time**: 4-8 hours for initial production deployment

---

## ✅ Pre-Deployment Checklist

### 1. Environment Validation

- [ ] **Infrastructure provisioned and ready**
  - [ ] Compute resources (CPU, memory match capacity plan)
  - [ ] Storage volumes (sufficient space for cache and logs)
  - [ ] Network configuration (VPC, subnets, security groups)
  - [ ] Load balancer configured and tested
  - [ ] DNS records created and propagated

- [ ] **Dependencies verified**
  - [ ] Database (PostgreSQL/MySQL) running and accessible
  - [ ] Cache (Redis) running with proper configuration
  - [ ] S3-compatible storage accessible (if using S3 cache)
  - [ ] Prometheus/Grafana monitoring stack deployed
  - [ ] Log aggregation system configured (ELK, Loki, etc.)

### 2. Configuration Validation

- [ ] **Environment variables set**
  ```bash
  # Verify all required env vars
  CONFIG_DIR=/etc/proxynd
  STORAGE_DIR=/var/lib/proxynd
  SERVER_PORT=8080
  LOG_LEVEL=info
  LOG_FORMAT=json
  GO_ENV=production  # CRITICAL: Must be "production"
  ```

- [ ] **Configuration file validated**
  ```bash
  # Test config syntax
  proxyndctl config validate

  # Review critical settings
  cat $CONFIG_DIR/config.yaml
  ```

  **Critical config items**:
  - [ ] Server port and bind address
  - [ ] Database connection string (with connection pooling)
  - [ ] Redis/cache configuration (with TTL settings)
  - [ ] Upstream proxy configurations for all 7 package managers
  - [ ] Authentication providers (OAuth2, JWT secrets)
  - [ ] Metrics export enabled (`/metrics` endpoint)
  - [ ] Log level set to `info` or `warn` (not `debug` in production)

- [ ] **Secrets management**
  - [ ] Database password stored in secrets manager
  - [ ] Redis password secured
  - [ ] OAuth2 client secrets encrypted
  - [ ] JWT signing keys rotated and secured
  - [ ] S3/cloud storage credentials secured
  - [ ] API keys for monitoring/alerting secured

### 3. License Validation

- [ ] **Enterprise license verified**
  ```bash
  # Check license file exists
  ls -la $CONFIG_DIR/license.json

  # Validate license
  proxyndctl license info

  # Verify license details
  # - Expiration date (> 30 days from now)
  # - Licensed features enabled
  # - User/seat limits sufficient
  # - Valid for production use
  ```

- [ ] **License expiration monitoring**
  - [ ] Alert configured for 30-day expiration warning
  - [ ] Renewal process documented
  - [ ] Contact info for license vendor saved

### 4. Security Hardening

- [ ] **TLS/SSL certificates**
  - [ ] Valid SSL certificates installed
  - [ ] Certificate expiration > 30 days
  - [ ] Intermediate certificates included
  - [ ] Certificate auto-renewal configured (Let's Encrypt, ACM, etc.)

- [ ] **Network security**
  - [ ] Firewall rules configured (allow only required ports)
  - [ ] Security groups restrict access to known IPs/CIDRs
  - [ ] DDoS protection enabled (CloudFlare, AWS Shield, etc.)
  - [ ] Rate limiting configured per endpoint

- [ ] **Access control**
  - [ ] RBAC roles defined and assigned
  - [ ] Admin access restricted to authorized personnel
  - [ ] Audit logging enabled for all privileged operations
  - [ ] MFA enabled for admin accounts

- [ ] **Vulnerability scanning**
  - [ ] Run security scan on container image or binary
    ```bash
    # Example with Trivy
    trivy image proxynd:latest
    ```
  - [ ] No critical vulnerabilities present
  - [ ] Known issues documented with mitigation plans

### 5. Database Migration

- [ ] **Database schema up-to-date**
  ```bash
  # Run migrations (if applicable)
  proxyndctl db migrate

  # Verify schema version
  proxyndctl db version
  ```

- [ ] **Database backup created**
  ```bash
  # Create pre-deployment backup
  pg_dump -U postgres proxynd > backup-$(date +%Y%m%d-%H%M%S).sql

  # Verify backup integrity
  pg_restore --list backup-*.sql
  ```

- [ ] **Database performance tuning**
  - [ ] Indexes created on frequently queried columns
  - [ ] Connection pool size configured (recommended: 20-50)
  - [ ] Query timeout set (recommended: 30s)

### 6. Testing

- [ ] **Unit tests passed**
  ```bash
  make test-unit
  ```

- [ ] **Integration tests passed**
  ```bash
  make test-integration
  ```

- [ ] **Load tests completed**
  ```bash
  # Run standard load test
  ./scripts/loadtest/load-test-enterprise.sh standard

  # Verify performance targets met:
  # - P95 latency < 200ms
  # - P99 latency < 500ms
  # - Throughput > 1000 rps
  # - Error rate < 0.1%
  ```

- [ ] **Smoke tests prepared**
  - [ ] Test scripts for each package manager
  - [ ] Test data ready (sample packages, users, etc.)

### 7. Monitoring & Alerting

- [ ] **Prometheus configured**
  ```bash
  # Verify Prometheus can scrape ProxyND
  curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | select(.job=="proxynd-enterprise-api")'
  ```

- [ ] **Alert rules loaded**
  ```bash
  # Check alert rules
  curl http://localhost:9090/api/v1/rules | jq '.data.groups[].name'

  # Expected: enterprise_api_performance, enterprise_api_errors, etc.
  ```

- [ ] **Grafana dashboard imported**
  - [ ] Dashboard accessible: http://grafana/d/proxynd-enterprise-api/
  - [ ] All panels showing data
  - [ ] Alerts configured in Grafana

- [ ] **Alertmanager configured**
  - [ ] Email notifications tested
  - [ ] Slack/PagerDuty integration tested
  - [ ] Alert routing rules configured

- [ ] **Log aggregation**
  - [ ] Logs flowing to aggregation system
  - [ ] Log retention policy configured
  - [ ] Log search/query tested

### 8. Documentation

- [ ] **Runbooks created**
  - [ ] Common operational tasks documented
  - [ ] Troubleshooting guides prepared
  - [ ] Emergency contact list updated

- [ ] **Deployment notes**
  - [ ] Deployment plan documented
  - [ ] Rollback procedure documented
  - [ ] Known issues and workarounds listed

- [ ] **Team notification**
  - [ ] DevOps team notified of deployment window
  - [ ] On-call engineer identified
  - [ ] Stakeholders informed

---

## 🚀 Deployment Checklist

### 1. Pre-Deployment Verification

- [ ] **Final checks**
  ```bash
  # Verify build
  ./tmp/bin/proxynd --version

  # Check config one last time
  proxyndctl config validate

  # Verify license
  proxyndctl license info
  ```

- [ ] **Deployment window confirmed**
  - [ ] Low-traffic period selected
  - [ ] Stakeholders notified
  - [ ] Maintenance window scheduled (if required)

- [ ] **Backup current production** (if upgrading)
  ```bash
  # Database backup
  pg_dump -U postgres proxynd > pre-deployment-backup-$(date +%Y%m%d-%H%M%S).sql

  # Configuration backup
  tar -czf config-backup-$(date +%Y%m%d-%H%M%S).tar.gz $CONFIG_DIR

  # Verify backups
  ls -lh *backup*.sql *backup*.tar.gz
  ```

### 2. Deployment Execution

#### Option A: Docker Deployment

```bash
# 1. Pull latest image
docker pull proxynd:v1.0.0

# 2. Stop old container (if exists)
docker stop proxynd || true
docker rm proxynd || true

# 3. Start new container
docker run -d \
  --name proxynd \
  --restart unless-stopped \
  -p 8080:8080 \
  -v $CONFIG_DIR:/etc/proxynd:ro \
  -v $STORAGE_DIR:/var/lib/proxynd \
  -e GO_ENV=production \
  proxynd:v1.0.0

# 4. Verify container started
docker ps | grep proxynd
docker logs --tail 50 proxynd
```

- [ ] Container started successfully
- [ ] Logs show no errors
- [ ] Health check passing

#### Option B: Kubernetes Deployment

```bash
# 1. Apply ConfigMap and Secrets
kubectl apply -f k8s/proxynd-configmap.yaml
kubectl apply -f k8s/proxynd-secrets.yaml

# 2. Deploy application
kubectl apply -f k8s/proxynd-deployment.yaml

# 3. Apply Service and Ingress
kubectl apply -f k8s/proxynd-service.yaml
kubectl apply -f k8s/proxynd-ingress.yaml

# 4. Verify deployment
kubectl rollout status deployment/proxynd -n production
kubectl get pods -n production -l app=proxynd

# 5. Check logs
kubectl logs -n production -l app=proxynd --tail=100
```

- [ ] Deployment rolled out successfully
- [ ] All pods running and ready
- [ ] Service endpoints created
- [ ] Ingress configured correctly

#### Option C: Systemd Service

```bash
# 1. Copy binary
sudo cp ./tmp/bin/proxynd /usr/local/bin/
sudo chmod +x /usr/local/bin/proxynd

# 2. Copy service file
sudo cp deployments/systemd/proxynd.service /etc/systemd/system/

# 3. Reload systemd
sudo systemctl daemon-reload

# 4. Start service
sudo systemctl start proxynd

# 5. Enable auto-start
sudo systemctl enable proxynd

# 6. Check status
sudo systemctl status proxynd
sudo journalctl -u proxynd -f
```

- [ ] Service started successfully
- [ ] Service enabled for auto-start
- [ ] No errors in journalctl

### 3. Health Checks

- [ ] **Service health check**
  ```bash
  # Health endpoint
  curl http://localhost:8080/health

  # Expected: {"status":"ok","version":"v1.0.0"}
  ```

- [ ] **Readiness check**
  ```bash
  curl http://localhost:8080/ready

  # Expected: {"status":"ready"}
  ```

- [ ] **Metrics endpoint**
  ```bash
  curl http://localhost:8080/metrics | head -20

  # Should see Prometheus metrics
  ```

### 4. Functional Testing

- [ ] **Test each package manager**
  ```bash
  # Maven
  proxyndctl test --proxy maven

  # NPM
  proxyndctl test --proxy npm

  # Docker
  proxyndctl test --proxy docker

  # PyPI
  proxyndctl test --proxy pypi

  # APT
  proxyndctl test --proxy apt

  # YUM
  proxyndctl test --proxy yum

  # APK
  proxyndctl test --proxy apk
  ```

- [ ] **Test Enterprise API endpoints**
  ```bash
  # RBAC
  curl http://localhost:8080/api/v1/enterprise/rbac/roles

  # Analytics
  curl http://localhost:8080/api/v1/enterprise/analytics/overview

  # Audit
  curl http://localhost:8080/api/v1/enterprise/audit/events

  # Security
  curl http://localhost:8080/api/v1/enterprise/security/vulnerabilities
  ```

- [ ] **Test authentication**
  ```bash
  # Get JWT token
  TOKEN=$(curl -X POST http://localhost:8080/auth/login \
    -d '{"username":"admin","password":"secret"}' | jq -r .token)

  # Use token
  curl -H "Authorization: Bearer $TOKEN" \
    http://localhost:8080/api/v1/enterprise/rbac/roles
  ```

### 5. Load Balancer Configuration

- [ ] **Add instance to load balancer**
  ```bash
  # AWS ALB example
  aws elbv2 register-targets \
    --target-group-arn arn:aws:elasticloadbalancing:... \
    --targets Id=i-xxxxx

  # Verify target health
  aws elbv2 describe-target-health \
    --target-group-arn arn:aws:elasticloadbalancing:...
  ```

- [ ] **Health checks passing on load balancer**
- [ ] **SSL termination configured**
- [ ] **Sticky sessions enabled** (if required)

---

## ✓ Post-Deployment Checklist

### 1. Immediate Verification (0-15 minutes)

- [ ] **Service status**
  ```bash
  # Check process is running
  ps aux | grep proxynd

  # Check listening ports
  netstat -tlnp | grep 8080

  # Check resource usage
  top -p $(pgrep proxynd)
  ```

- [ ] **Log monitoring**
  ```bash
  # Watch logs in real-time
  tail -f /var/log/proxynd/proxynd.log

  # Or with Docker
  docker logs -f proxynd

  # Or with Kubernetes
  kubectl logs -f -l app=proxynd -n production

  # Look for:
  # - No error/critical messages
  # - Successful startup messages
  # - API requests being logged
  ```

- [ ] **Metrics verification**
  ```bash
  # Check basic metrics
  curl -s http://localhost:8080/metrics | grep -E "proxynd_enterprise_api_(requests|errors|cache)"

  # Expected metrics present:
  # - proxynd_enterprise_api_requests_total
  # - proxynd_enterprise_api_errors_total
  # - proxynd_enterprise_api_cache_hits_total
  ```

### 2. Short-term Monitoring (15-60 minutes)

- [ ] **Grafana dashboard review**
  - [ ] Request rate stable
  - [ ] Latency within acceptable range (P95 < 200ms)
  - [ ] Error rate low (< 1%)
  - [ ] Cache hit rate good (> 85%)
  - [ ] No memory leaks (memory usage stable)

- [ ] **Alert verification**
  - [ ] No critical alerts fired
  - [ ] Alertmanager receiving metrics
  - [ ] Test alert (send manual test alert to verify notifications)

- [ ] **Database monitoring**
  ```bash
  # Check connection count
  SELECT count(*) FROM pg_stat_activity WHERE datname='proxynd';

  # Check slow queries
  SELECT query, calls, mean_exec_time
  FROM pg_stat_statements
  ORDER BY mean_exec_time DESC
  LIMIT 10;
  ```

- [ ] **Cache monitoring**
  ```bash
  # Redis info
  redis-cli info stats

  # Check hit rate
  redis-cli info stats | grep keyspace
  ```

### 3. Medium-term Validation (1-4 hours)

- [ ] **Performance baseline established**
  ```bash
  # Run load test against production
  ./scripts/loadtest/load-test-enterprise.sh quick

  # Compare results to staging/pre-production
  # - Latency comparable or better
  # - Throughput meets expectations
  # - No errors under load
  ```

- [ ] **User acceptance**
  - [ ] Sample of users tested successfully
  - [ ] No user-reported issues
  - [ ] Package downloads working for all 7 managers

- [ ] **Integration verification**
  - [ ] Upstream registries accessible
  - [ ] Cache warming working
  - [ ] Authentication working
  - [ ] Audit logs being written

### 4. Long-term Monitoring (4-24 hours)

- [ ] **Trend analysis**
  - [ ] Request patterns match expectations
  - [ ] No unusual error spikes
  - [ ] Cache performance stable
  - [ ] Memory usage not growing unbounded

- [ ] **Capacity validation**
  - [ ] CPU utilization < 70% during peak
  - [ ] Memory utilization < 80%
  - [ ] Disk space sufficient (> 20% free)
  - [ ] Network bandwidth adequate

- [ ] **Cost analysis**
  - [ ] Cloud costs within budget
  - [ ] Cache efficiency reducing upstream bandwidth
  - [ ] Database query efficiency acceptable

### 5. Documentation Updates

- [ ] **Update deployment log**
  ```markdown
  ## Deployment 2025-01-17
  - Version: v1.0.0
  - Deployed by: [Name]
  - Start time: 2025-01-17 10:00 UTC
  - End time: 2025-01-17 11:30 UTC
  - Status: Success
  - Issues: None
  - Performance: P95 latency 150ms, throughput 1200 rps
  ```

- [ ] **Update runbooks with lessons learned**
- [ ] **Document any configuration changes**
- [ ] **Update capacity planning based on actual usage**

---

## 🔙 Rollback Plan

**When to rollback**:
- Critical errors preventing service operation
- Data corruption detected
- Security vulnerability introduced
- Performance degradation > 50%
- Error rate > 10% for > 5 minutes

### Quick Rollback (< 5 minutes)

#### Docker Rollback

```bash
# 1. Stop new container
docker stop proxynd
docker rm proxynd

# 2. Start previous version
docker run -d \
  --name proxynd \
  --restart unless-stopped \
  -p 8080:8080 \
  -v $CONFIG_DIR:/etc/proxynd:ro \
  -v $STORAGE_DIR:/var/lib/proxynd \
  -e GO_ENV=production \
  proxynd:v0.9.0  # Previous version tag

# 3. Verify rollback
docker ps | grep proxynd
curl http://localhost:8080/health
```

#### Kubernetes Rollback

```bash
# 1. Rollback deployment
kubectl rollout undo deployment/proxynd -n production

# 2. Verify rollback
kubectl rollout status deployment/proxynd -n production

# 3. Check pods are running previous version
kubectl get pods -n production -l app=proxynd -o jsonpath='{.items[*].spec.containers[*].image}'
```

#### Systemd Rollback

```bash
# 1. Stop service
sudo systemctl stop proxynd

# 2. Replace binary with previous version
sudo cp /opt/proxynd/backups/proxynd-v0.9.0 /usr/local/bin/proxynd

# 3. Start service
sudo systemctl start proxynd

# 4. Verify
sudo systemctl status proxynd
curl http://localhost:8080/health
```

### Database Rollback

**Only if schema changes were made**:

```bash
# 1. Stop application
# (Use appropriate method from above)

# 2. Restore database backup
psql -U postgres proxynd < pre-deployment-backup-*.sql

# 3. Verify restoration
psql -U postgres proxynd -c "SELECT version FROM schema_migrations;"

# 4. Restart application with previous version
```

### Post-Rollback

- [ ] **Verify service health**
- [ ] **Notify stakeholders**
- [ ] **Document rollback reason**
- [ ] **Schedule post-mortem**
- [ ] **Fix issues in development**
- [ ] **Plan re-deployment**

---

## 📞 Emergency Contacts

### Key Personnel

| Role | Name | Contact | Availability |
|------|------|---------|--------------|
| **On-Call Engineer** | [UPDATE] | [UPDATE] | 24/7 |
| **DevOps Lead** | [UPDATE] | [UPDATE] | Business hours |
| **Database Admin** | [UPDATE] | [UPDATE] | On-call |
| **Security Team** | [UPDATE] | [UPDATE] | On-call |

### External Contacts

| Service | Contact | Purpose |
|---------|---------|---------|
| **Cloud Provider Support** | [UPDATE] | Infrastructure issues |
| **License Vendor** | [UPDATE] | License problems |
| **Package Registry** | [UPDATE] | Upstream issues |

---

## 📚 Reference Links

- **Documentation**: [../../README.md](../../README.md)
- **API Reference**: [../04-api-reference/README.md](../04-api-reference/README.md)
- **Monitoring Setup**: [../deployment/performance-baseline.md](performance-baseline.md)
- **Alert Rules**: [../../deployments/prometheus/README.md](../../deployments/prometheus/README.md)
- **Grafana Dashboard**: [../../deployments/grafana/README.md](../../deployments/grafana/README.md)
- **Load Testing**: [../../scripts/loadtest/README.md](../../scripts/loadtest/README.md)

---

**Version**: 1.0
**Last Updated**: 2025-01-17
**Next Review**: [UPDATE: Schedule quarterly review]
