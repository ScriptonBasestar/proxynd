# GitHub Workflow File - Manual Commit Required

## Status

The file `enterprise-load-test.yml` has been created but requires **manual commit** due to GitHub App permission restrictions.

## Why This Happened

GitHub Apps require special `workflows` permission to create or modify files in `.github/workflows/`. This is a security feature to prevent automated systems from modifying CI/CD pipelines without proper authorization.

## How to Complete the Commit

### Option 1: Manual Commit (Recommended)

```bash
# Add the workflow file
git add .github/workflows/enterprise-load-test.yml

# Commit with descriptive message
git commit -m "feat: Add GitHub Actions workflow for Enterprise API load testing

Automated load testing with performance regression detection.

Features:
- Quick load test for PRs (10s, 10 concurrent users)
- Standard load test for develop/master (60s, 100 concurrent users)
- Manual dispatch with scenario selection
- Performance thresholds and regression detection
- PR comments with test results
- Artifact upload (30-90 day retention)

Related: docs/deployment/ci-cd-load-testing.md"

# Push to remote
git push origin claude/add-openapi-swagger-docs-01RJC8rgFqKVeje7QSFsM9bz
```

### Option 2: Create New PR

If you prefer, you can create a separate Pull Request for the workflow file:

```bash
# Create new branch
git checkout -b add-enterprise-load-test-workflow

# Add and commit
git add .github/workflows/enterprise-load-test.yml
git commit -m "feat: Add Enterprise API load testing workflow"

# Push and create PR
git push origin add-enterprise-load-test-workflow
gh pr create --title "Add Enterprise API Load Testing Workflow" --body "Automated load testing integration"
```

### Option 3: Grant GitHub App Permissions

Contact your GitHub repository administrator to grant the GitHub App `workflows` permission.

## File Location

The workflow file is ready at:
```
.github/workflows/enterprise-load-test.yml
```

Documentation is already committed and pushed:
```
docs/deployment/ci-cd-load-testing.md  ✓ Pushed (commit 341199e)
```

## Verification

After committing the workflow file, verify it works:

1. Push a test commit to a PR that modifies Enterprise API files
2. Check GitHub Actions tab for the workflow execution
3. Review the PR comment with test results

## Related Documentation

- [CI/CD Load Testing Guide](ci-cd-load-testing.md)
- [Load Testing Scripts](../../scripts/loadtest/README.md)
- [Prometheus Metrics](../api/METRICS.md)
