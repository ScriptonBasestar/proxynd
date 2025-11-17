# Pending Manual Commits

This file tracks files that are ready but require manual commit due to permission restrictions.

## Files Requiring Manual Commit

### 1. GitHub Actions Workflow - Enterprise API Load Testing

**File**: `.github/workflows/enterprise-load-test.yml`
**Status**: ✅ Created, ⏳ Awaiting Manual Commit
**Reason**: GitHub App lacks `workflows` permission

**Documentation**: See [docs/deployment/workflow-manual-commit-guide.md](docs/deployment/workflow-manual-commit-guide.md) for complete instructions.

**Quick Commit**:
```bash
git add .github/workflows/enterprise-load-test.yml
git commit -m "feat: Add GitHub Actions workflow for Enterprise API load testing"
git push origin claude/add-openapi-swagger-docs-01RJC8rgFqKVeje7QSFsM9bz
```

**Related Documentation**:
- Workflow documentation: [docs/deployment/ci-cd-load-testing.md](docs/deployment/ci-cd-load-testing.md)
- Manual commit guide: [docs/deployment/workflow-manual-commit-guide.md](docs/deployment/workflow-manual-commit-guide.md)

---

## Why These Files Exist

Some files cannot be automatically committed by AI assistants due to security restrictions:
- **Workflow files** (`.github/workflows/*.yml`) - Require special `workflows` permission
- **Sensitive configuration** - Should be reviewed by humans before commit

These files are intentionally left untracked and should be committed manually after review.

## After Manual Commit

Once you've manually committed the pending files, you can delete this file:
```bash
git rm PENDING-MANUAL-COMMIT.md
git commit -m "chore: Remove pending manual commit tracker"
```
